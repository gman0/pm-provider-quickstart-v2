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

package wildwest

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	wildwestv1alpha1 "github.com/gman0/provider-quickstart-v2/apis/wildwest/v1alpha1"
)

// CowboyReconciler reconciles Cowboy resources in the manager's own cluster.
type CowboyReconciler struct {
	Client   client.Client
	Recorder record.EventRecorder
}

func (r *CowboyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.Recorder = mgr.GetEventRecorderFor("cowboy-controller")

	return ctrl.NewControllerManagedBy(mgr).
		Named("cowboy-controller").
		For(&wildwestv1alpha1.Cowboy{}).
		Complete(r)
}

func (r *CowboyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	cowboy := &wildwestv1alpha1.Cowboy{}
	if err := r.Client.Get(ctx, req.NamespacedName, cowboy); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("get cowboy: %w", err)
	}

	logger.Info("Reconciling Cowboy", "name", cowboy.Name, "intent", cowboy.Spec.Intent)

	if cowboy.Spec.Intent != "" && cowboy.Status.Result == "" {
		cowboy.Status.Result = fmt.Sprintf("Yeehaw! %s completed", cowboy.Spec.Intent)
		if err := r.Client.Status().Update(ctx, cowboy); err != nil {
			return reconcile.Result{}, fmt.Errorf("update cowboy status: %w", err)
		}
		logger.Info("Updated Cowboy status", "result", cowboy.Status.Result)
	}

	r.Recorder.Eventf(cowboy, corev1.EventTypeNormal, "Reconciled", "Cowboy %s reconciled", cowboy.Name)
	return reconcile.Result{}, nil
}
