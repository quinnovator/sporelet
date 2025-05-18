package oci

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPushSnapshot_NoOras(t *testing.T) {
	dir := t.TempDir()
	mem := filepath.Join(dir, "snapshot.mem")
	vm := filepath.Join(dir, "snapshot.vmstate")
	cfg := filepath.Join(dir, "snapshot.config")
	for _, f := range []string{mem, vm, cfg} {
		if err := os.WriteFile(f, []byte("dummy"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	if err := PushSnapshot(context.Background(), "test", mem, vm, cfg); err == nil {
		t.Fatal("expected error when oras is missing")
	}
}

func TestPushSnapshot_FileMissing(t *testing.T) {
	dir := t.TempDir()
	mem := filepath.Join(dir, "snapshot.mem")
	os.WriteFile(mem, []byte("dummy"), 0644)

	vm := filepath.Join(dir, "snapshot.vmstate")
	cfg := filepath.Join(dir, "snapshot.config")

	if err := PushSnapshot(context.Background(), "test", mem, vm, cfg); err == nil {
		t.Fatal("expected error for missing files")
	}
}

func TestPullSnapshot_NoOras(t *testing.T) {
	dir := t.TempDir()
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	if err := PullSnapshot(context.Background(), "test", dir); err == nil {
		t.Fatal("expected error when oras is missing")
	}
}
