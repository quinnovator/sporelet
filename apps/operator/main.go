package main

import (
    "flag"

    "github.com/quinnovator/sporelet/apps/operator/api/v1alpha1"
    "github.com/quinnovator/sporelet/apps/operator/controllers"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client/config"
    "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
    var metricsAddr string
    flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
    flag.Parse()

    ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

    mgr, err := ctrl.NewManager(config.GetConfigOrDie(), ctrl.Options{
        Scheme:             ctrl.NewScheme(),
        MetricsBindAddress: metricsAddr,
    })
    if err != nil {
        panic(err)
    }

    if err := v1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
        panic(err)
    }

    if err := (&controllers.SporeletReconciler{Client: mgr.GetClient()}).SetupWithManager(mgr); err != nil {
        panic(err)
    }

    if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
        panic(err)
    }
}
