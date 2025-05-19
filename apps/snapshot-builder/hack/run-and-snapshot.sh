#!/usr/bin/env bash
set -euo pipefail

KERNEL=""
ROOTFS=""
SNAP_PREFIX=""

usage() {
  echo "Usage: $0 --kernel <vmlinux> --rootfs <rootfs.ext4> --snapshot-prefix <prefix>" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case $1 in
    --kernel) KERNEL=$2; shift 2;;
    --rootfs) ROOTFS=$2; shift 2;;
    --snapshot-prefix) SNAP_PREFIX=$2; shift 2;;
    *) usage;;
  esac
done

[[ -z $KERNEL || -z $ROOTFS || -z $SNAP_PREFIX ]] && usage

CMDLINE="console=ttyS0 reboot=k panic=1 pci=off"

# Ensure tap0 exists for the VM networking
if ! ip link show tap0 &>/dev/null; then
  "$(dirname "$0")/setup-tap0.sh"
fi

# Launch the VM, run compose-preheater inside via SSH, then snapshot
go run "$(dirname "$0")/run_and_snapshot.go" \
  --kernel "$KERNEL" \
  --rootfs "$ROOTFS" \
  --cmdline "$CMDLINE" \
  --snapshot-prefix "$SNAP_PREFIX"

