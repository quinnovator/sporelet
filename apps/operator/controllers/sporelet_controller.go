package controllers

import (
    "context"
    "fmt"
    "os/exec"
    "path/filepath"

    fcoci "github.com/quinnovator/sporelet/packages/fc-snapshot-tools/pkg/oci"
    "github.com/quinnovator/sporelet/apps/operator/api/v1alpha1"
    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

type SporeletReconciler struct {
    client.Client
}

func (r *SporeletReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    var sp v1alpha1.Sporelet
    if err := r.Get(ctx, req.NamespacedName, &sp); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    if sp.Status.Phase == "Ready" {
        return ctrl.Result{}, nil
    }

    workDir := filepath.Join("/var/lib/sporelet", req.Namespace, req.Name)
    if err := fcoci.PullSnapshot(ctx, sp.Spec.Snapshot, workDir); err != nil {
        r.updatePhase(ctx, &sp, "Error")
        return ctrl.Result{}, fmt.Errorf("pull snapshot: %w", err)
    }

    cmd := exec.CommandContext(ctx, "spore-shim", "restore", workDir)
    output, err := cmd.CombinedOutput()
    if err != nil {
        r.updatePhase(ctx, &sp, "Error")
        return ctrl.Result{}, fmt.Errorf("restore failed: %s: %w", string(output), err)
    }

    r.updatePhase(ctx, &sp, "Ready")
    return ctrl.Result{}, nil
}

func (r *SporeletReconciler) updatePhase(ctx context.Context, sp *v1alpha1.Sporelet, phase string) {
    sp.Status.Phase = phase
    _ = r.Status().Update(ctx, sp)
}

func (r *SporeletReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&v1alpha1.Sporelet{}).
        Complete(r)
}
