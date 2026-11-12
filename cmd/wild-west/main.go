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

package main

import (
	"os"

	"github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	wildwestv1alpha1 "github.com/gman0/provider-quickstart-v2/apis/wildwest/v1alpha1"
	wildwest "github.com/gman0/provider-quickstart-v2/operator/wild-west"
)

func init() {
	runtime.Must(wildwestv1alpha1.AddToScheme(scheme.Scheme))
}

func main() {
	log.SetLogger(zap.New(zap.UseDevMode(true)))

	var (
		metricsAddr          string
		probeAddr            string
		enableLeaderElection bool
		leaderElectionID     string
		leaderElectionNS     string
	)
	pflag.StringVar(&metricsAddr, "metrics-bind-address", ":9080", "Address the metric endpoint binds to.")
	pflag.StringVar(&probeAddr, "health-probe-bind-address", ":8080", "Address the probe endpoint binds to.")
	pflag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election to ensure only one active controller instance.")
	pflag.StringVar(&leaderElectionID, "leader-election-id", "wild-west-operator", "Leader election resource lock name.")
	pflag.StringVar(&leaderElectionNS, "leader-election-namespace", "", "Namespace for the leader election lease.")
	pflag.Parse()

	ctx := signals.SetupSignalHandler()
	entryLog := log.Log.WithName("entrypoint")

	mgr, err := manager.New(ctrl.GetConfigOrDie(), manager.Options{
		Scheme:                  scheme.Scheme,
		HealthProbeBindAddress:  probeAddr,
		Metrics:                 metricsserver.Options{BindAddress: metricsAddr},
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        leaderElectionID,
		LeaderElectionNamespace: leaderElectionNS,
	})
	if err != nil {
		entryLog.Error(err, "unable to set up manager")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		entryLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		entryLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if err := (&wildwest.CowboyReconciler{}).SetupWithManager(mgr); err != nil {
		entryLog.Error(err, "failed to setup cowboy controller")
		os.Exit(1)
	}

	entryLog.Info("Starting manager")
	if err := mgr.Start(ctx); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}
}
