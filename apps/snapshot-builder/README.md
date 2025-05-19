# Snapshot Builder

Utilities for producing Firecracker snapshot images used by Sporelet.

## Layer 1 snapshot workflow

Layer 1 images bundle a VM with containerd and Docker Compose services already running. They are produced by `build.sh` using the following steps:

1. `hack/build-rootfs.sh` creates a minimal Debian root filesystem and copies the `compose-preheater` and `guest-agent` binaries into `/usr/local/bin` when available.
2. `hack/run-and-snapshot.sh` boots the VM with the kernel from `hack/build-kernel.sh`. Once the guest agent is ready, it executes `compose-preheater` inside the VM to start the Compose stack and then triggers a snapshot through the Firecracker API.
3. The resulting `.mem`, `.vmstate` and `.config` files under `dist/` form the Layer 1 OCI artifact.

These snapshots can then be pushed to any OCI registry using `oras push`.
