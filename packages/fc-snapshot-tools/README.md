# fc-snapshot-tools

> Firecracker snapshot tools for Sporelet

This package provides Go libraries and CLI tools for creating and managing Firecracker microVM snapshots as OCI artifacts.

## Features

- Start Firecracker microVMs with jailer
- Create snapshots of running microVMs
- Push snapshots to OCI registries as artifacts
- Pull snapshots from OCI registries

## Installation

```bash
go install github.com/quinnovator/sporelet/packages/fc-snapshot-tools/cmd/fc-tools@latest
```

## Usage as a library

```go
package main

import (
	"context"
	"log"

	"github.com/quinnovator/sporelet/packages/fc-snapshot-tools"
)

func main() {
	// Create a snapshot spec
	spec := fc.SnapshotSpec{
		Kernel:  "/path/to/vmlinux",
		Rootfs:  "/path/to/rootfs.ext4",
		Cmdline: "console=ttyS0 reboot=k panic=1 pci=off",
		Net: fc.NetConfig{
			HostDevName: "tap0",
			MacAddr:     "AA:FC:00:00:00:01",
			IPAddr:      "172.16.0.2",
			Mask:        "255.255.255.0",
			Gateway:     "172.16.0.1",
		},
		MemSizeMB: 1024,
		VCPUCount: 1,
	}

	// Start VM and create snapshot
	ctx := context.Background()
	outDir := "/path/to/output"
	
	if err := fc.StartAndSnapshot(ctx, spec, outDir); err != nil {
		log.Fatalf("Failed to create snapshot: %v", err)
	}

	// Push snapshot to OCI registry
	ociRef := "ghcr.io/quinnovator/sporelet/layer1:dev"
	memFile := outDir + "/snapshot.mem"
	vmstateFile := outDir + "/snapshot.vmstate"
	configFile := outDir + "/snapshot.config"
	
	if err := fc.PushSnapshot(ctx, ociRef, memFile, vmstateFile, configFile); err != nil {
		log.Fatalf("Failed to push snapshot: %v", err)
	}
}
```

## CLI Usage

### Creating a snapshot

```bash
fc-tools snapshot \
  --kernel /path/to/vmlinux \
  --rootfs /path/to/rootfs.ext4 \
  --out-dir /path/to/output \
  --snapshot-prefix layer1
```

### Pushing a snapshot to an OCI registry

```bash
fc-tools push \
  --out-dir /path/to/output \
  --snapshot-prefix layer1 \
  --oci-ref ghcr.io/quinnovator/sporelet/layer1:dev
```

### Creating and pushing in one step

```bash
fc-tools snapshot \
  --kernel /path/to/vmlinux \
  --rootfs /path/to/rootfs.ext4 \
  --out-dir /path/to/output \
  --snapshot-prefix layer1 \
  --oci-ref ghcr.io/quinnovator/sporelet/layer1:dev \
  --push
```

## Requirements

- Firecracker binary in PATH
- Jailer binary in PATH (optional, but recommended)
- ORAS CLI for pushing to OCI registries

## Integration with Sporelet

This package is used by the Sporelet snapshot builder to create and manage microVM snapshots. It's designed to work seamlessly with the Sporelet ecosystem.

## License

Apache License 2.0
