package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	fc "github.com/quinnovator/sporelet/packages/fc-snapshot-tools"
	"github.com/quinnovator/sporelet/packages/fc-snapshot-tools/pkg/oci"
)

// allow tests to stub snapshot logic
var startAndSnapshot = fc.StartAndSnapshot
var compareSnapshotDirs = fc.CompareSnapshotDirs

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "snapshot":
		snapshotCmd(os.Args[2:])
	case "push":
		pushCmd(os.Args[2:])
	case "pull":
		pullCmd(os.Args[2:])
	case "diff":
		diffCmd(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: sporectl <command> [options]")
	fmt.Println("Commands:")
	fmt.Println("  snapshot    Create a Firecracker snapshot")
	fmt.Println("  push        Push snapshot to OCI registry")
	fmt.Println("  pull        Pull snapshot from OCI registry")
	fmt.Println("  diff        Snapshot and compare against base layer")
}

func snapshotCmd(args []string) {
	fs := flag.NewFlagSet("snapshot", flag.ExitOnError)
	var (
		kernel  = fs.String("kernel", "", "Path to kernel image")
		rootfs  = fs.String("rootfs", "", "Path to rootfs image")
		cmdline = fs.String("cmdline", "console=ttyS0 reboot=k panic=1 pci=off", "Kernel command line")
		outDir  = fs.String("out-dir", ".", "Output directory")
		prefix  = fs.String("snapshot-prefix", "snapshot", "Snapshot file prefix")
		memMB   = fs.Int("mem", 1024, "Memory size (MB)")
		vcpus   = fs.Int("vcpu", 1, "Number of vCPUs")
		ociRef  = fs.String("oci-ref", "", "OCI reference to push")
		push    = fs.Bool("push", false, "Push after snapshot")
	)
	fs.Parse(args)

	if *kernel == "" || *rootfs == "" {
		fmt.Fprintln(os.Stderr, "kernel and rootfs are required")
		fs.Usage()
		os.Exit(1)
	}

	spec := fc.SnapshotSpec{
		Kernel:    *kernel,
		Rootfs:    *rootfs,
		Cmdline:   *cmdline,
		MemSizeMB: *memMB,
		VCPUCount: *vcpus,
	}

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create output dir: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if err := fc.StartAndSnapshot(ctx, spec, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "snapshot failed: %v\n", err)
		os.Exit(1)
	}

	if *prefix != "snapshot" {
		files := []string{"snapshot.mem", "snapshot.vmstate", "snapshot.config"}
		for _, f := range files {
			old := filepath.Join(*outDir, f)
			new := filepath.Join(*outDir, strings.Replace(f, "snapshot", *prefix, 1))
			os.Rename(old, new)
		}
	}

	if *push && *ociRef != "" {
		mem := filepath.Join(*outDir, fmt.Sprintf("%s.mem", *prefix))
		vmstate := filepath.Join(*outDir, fmt.Sprintf("%s.vmstate", *prefix))
		config := filepath.Join(*outDir, fmt.Sprintf("%s.config", *prefix))
		if err := fc.PushSnapshot(ctx, *ociRef, mem, vmstate, config); err != nil {
			fmt.Fprintf(os.Stderr, "push failed: %v\n", err)
			os.Exit(1)
		}
	}
}

func pushCmd(args []string) {
	fs := flag.NewFlagSet("push", flag.ExitOnError)
	var (
		outDir = fs.String("out-dir", ".", "Directory with snapshot files")
		prefix = fs.String("snapshot-prefix", "snapshot", "Snapshot file prefix")
		ociRef = fs.String("oci-ref", "", "OCI reference")
	)
	fs.Parse(args)

	if *ociRef == "" {
		fmt.Fprintln(os.Stderr, "oci-ref is required")
		fs.Usage()
		os.Exit(1)
	}

	mem := filepath.Join(*outDir, fmt.Sprintf("%s.mem", *prefix))
	vmstate := filepath.Join(*outDir, fmt.Sprintf("%s.vmstate", *prefix))
	config := filepath.Join(*outDir, fmt.Sprintf("%s.config", *prefix))

	ctx := context.Background()
	if err := fc.PushSnapshot(ctx, *ociRef, mem, vmstate, config); err != nil {
		fmt.Fprintf(os.Stderr, "push failed: %v\n", err)
		os.Exit(1)
	}
}

func pullCmd(args []string) {
	fs := flag.NewFlagSet("pull", flag.ExitOnError)
	var (
		ociRef = fs.String("oci-ref", "", "OCI reference")
		outDir = fs.String("out-dir", ".", "Directory to write snapshot")
	)
	fs.Parse(args)

	if *ociRef == "" {
		fmt.Fprintln(os.Stderr, "oci-ref is required")
		fs.Usage()
		os.Exit(1)
	}

	ctx := context.Background()
	if err := oci.PullSnapshot(ctx, *ociRef, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "pull failed: %v\n", err)
		os.Exit(1)
	}
}

func diffCmd(args []string) {
	if err := runDiff(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runDiff(args []string) error {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	var (
		baseDir = fs.String("base-dir", "", "Directory of base snapshot")
		kernel  = fs.String("kernel", "", "Path to kernel image")
		rootfs  = fs.String("rootfs", "", "Path to rootfs image")
		cmdline = fs.String("cmdline", "console=ttyS0 reboot=k panic=1 pci=off", "Kernel command line")
		outDir  = fs.String("out-dir", ".", "Output directory")
		prefix  = fs.String("snapshot-prefix", "snapshot", "Snapshot file prefix")
		memMB   = fs.Int("mem", 1024, "Memory size (MB)")
		vcpus   = fs.Int("vcpu", 1, "Number of vCPUs")
	)
	fs.Parse(args)

	if *baseDir == "" || *kernel == "" || *rootfs == "" {
		fs.Usage()
		return fmt.Errorf("base-dir, kernel and rootfs are required")
	}

	spec := fc.SnapshotSpec{
		Kernel:    *kernel,
		Rootfs:    *rootfs,
		Cmdline:   *cmdline,
		MemSizeMB: *memMB,
		VCPUCount: *vcpus,
	}

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output dir: %w", err)
	}

	ctx := context.Background()
	if err := startAndSnapshot(ctx, spec, *outDir); err != nil {
		return fmt.Errorf("snapshot failed: %w", err)
	}

	if *prefix != "snapshot" {
		files := []string{"snapshot.mem", "snapshot.vmstate", "snapshot.config"}
		for _, f := range files {
			old := filepath.Join(*outDir, f)
			new := filepath.Join(*outDir, strings.Replace(f, "snapshot", *prefix, 1))
			os.Rename(old, new)
		}
	}

	changes, err := compareSnapshotDirs(*baseDir, *outDir, *prefix)
	if err != nil {
		return fmt.Errorf("diff failed: %w", err)
	}

	if len(changes) == 0 {
		fmt.Println("no layer changes detected")
		return nil
	}
	fmt.Println("changed files:")
	for _, c := range changes {
		fmt.Printf("  %s\n", c)
	}

	return nil
}
