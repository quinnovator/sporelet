// Package fc provides helpers to talk to the Firecracker REST API
// and to bundle snapshots as OCI artifacts.
package fc

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/quinnovator/sporelet/packages/fc-snapshot-tools/pkg/firecracker"
	"github.com/quinnovator/sporelet/packages/fc-snapshot-tools/pkg/oci"
)

// NetConfig defines the network configuration for a Firecracker VM
type NetConfig struct {
	HostDevName string // Host network device name
	MacAddr     string // MAC address for the guest
	IPAddr      string // IP address for the guest
	Mask        string // Network mask
	Gateway     string // Gateway IP address
}

// SnapshotSpec defines the configuration for creating a VM snapshot
type SnapshotSpec struct {
	Kernel     string    // Path to the kernel image
	Rootfs     string    // Path to the rootfs image
	Cmdline    string    // Kernel command line
	Net        NetConfig // Network configuration
	MemSizeMB  int       // Memory size in MB (default: 1024)
	VCPUCount  int       // Number of vCPUs (default: 1)
	JailerBin  string    // Path to the jailer binary (default: "jailer")
	FCBin      string    // Path to the firecracker binary (default: "firecracker")
	SocketPath string    // Path to the Firecracker socket (default: auto-generated)
	ID         string    // VM ID (default: auto-generated)
}

// StartAndSnapshot launches a Firecracker VM with the given configuration,
// waits for it to be ready, and then creates a snapshot.
// The snapshot files (.mem, .vmstate, .config) are written to the outDir.
func StartAndSnapshot(ctx context.Context, s SnapshotSpec, outDir string) error {
	// Set defaults
	if s.MemSizeMB == 0 {
		s.MemSizeMB = 1024
	}
	if s.VCPUCount == 0 {
		s.VCPUCount = 1
	}
	if s.JailerBin == "" {
		s.JailerBin = "jailer"
	}
	if s.FCBin == "" {
		s.FCBin = "firecracker"
	}
	if s.ID == "" {
		s.ID = fmt.Sprintf("sporelet-%d", time.Now().Unix())
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create Firecracker client
	client, err := firecracker.NewClient(s.FCBin, s.JailerBin, s.ID, s.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to create Firecracker client: %w", err)
	}

	// Start the VM
	vmConfig := firecracker.VMConfig{
		KernelImagePath: s.Kernel,
		RootDrive: firecracker.Drive{
			PathOnHost:   s.Rootfs,
			IsReadOnly:   false,
			IsRootDevice: true,
		},
		KernelArgs: s.Cmdline,
		MemSizeMB:  s.MemSizeMB,
		VCPUCount:  s.VCPUCount,
		NetworkInterfaces: []firecracker.NetworkInterface{
			{
				HostDevName: s.Net.HostDevName,
				MacAddress:  s.Net.MacAddr,
				IPAddress:   s.Net.IPAddr,
				Netmask:     s.Net.Mask,
				Gateway:     s.Net.Gateway,
			},
		},
	}

	if err := client.StartVM(ctx, vmConfig); err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	// Wait for vsock handshake to complete
	if err := client.WaitForVSockHandshake(ctx); err != nil {
		return fmt.Errorf("vsock handshake failed: %w", err)
	}

	// Create snapshot
	snapshotConfig := firecracker.SnapshotConfig{
		MemFilePath:     filepath.Join(outDir, "snapshot.mem"),
		VMStateFilePath: filepath.Join(outDir, "snapshot.vmstate"),
		ConfigFilePath:  filepath.Join(outDir, "snapshot.config"),
	}

	if err := client.CreateSnapshot(ctx, snapshotConfig); err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	return nil
}

// PushSnapshot pushes a snapshot to an OCI registry
func PushSnapshot(ctx context.Context, ociRef, memFile, vmstateFile, configFile string) error {
	// Check if files exist
	for _, file := range []string{memFile, vmstateFile, configFile} {
		if _, err := os.Stat(file); err != nil {
			return fmt.Errorf("snapshot file not found: %s: %w", file, err)
		}
	}

	// Push to OCI registry
	return oci.PushSnapshot(ctx, ociRef, memFile, vmstateFile, configFile)
}

// RestoreSpec defines the configuration for restoring a VM from a snapshot
// directory. MemFile, VMStateFile and ConfigFile must point to the snapshot
// files produced by StartAndSnapshot.
type RestoreSpec struct {
	MemFile     string
	VMStateFile string
	ConfigFile  string
	JailerBin   string // Path to jailer binary (default "jailer")
	FCBin       string // Path to firecracker binary (default "firecracker")
	SocketPath  string // Optional socket path
	ID          string // Optional VM ID
}

// Restore launches Firecracker and loads the given snapshot to resume the VM.
func Restore(ctx context.Context, s RestoreSpec) error {
	if s.JailerBin == "" {
		s.JailerBin = "jailer"
	}
	if s.FCBin == "" {
		s.FCBin = "firecracker"
	}
	if s.ID == "" {
		s.ID = fmt.Sprintf("sporelet-%d", time.Now().Unix())
	}

	for _, f := range []string{s.MemFile, s.VMStateFile, s.ConfigFile} {
		if _, err := os.Stat(f); err != nil {
			return fmt.Errorf("snapshot file not found: %s: %w", f, err)
		}
	}

	client, err := firecracker.NewClient(s.FCBin, s.JailerBin, s.ID, s.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to create Firecracker client: %w", err)
	}

	rcfg := firecracker.RestoreConfig{
		MemFilePath:     s.MemFile,
		VMStateFilePath: s.VMStateFile,
		ConfigFilePath:  s.ConfigFile,
	}
	if err := client.RestoreSnapshot(ctx, rcfg); err != nil {
		return fmt.Errorf("failed to restore snapshot: %w", err)
	}

	// Wait for guest agent readiness
	if err := client.WaitForVSockHandshake(ctx); err != nil {
		return fmt.Errorf("vsock handshake failed: %w", err)
	}

	return nil
}

// CompareSnapshotDirs compares snapshot files in baseDir and newDir using the
// provided snapshot prefix. It returns a slice of file names that differ.
func CompareSnapshotDirs(baseDir, newDir, prefix string) ([]string, error) {
	files := []string{
		fmt.Sprintf("%s.mem", prefix),
		fmt.Sprintf("%s.vmstate", prefix),
		fmt.Sprintf("%s.config", prefix),
	}

	var changed []string
	for _, f := range files {
		basePath := filepath.Join(baseDir, f)
		newPath := filepath.Join(newDir, f)

		bHash, err := fileHash(basePath)
		if err != nil {
			return nil, err
		}
		nHash, err := fileHash(newPath)
		if err != nil {
			return nil, err
		}

		if bHash != nHash {
			changed = append(changed, f)
		}
	}

	return changed, nil
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
