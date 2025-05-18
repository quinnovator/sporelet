package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	fcclient "github.com/quinnovator/sporelet/packages/fc-snapshot-tools/pkg/firecracker"
)

// TestSnapshotBuild runs the snapshot-builder script to produce a snapshot.
func TestSnapshotBuild(t *testing.T) {
	if _, err := exec.LookPath("firecracker"); err != nil {
		t.Skip("firecracker binary not available")
	}

	snapDir := filepath.Join("dist", "e2e")
	cmd := exec.Command("bash", filepath.Join("apps", "snapshot-builder", "build.sh"))
	cmd.Env = append(os.Environ(), "SNAP_DIR="+snapDir, "OCI_REF=test/example:latest")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("snapshot build failed: %v\n%s", err, out)
	}

	files := []string{"layer1.mem", "layer1.vmstate", "layer1.config"}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(snapDir, f)); err != nil {
			t.Fatalf("expected %s to exist: %v", f, err)
		}
	}
}

// TestBootMicroVMFromSnapshot attempts to boot the microVM from the built snapshot.
// TODO: implement once a restore utility is available.
func TestBootMicroVMFromSnapshot(t *testing.T) {
	if _, err := exec.LookPath("firecracker"); err != nil {
		t.Skip("firecracker binary not available")
	}

	snapDir := filepath.Join("dist", "e2e")
	files := []string{"layer1.mem", "layer1.vmstate", "layer1.config"}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(snapDir, f)); err != nil {
			t.Skipf("snapshot file missing: %v", err)
		}
	}

	socket := filepath.Join(t.TempDir(), "fc.sock")
	vmID := "e2e-test"
	cmd := exec.Command("go", "run", "./cmd/spore-shim", "restore",
		"--socket-path", socket,
		"--id", vmID,
		snapDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("restore failed: %v\n%s", err, out)
	}

	client, err := fcclient.NewClient("", "", vmID, socket,
		fcclient.WithStartFunc(func(context.Context) error { return nil }))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.WaitForVSockHandshake(ctx); err != nil {
		t.Fatalf("handshake failed: %v", err)
	}

	exec.Command("pkill", "-f", fmt.Sprintf("--id %s", vmID)).Run()
}
