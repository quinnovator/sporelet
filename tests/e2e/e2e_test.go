package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
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
	t.Skip("boot from snapshot not implemented")
}
