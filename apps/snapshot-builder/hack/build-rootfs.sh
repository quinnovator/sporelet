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
sudo chroot "$ROOTFS_DIR" apt-get install -y \
  containerd docker.io docker-compose-plugin openssh-server

# Enable root SSH login
sudo sed -i 's/^#PermitRootLogin.*/PermitRootLogin yes/' "$ROOTFS_DIR/etc/ssh/sshd_config"

# Copy host public key for passwordless login
PUBKEY_FILE=${PUBKEY_FILE:-$HOME/.ssh/id_rsa.pub}
if [[ -f $PUBKEY_FILE ]]; then
  sudo mkdir -p "$ROOTFS_DIR/root/.ssh"
  sudo install -m 600 "$PUBKEY_FILE" "$ROOTFS_DIR/root/.ssh/authorized_keys"
fi

# Configure static IP using systemd-networkd
sudo mkdir -p "$ROOTFS_DIR/etc/systemd/network"
cat <<EOF | sudo tee "$ROOTFS_DIR/etc/systemd/network/eth0.network" >/dev/null
[Match]
Name=eth0

[Network]
Address=172.16.0.2/24
Gateway=172.16.0.1
EOF
sudo mkdir -p "$ROOTFS_DIR/etc/systemd/system/multi-user.target.wants"
sudo ln -sf /lib/systemd/system/systemd-networkd.service \
  "$ROOTFS_DIR/etc/systemd/system/multi-user.target.wants/systemd-networkd.service"

# Install compose-preheater if available
if command -v compose-preheater >/dev/null 2>&1; then
  sudo cp "$(command -v compose-preheater)" "$ROOTFS_DIR/usr/local/bin/"
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
