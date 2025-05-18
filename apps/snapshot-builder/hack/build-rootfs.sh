#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <output-rootfs>" >&2
  exit 1
fi

OUT=$1
DEBIAN_VERSION=${DEBIAN_VERSION:-bookworm}
ROOTFS_DIR=$(mktemp -d)
trap 'sudo rm -rf "$ROOTFS_DIR"' EXIT

sudo debootstrap --arch=amd64 --variant=minbase "$DEBIAN_VERSION" "$ROOTFS_DIR" http://deb.debian.org/debian

sudo chroot "$ROOTFS_DIR" apt-get update
sudo chroot "$ROOTFS_DIR" apt-get install -y containerd docker.io docker-compose-plugin

# Install compose-preheater if available
if command -v compose-preheater >/dev/null 2>&1; then
  sudo cp "$(command -v compose-preheater)" "$ROOTFS_DIR/usr/local/bin/"
  sudo chroot "$ROOTFS_DIR" compose-preheater || true
fi

# Install guest-agent if available
if command -v guest-agent >/dev/null 2>&1; then
  sudo cp "$(command -v guest-agent)" "$ROOTFS_DIR/usr/local/bin/"
fi

# Create ext4 image
IMG_SIZE=${IMG_SIZE:-512} # in MB
sudo dd if=/dev/zero of="$OUT" bs=1M count="$IMG_SIZE"
sudo mkfs.ext4 -F "$OUT"
MNT=$(mktemp -d)
trap 'sudo umount "$MNT" 2>/dev/null || true; sudo rm -rf "$MNT"; sudo rm -rf "$ROOTFS_DIR"' EXIT
sudo mount "$OUT" "$MNT"
sudo cp -a "$ROOTFS_DIR"/. "$MNT"/
sudo umount "$MNT"
