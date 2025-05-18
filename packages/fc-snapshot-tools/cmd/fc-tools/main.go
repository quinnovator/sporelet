// Command fc-tools provides a CLI for working with Firecracker snapshots
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/quinnovator/sporelet/packages/fc-snapshot-tools"
)

func main() {
	// Define command-line flags
	var (
		kernelPath   = flag.String("kernel", "", "Path to kernel image")
		rootfsPath   = flag.String("rootfs", "", "Path to rootfs image")
		cmdline      = flag.String("cmdline", "console=ttyS0 reboot=k panic=1 pci=off", "Kernel command line")
		hostDev      = flag.String("host-dev", "tap0", "Host network device name")
		macAddr      = flag.String("mac", "AA:FC:00:00:00:01", "MAC address for the guest")
		ipAddr       = flag.String("ip", "172.16.0.2", "IP address for the guest")
		netmask      = flag.String("netmask", "255.255.255.0", "Network mask")
		gateway      = flag.String("gateway", "172.16.0.1", "Gateway IP address")
		memSize      = flag.Int("mem", 1024, "Memory size in MB")
		vcpuCount    = flag.Int("vcpu", 1, "Number of vCPUs")
		outDir       = flag.String("out-dir", ".", "Output directory for snapshot files")
		snapshotPrefix = flag.String("snapshot-prefix", "snapshot", "Prefix for snapshot files")
		ociRef       = flag.String("oci-ref", "", "OCI reference for pushing snapshot")
		push         = flag.Bool("push", false, "Push snapshot to OCI registry")
	)

	// Define subcommands
	snapshotCmd := flag.NewFlagSet("snapshot", flag.ExitOnError)
	pushCmd := flag.NewFlagSet("push", flag.ExitOnError)

	// Parse flags
	flag.Parse()

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Process subcommands
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "snapshot":
		snapshotCmd.Parse(os.Args[2:])
		if err := runSnapshot(ctx, *kernelPath, *rootfsPath, *cmdline, *hostDev, *macAddr, *ipAddr, *netmask, *gateway, *memSize, *vcpuCount, *outDir, *snapshotPrefix, *ociRef, *push); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "push":
		pushCmd.Parse(os.Args[2:])
		if err := runPush(ctx, *outDir, *snapshotPrefix, *ociRef); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: fc-tools <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  snapshot    Create a snapshot from a Firecracker VM")
	fmt.Println("  push        Push a snapshot to an OCI registry")
	fmt.Println("\nOptions:")
	flag.PrintDefaults()
}

func runSnapshot(ctx context.Context, kernelPath, rootfsPath, cmdline, hostDev, macAddr, ipAddr, netmask, gateway string, memSize, vcpuCount int, outDir, snapshotPrefix, ociRef string, push bool) error {
	// Validate required parameters
	if kernelPath == "" {
		return fmt.Errorf("kernel path is required")
	}
	if rootfsPath == "" {
		return fmt.Errorf("rootfs path is required")
	}

	// Create snapshot spec
	spec := fc.SnapshotSpec{
		Kernel:  kernelPath,
		Rootfs:  rootfsPath,
		Cmdline: cmdline,
		Net: fc.NetConfig{
			HostDevName: hostDev,
			MacAddr:     macAddr,
			IPAddr:      ipAddr,
			Mask:        netmask,
			Gateway:     gateway,
		},
		MemSizeMB: memSize,
		VCPUCount: vcpuCount,
	}

	// Create output directory
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Start VM and create snapshot
	fmt.Println("Starting VM and creating snapshot...")
	if err := fc.StartAndSnapshot(ctx, spec, outDir); err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	// Rename snapshot files with the specified prefix
	if snapshotPrefix != "snapshot" {
		files := []string{"snapshot.mem", "snapshot.vmstate", "snapshot.config"}
		for _, file := range files {
			oldPath := filepath.Join(outDir, file)
			newPath := filepath.Join(outDir, strings.Replace(file, "snapshot", snapshotPrefix, 1))
			if err := os.Rename(oldPath, newPath); err != nil {
				return fmt.Errorf("failed to rename snapshot file: %w", err)
			}
		}
	}

	fmt.Printf("Snapshot created in %s\n", outDir)

	// Push snapshot to OCI registry if requested
	if push && ociRef != "" {
		return runPush(ctx, outDir, snapshotPrefix, ociRef)
	}

	return nil
}

func runPush(ctx context.Context, outDir, snapshotPrefix, ociRef string) error {
	// Validate required parameters
	if ociRef == "" {
		return fmt.Errorf("OCI reference is required")
	}

	// Get snapshot file paths
	memFile := filepath.Join(outDir, fmt.Sprintf("%s.mem", snapshotPrefix))
	vmstateFile := filepath.Join(outDir, fmt.Sprintf("%s.vmstate", snapshotPrefix))
	configFile := filepath.Join(outDir, fmt.Sprintf("%s.config", snapshotPrefix))

	// Push snapshot to OCI registry
	fmt.Printf("Pushing snapshot to %s...\n", ociRef)
	if err := fc.PushSnapshot(ctx, ociRef, memFile, vmstateFile, configFile); err != nil {
		return fmt.Errorf("failed to push snapshot: %w", err)
	}

	fmt.Printf("Snapshot pushed to %s\n", ociRef)
	return nil
}
