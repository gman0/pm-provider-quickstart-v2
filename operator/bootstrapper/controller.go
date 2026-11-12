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

package bootstrapper

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlbuilder "sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	ctrlmanager "sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gman0/provider-quickstart-v2/pkg/bootstrap"
)

// BootstrapReconciler watches kubeconfig Secrets and runs Bootstrap
// against each one.
type BootstrapReconciler struct {
	Manager ctrlmanager.Manager
	// SecretSelector, if non-nil, restricts reconciliation to Secrets
	// whose labels match it. nil means all Secrets are reconciled.
	SecretSelector labels.Selector
	// KubeconfigKey is the key in the Secret data that holds the
	// kubeconfig. Defaults to "kubeconfig".
	KubeconfigKey string
}

func (r *BootstrapReconciler) SetupWithManager(mgr ctrlmanager.Manager) error {
	r.Manager = mgr
	if r.KubeconfigKey == "" {
		r.KubeconfigKey = "kubeconfig"
	}

	b := ctrlbuilder.ControllerManagedBy(mgr).
		Named("bootstrapper").
		For(&corev1.Secret{})

	if r.SecretSelector != nil {
		b = b.WithEventFilter(predicate.NewPredicateFuncs(func(obj client.Object) bool {
			return r.SecretSelector.Matches(labels.Set(obj.GetLabels()))
		}))
	}

	return b.Complete(reconcile.Func(r.Reconcile))
}

func (r *BootstrapReconciler) Reconcile(ctx context.Context, req reconcile.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues("secret", req.NamespacedName)
	c := r.Manager.GetClient()

	secret := &corev1.Secret{}
	if err := c.Get(ctx, req.NamespacedName, secret); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("get secret: %w", err)
	}
	if !secret.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}

	raw, ok := secret.Data[r.KubeconfigKey]
	if !ok || len(raw) == 0 {
		logger.Info("skipping Secret with no kubeconfig data", "key", r.KubeconfigKey)
		return reconcile.Result{}, nil
	}

	cfg, err := clientcmd.RESTConfigFromKubeConfig(raw)
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("parse kubeconfig from secret: %w", err)
	}

	logger.Info("Bootstrapping provider resources", "host", cfg.Host)
	if err := bootstrap.Bootstrap(ctx, cfg); err != nil {
		return reconcile.Result{}, fmt.Errorf("bootstrap: %w", err)
	}
	logger.Info("Bootstrap completed")
	return reconcile.Result{}, nil
}
