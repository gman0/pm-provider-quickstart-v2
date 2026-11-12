/*
Copyright The Platform Mesh Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package bootstrap

import (
	"bufio"
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeyaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"

	configbootstrap "github.com/gman0/provider-quickstart-v2/config/bootstrap"
)

// Bootstrap upserts all embedded bootstrap resources into the workspace
// pointed at by config.
func Bootstrap(ctx context.Context, config *rest.Config) error {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	cache := memory.NewMemCacheClient(discoveryClient)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(cache)

	logger := klog.FromContext(ctx)
	logger.Info("Bootstrapping provider resources")
	if err := bootstrapFS(ctx, dynamicClient, mapper, cache, configbootstrap.Resources); err != nil {
		return fmt.Errorf("failed to bootstrap provider resources: %w", err)
	}
	return nil
}

func bootstrapFS(ctx context.Context, dynamicClient dynamic.Interface, mapper meta.RESTMapper, cache discovery.CachedDiscoveryInterface, fs embed.FS) error {
	logger := klog.FromContext(ctx)
	var lastErr error
	attempt := 0
	err := wait.PollUntilContextCancel(ctx, time.Second, true, func(ctx context.Context) (bool, error) {
		attempt++
		logger.Info("bootstrap attempt", "attempt", attempt)
		if err := createResourcesFromFS(ctx, dynamicClient, mapper, fs); err != nil {
			logger.Info("failed to bootstrap resources, retrying", "attempt", attempt, "error", err)
			lastErr = err
			cache.Invalidate()
			return false, nil
		}
		logger.Info("bootstrap succeeded", "attempt", attempt)
		return true, nil
	})
	if err != nil && lastErr != nil {
		return fmt.Errorf("%w: %v", err, lastErr)
	}
	return err
}

func createResourcesFromFS(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, fs embed.FS) error {
	logger := klog.FromContext(ctx)
	files, err := fs.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read embedded filesystem: %w", err)
	}

	var errs []error
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		logger.Info("processing file", "filename", name)
		if err := createResourceFromFS(ctx, client, mapper, name, fs); err != nil {
			errs = append(errs, err)
		}
	}
	return utilerrors.NewAggregate(errs)
}

func createResourceFromFS(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, filename string, fs embed.FS) error {
	logger := klog.FromContext(ctx)
	raw, err := fs.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("could not read %s: %w", filename, err)
	}
	if len(raw) == 0 {
		logger.Info("skipping empty file", "filename", filename)
		return nil
	}

	d := kubeyaml.NewYAMLReader(bufio.NewReader(bytes.NewReader(raw)))
	var errs []error
	for i := 1; ; i++ {
		doc, err := d.Read()
		if errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			return fmt.Errorf("failed to read YAML document %d from %s: %w", i, filename, err)
		}
		if len(bytes.TrimSpace(doc)) == 0 {
			continue
		}
		if err := createResource(ctx, client, mapper, doc); err != nil {
			errs = append(errs, fmt.Errorf("failed to create resource from %s doc %d: %w", filename, i, err))
		}
	}
	return utilerrors.NewAggregate(errs)
}

func createResource(ctx context.Context, client dynamic.Interface, mapper meta.RESTMapper, raw []byte) error {
	logger := klog.FromContext(ctx)

	u := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(raw, &u.Object); err != nil {
		return fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	gvk := u.GroupVersionKind()
	if gvk.Kind == "" {
		return fmt.Errorf("missing kind in resource")
	}

	logger = logger.WithValues("kind", gvk.Kind, "name", u.GetName(), "namespace", u.GetNamespace())

	m, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("failed to get REST mapping for %s: %w", gvk, err)
	}

	resourceClient := client.Resource(m.Resource).Namespace(u.GetNamespace())
	_, err = resourceClient.Create(ctx, u, metav1.CreateOptions{})
	if err == nil {
		logger.Info("created resource", "resource", m.Resource.String())
		return nil
	}
	if !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create %s %s: %w", gvk.Kind, u.GetName(), err)
	}

	existing, getErr := resourceClient.Get(ctx, u.GetName(), metav1.GetOptions{})
	if getErr != nil {
		return fmt.Errorf("failed to get existing %s %s: %w", gvk.Kind, u.GetName(), getErr)
	}
	u.SetResourceVersion(existing.GetResourceVersion())
	if _, err := resourceClient.Update(ctx, u, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("failed to update %s %s: %w", gvk.Kind, u.GetName(), err)
	}
	logger.Info("updated resource")
	return nil
}
