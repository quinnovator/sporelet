package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	fc "github.com/quinnovator/sporelet/packages/fc-snapshot-tools"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "snapshot":
		snapshotCmd(os.Args[2:])
	case "diff":
		diffCmd(os.Args[2:])
	case "restore":
		restoreCmd(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: spore-shim <command> [options]")
	fmt.Println("Commands:")
	fmt.Println("  snapshot  Create a Firecracker snapshot")
	fmt.Println("  diff      Snapshot and compare against a base layer")
	fmt.Println("  restore   Restore a microVM from snapshot files")
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
}

func diffCmd(args []string) {
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
		fmt.Fprintln(os.Stderr, "base-dir, kernel and rootfs are required")
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

	changes, err := fc.CompareSnapshotDirs(*baseDir, *outDir, *prefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "diff failed: %v\n", err)
		os.Exit(1)
	}

	if len(changes) == 0 {
		fmt.Println("no layer changes detected")
		return
	}
	fmt.Println("changed files:")
	for _, c := range changes {
		fmt.Printf("  %s\n", c)
	}
}

func restoreCmd(args []string) {
	fs := flag.NewFlagSet("restore", flag.ExitOnError)
	var (
		fcBin     = fs.String("fc-bin", "firecracker", "firecracker binary")
		jailerBin = fs.String("jailer-bin", "jailer", "jailer binary")
		socket    = fs.String("socket-path", "", "firecracker socket path")
		id        = fs.String("id", "", "vm id")
	)
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "snapshot directory required")
		fs.Usage()
		os.Exit(1)
	}

	dir := fs.Arg(0)
	spec := fc.RestoreSpec{
		MemFile:     filepath.Join(dir, "snapshot.mem"),
		VMStateFile: filepath.Join(dir, "snapshot.vmstate"),
		ConfigFile:  filepath.Join(dir, "snapshot.config"),
		JailerBin:   *jailerBin,
		FCBin:       *fcBin,
		SocketPath:  *socket,
		ID:          *id,
	}

	if err := fc.Restore(context.Background(), spec); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
