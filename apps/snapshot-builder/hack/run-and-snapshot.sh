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

OUT_DIR=$(dirname "$SNAP_PREFIX")
BASENAME=$(basename "$SNAP_PREFIX")

CMDLINE="console=ttyS0 reboot=k panic=1 pci=off init=/usr/local/bin/compose-preheater"

fc-tools snapshot \
  --kernel "$KERNEL" \
  --rootfs "$ROOTFS" \
  --cmdline "$CMDLINE" \
  --snapshot-prefix "$BASENAME" \
  --out-dir "$OUT_DIR"
