# Snapshot Builder

Utilities for producing Firecracker snapshot images used by Sporelet.

## Layer 1 snapshot workflow

Layer 1 images bundle a VM with containerd and Docker Compose services already running. They are produced by `build.sh` using the following steps:

1. `hack/build-rootfs.sh` creates a minimal Debian root filesystem, installs an
   SSH server and configures it for root login. It also sets up a static IP of
   `172.16.0.2/24` via systemd-networkd and copies the host's public key defined
   by `PUBKEY_FILE` (defaults to `~/.ssh/id_rsa.pub`) into
   `/root/.ssh/authorized_keys`. The `compose-preheater` and `guest-agent`
   binaries are copied into `/usr/local/bin` when available.
2. `hack/run-and-snapshot.sh` boots the VM with the kernel from `hack/build-kernel.sh`. Once the guest agent is ready, it executes `compose-preheater` inside the VM to start the Compose stack and then triggers a snapshot through the Firecracker API.
3. The resulting `.mem`, `.vmstate` and `.config` files under `dist/` form the Layer 1 OCI artifact.

These snapshots can then be pushed to any OCI registry using `oras push`.

### SSH public key

`build-rootfs.sh` looks for an SSH public key specified by the `PUBKEY_FILE`
environment variable. If unset, it defaults to `~/.ssh/id_rsa.pub`. When found,
the key is placed in `/root/.ssh/authorized_keys` inside the rootfs so that
`run_and_snapshot.go` can log in without prompting.
