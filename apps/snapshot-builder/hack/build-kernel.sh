#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <output-vmlinux>" >&2
  exit 1
fi

OUT=$1
KERNEL_VERSION=${KERNEL_VERSION:-6.6.2}
WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT

cd "$WORKDIR"

# Download and extract Linux sources
curl -fsSL "https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-${KERNEL_VERSION}.tar.xz" -o linux.tar.xz
mkdir src && tar -xf linux.tar.xz -C src --strip-components=1
cd src

# Use Firecracker microVM config
curl -fsSL https://raw.githubusercontent.com/firecracker-microvm/firecracker/main/resources/microvm-kernel-x86_64.config -o .config
make olddefconfig

# Compile the kernel
make -j"$(nproc)" vmlinux

# Copy result
cp vmlinux "$OUT"
