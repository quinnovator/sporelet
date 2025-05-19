package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	fc "github.com/quinnovator/sporelet/packages/fc-snapshot-tools"
)

func TestRunDiffMissingArgs(t *testing.T) {
	if err := runDiff([]string{}); err == nil {
		t.Fatal("expected error for missing args")
	}
}

func TestRunDiffDetectsChanges(t *testing.T) {
	base := t.TempDir()
	out := t.TempDir()

	// base snapshot files
	os.WriteFile(filepath.Join(base, "layer.mem"), []byte("same"), 0644)
	os.WriteFile(filepath.Join(base, "layer.vmstate"), []byte("old"), 0644)
	os.WriteFile(filepath.Join(base, "layer.config"), []byte("same"), 0644)

	// stub snapshot creation
	startAndSnapshot = func(ctx context.Context, spec fc.SnapshotSpec, dir string) error {
		os.WriteFile(filepath.Join(dir, "snapshot.mem"), []byte("same"), 0644)
		os.WriteFile(filepath.Join(dir, "snapshot.vmstate"), []byte("new"), 0644)
		os.WriteFile(filepath.Join(dir, "snapshot.config"), []byte("same"), 0644)
		return nil
	}
	defer func() { startAndSnapshot = fc.StartAndSnapshot }()

	// capture output
	r, w, _ := os.Pipe()
	old := os.Stdout
	os.Stdout = w
	err := runDiff([]string{
		"--base-dir", base,
		"--kernel", "k",
		"--rootfs", "r",
		"--out-dir", out,
		"--snapshot-prefix", "layer",
	})
	w.Close()
	outBytes, _ := io.ReadAll(r)
	os.Stdout = old

	if err != nil {
		t.Fatalf("runDiff failed: %v", err)
	}
	if !contains(string(outBytes), "layer.vmstate") {
		t.Fatalf("expected changed file in output: %s", outBytes)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }
