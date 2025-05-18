#!/usr/bin/env bash
set -euo pipefail

SNAP_DIR=${SNAP_DIR:-dist}
mkdir -p "$SNAP_DIR"

# 1. Build kernel with virt‑config (cached)
KERNEL="$SNAP_DIR/vmlinux"
if [[ ! -f $KERNEL ]]; then
  ./hack/build-kernel.sh "$KERNEL"
fi

# 2. Assemble rootfs (layer0)
ROOTFS="$SNAP_DIR/rootfs.ext4"
./hack/build-rootfs.sh "$ROOTFS"

# 3. Start microVM with Firecracker jailer, run compose‑preheater inside,
#    then trigger snapshot via FC API
./hack/run-and-snapshot.sh \
  --kernel "$KERNEL" \
  --rootfs "$ROOTFS" \
  --snapshot-prefix "$SNAP_DIR/layer1"

# 4. Push snapshot to registry via ORAS (layer1)
oras push "$OCI_REF" \
  --artifact-type application/vnd.firecracker.layer.v1 \
  "$SNAP_DIR/layer1.mem" "$SNAP_DIR/layer1.vmstate" "$SNAP_DIR/layer1.config"