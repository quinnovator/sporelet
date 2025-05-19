#!/usr/bin/env bash
set -euo pipefail

# Create the tap0 interface for snapshot builds

if ip link show tap0 &>/dev/null; then
  echo "tap0 already exists"
  exit 0
fi

sudo ip tuntap add dev tap0 mode tap
sudo ip addr add 172.16.0.1/24 dev tap0
sudo ip link set tap0 up

echo "tap0 configured with 172.16.0.1/24"
