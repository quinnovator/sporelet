package controllers

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/quinnovator/sporelet/apps/operator/api/v1alpha1"
	fcoci "github.com/quinnovator/sporelet/packages/fc-snapshot-tools/pkg/oci"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type SporeletReconciler struct {
	client.Client
}

func (r *SporeletReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var sp v1alpha1.Sporelet
	if err := r.Get(ctx, req.NamespacedName, &sp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if sp.Status.Phase == v1alpha1.PhaseReady && sp.Status.Snapshot == sp.Spec.Snapshot {
		return ctrl.Result{}, nil
	}

	r.updateStatus(ctx, &sp, v1alpha1.PhasePending, metav1.Condition{})

	workDir := filepath.Join("/var/lib/sporelet", req.Namespace, req.Name)
	if err := fcoci.PullSnapshot(ctx, sp.Spec.Snapshot, workDir); err != nil {
		cond := metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Reason: "PullFailed", Message: err.Error(), LastTransitionTime: metav1.Now()}
		r.updateStatus(ctx, &sp, v1alpha1.PhaseError, cond)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	r.updateStatus(ctx, &sp, v1alpha1.PhaseRestoring, metav1.Condition{})

	cmd := exec.CommandContext(ctx, "/spore-shim", "restore", workDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		cond := metav1.Condition{Type: "Ready", Status: metav1.ConditionFalse, Reason: "RestoreFailed", Message: fmt.Sprintf("%s: %v", string(output), err), LastTransitionTime: metav1.Now()}
		r.updateStatus(ctx, &sp, v1alpha1.PhaseError, cond)
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	cond := metav1.Condition{Type: "Ready", Status: metav1.ConditionTrue, Reason: "Restored", Message: "snapshot restored", LastTransitionTime: metav1.Now()}
	sp.Status.Snapshot = sp.Spec.Snapshot
	r.updateStatus(ctx, &sp, v1alpha1.PhaseReady, cond)
	return ctrl.Result{}, nil
}

func (r *SporeletReconciler) updateStatus(ctx context.Context, sp *v1alpha1.Sporelet, phase string, cond metav1.Condition) {
	sp.Status.Phase = phase
	if cond.Type != "" {
		meta.SetStatusCondition(&sp.Status.Conditions, cond)
	}
	_ = r.Status().Update(ctx, sp)
}

func (r *SporeletReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Sporelet{}, ctrl.WithEventFilter(predicate.GenerationChangedPredicate{})).
		Complete(r)
}
