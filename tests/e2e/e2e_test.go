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

// TestBootMicroVMFromSnapshot attempts to boot the microVM from the built snapshot
// and verifies the guest agent handshake succeeds.
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.WaitForVSockHandshake(ctx); err != nil {
		t.Fatalf("handshake failed: %v", err)
	}

	exec.Command("pkill", "-f", fmt.Sprintf("--id %s", vmID)).Run()
}

// TestSnapshotDiff builds a base snapshot, modifies the rootfs, creates a diff
// layer using sporectl and verifies the change after restore.
func TestSnapshotDiff(t *testing.T) {
	if _, err := exec.LookPath("firecracker"); err != nil {
		t.Skip("firecracker binary not available")
	}

	// Prepare snapshot directory and SSH key for root login
	snapDir := t.TempDir()
	keyDir := filepath.Join(snapDir, "ssh")
	if err := os.Mkdir(keyDir, 0700); err != nil {
		t.Fatalf("mkdir ssh key dir: %v", err)
	}
	privKey := filepath.Join(keyDir, "id_rsa")
	pubKey := privKey + ".pub"
	if out, err := exec.Command("ssh-keygen", "-t", "rsa", "-N", "", "-f", privKey).CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen failed: %v\n%s", err, out)
	}

	// Build the base (layer1) snapshot
	buildCmd := exec.Command("bash", filepath.Join("apps", "snapshot-builder", "build.sh"))
	buildCmd.Env = append(os.Environ(), "SNAP_DIR="+snapDir, "OCI_REF=test/example:latest", "PUBKEY_FILE="+pubKey)
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("snapshot build failed: %v\n%s", err, out)
	}

	// Rename layer1 files for restore
	os.Rename(filepath.Join(snapDir, "layer1.mem"), filepath.Join(snapDir, "snapshot.mem"))
	os.Rename(filepath.Join(snapDir, "layer1.vmstate"), filepath.Join(snapDir, "snapshot.vmstate"))
	os.Rename(filepath.Join(snapDir, "layer1.config"), filepath.Join(snapDir, "snapshot.config"))

	// Boot the microVM from the layer1 snapshot
	socket := filepath.Join(t.TempDir(), "fc.sock")
	vmID := "e2e-diff"
	restoreCmd := exec.Command("go", "run", "./cmd/spore-shim", "restore",
		"--socket-path", socket,
		"--id", vmID,
		snapDir)
	if out, err := restoreCmd.CombinedOutput(); err != nil {
		t.Fatalf("restore failed: %v\n%s", err, out)
	}

	client, err := fcclient.NewClient("", "", vmID, socket,
		fcclient.WithStartFunc(func(context.Context) error { return nil }))
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.WaitForVSockHandshake(ctx); err != nil {
		t.Fatalf("handshake failed: %v", err)
	}

	// Modify a file inside the VM via SSH
	sshCmd := exec.Command("ssh", "-i", privKey, "-o", "StrictHostKeyChecking=no", "root@172.16.0.2", "echo hello > /root/layer2.txt")
	if out, err := sshCmd.CombinedOutput(); err != nil {
		exec.Command("pkill", "-f", fmt.Sprintf("--id %s", vmID)).Run()
		t.Fatalf("ssh modify failed: %v\n%s", err, out)
	}

	// Stop the VM
	exec.Command("pkill", "-f", fmt.Sprintf("--id %s", vmID)).Run()

	// Create a diff snapshot (layer2)
	diffCmd := exec.Command("go", "run", "./apps/sporectl", "diff",
		"--base-dir", snapDir,
		"--kernel", filepath.Join(snapDir, "vmlinux"),
		"--rootfs", filepath.Join(snapDir, "rootfs.ext4"),
		"--out-dir", snapDir,
		"--snapshot-prefix", "layer2")
	if out, err := diffCmd.CombinedOutput(); err != nil {
		t.Fatalf("snapshot diff failed: %v\n%s", err, out)
	}

	files := []string{"layer2.mem", "layer2.vmstate", "layer2.config"}
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(snapDir, f)); err != nil {
			t.Fatalf("expected %s to exist: %v", f, err)
		}
	}

	// Replace snapshot files with layer2 for restore
	os.Rename(filepath.Join(snapDir, "layer2.mem"), filepath.Join(snapDir, "snapshot.mem"))
	os.Rename(filepath.Join(snapDir, "layer2.vmstate"), filepath.Join(snapDir, "snapshot.vmstate"))
	os.Rename(filepath.Join(snapDir, "layer2.config"), filepath.Join(snapDir, "snapshot.config"))

	// Restore again and verify the change exists
	socket2 := filepath.Join(t.TempDir(), "fc.sock")
	vmID2 := "e2e-diff2"
	restore2 := exec.Command("go", "run", "./cmd/spore-shim", "restore",
		"--socket-path", socket2,
		"--id", vmID2,
		snapDir)
	if out, err := restore2.CombinedOutput(); err != nil {
		t.Fatalf("restore layer2 failed: %v\n%s", err, out)
	}

	client2, err := fcclient.NewClient("", "", vmID2, socket2,
		fcclient.WithStartFunc(func(context.Context) error { return nil }))
	if err != nil {
		t.Fatalf("client2 create failed: %v", err)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	if err := client2.WaitForVSockHandshake(ctx2); err != nil {
		t.Fatalf("handshake layer2 failed: %v", err)
	}

	catCmd := exec.Command("ssh", "-i", privKey, "-o", "StrictHostKeyChecking=no", "root@172.16.0.2", "cat /root/layer2.txt")
	out, err := catCmd.CombinedOutput()
	exec.Command("pkill", "-f", fmt.Sprintf("--id %s", vmID2)).Run()
	if err != nil {
		t.Fatalf("ssh check failed: %v\n%s", err, out)
	}
	if string(out) == "" {
		t.Fatalf("expected layer2.txt contents, got empty output")
	}
}
