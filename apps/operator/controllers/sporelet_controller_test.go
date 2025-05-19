package controllers

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	v1alpha1 "github.com/quinnovator/sporelet/apps/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/types"
)

func TestReconcileCreate(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	sp := &v1alpha1.Sporelet{
		ObjectMeta: metav1.ObjectMeta{Name: "sp", Namespace: "ns"},
		Spec:       v1alpha1.SporeletSpec{Snapshot: "ref"},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sp).Build()
	r := &SporeletReconciler{Client: c}

	dir := t.TempDir()
	baseWorkDir = dir
	pullCalled := false
	pullSnapshotFn = func(ctx context.Context, ociRef, outDir string) error {
		pullCalled = true
		return os.MkdirAll(outDir, 0755)
	}
	execCalled := false
	execCommandCtx = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		execCalled = true
		return exec.CommandContext(ctx, "true")
	}
	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "sp"}})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	var out v1alpha1.Sporelet
	_ = c.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "sp"}, &out)
	if out.Status.Phase != v1alpha1.PhaseReady {
		t.Fatalf("phase %s", out.Status.Phase)
	}
	if !pullCalled || !execCalled {
		t.Fatalf("expected pull and exec to be called")
	}
	if !containsString(out.Finalizers, v1alpha1.SporeletFinalizer) {
		t.Fatalf("finalizer missing")
	}
}

func TestReconcileDelete(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	tdir := t.TempDir()
	baseWorkDir = tdir
	workDir := filepath.Join(tdir, "ns", "sp")
	os.MkdirAll(workDir, 0755)

	now := metav1.NewTime(time.Now())
	sp := &v1alpha1.Sporelet{
		ObjectMeta: metav1.ObjectMeta{Name: "sp", Namespace: "ns", Finalizers: []string{v1alpha1.SporeletFinalizer}, DeletionTimestamp: &now},
		Status:     v1alpha1.SporeletStatus{Phase: v1alpha1.PhaseReady},
		Spec:       v1alpha1.SporeletSpec{Snapshot: "ref"},
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(sp).Build()
	r := &SporeletReconciler{Client: c}

	killed := false
	execCommand = func(name string, args ...string) *exec.Cmd {
		killed = true
		return exec.Command("true")
	}

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Namespace: "ns", Name: "sp"}})
	if err != nil {
		t.Fatalf("reconcile delete: %v", err)
	}
	var out v1alpha1.Sporelet
	_ = c.Get(context.Background(), types.NamespacedName{Namespace: "ns", Name: "sp"}, &out)
	if out.Status.Phase != v1alpha1.PhaseStopped {
		t.Fatalf("phase %s", out.Status.Phase)
	}
	if killed == false {
		t.Fatalf("expected kill to be called")
	}
	if containsString(out.Finalizers, v1alpha1.SporeletFinalizer) {
		t.Fatalf("finalizer not removed")
	}
	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Fatalf("expected workdir removed")
	}
}
