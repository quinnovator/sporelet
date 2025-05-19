package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	fc "github.com/quinnovator/sporelet/packages/fc-snapshot-tools/pkg/firecracker"
)

func main() {
	kernel := flag.String("kernel", "", "path to vmlinux")
	rootfs := flag.String("rootfs", "", "path to rootfs")
	snapPrefix := flag.String("snapshot-prefix", "snapshot", "snapshot prefix")
	cmdline := flag.String("cmdline", "console=ttyS0 reboot=k panic=1 pci=off", "kernel cmdline")
	flag.Parse()

	if *kernel == "" || *rootfs == "" || *snapPrefix == "" {
		log.Fatalf("usage: %s --kernel <vmlinux> --rootfs <rootfs.ext4> --snapshot-prefix <prefix>", os.Args[0])
	}

	outDir := filepath.Dir(*snapPrefix)
	base := filepath.Base(*snapPrefix)

	ctx := context.Background()

	client, err := fc.NewClient("firecracker", "jailer", base, "")
	if err != nil {
		log.Fatalf("client: %v", err)
	}

	vmCfg := fc.VMConfig{
		KernelImagePath: *kernel,
		RootDrive: fc.Drive{
			PathOnHost:   *rootfs,
			IsRootDevice: true,
			IsReadOnly:   false,
		},
		KernelArgs: *cmdline,
		MemSizeMB:  1024,
		VCPUCount:  1,
	}

	if err := client.StartVM(ctx, vmCfg); err != nil {
		log.Fatalf("start VM: %v", err)
	}

	if err := client.WaitForVSockHandshake(ctx); err != nil {
		log.Fatalf("handshake: %v", err)
	}

	preheat := exec.CommandContext(ctx, "ssh", "-o", "StrictHostKeyChecking=no", "root@172.16.0.2", "compose-preheater")
	preheat.Stdout = os.Stdout
	preheat.Stderr = os.Stderr
	if err := preheat.Run(); err != nil {
		log.Fatalf("compose-preheater: %v", err)
	}

	snapCfg := fc.SnapshotConfig{
		MemFilePath:     filepath.Join(outDir, base+".mem"),
		VMStateFilePath: filepath.Join(outDir, base+".vmstate"),
		ConfigFilePath:  filepath.Join(outDir, base+".config"),
	}

	if err := client.CreateSnapshot(ctx, snapCfg); err != nil {
		log.Fatalf("snapshot: %v", err)
	}
}
