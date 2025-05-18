# sporectl

Command-line tool for working with Sporelet snapshots.

## Usage

```bash
# build a snapshot and push to an OCI registry
sporectl snapshot \
  --kernel /path/to/vmlinux \
  --rootfs /path/to/rootfs.ext4 \
  --out-dir dist \
  --snapshot-prefix layer1 \
  --oci-ref ghcr.io/your/repo/layer1:latest \
  --push

# push an existing snapshot
sporectl push \
  --out-dir dist \
  --snapshot-prefix layer1 \
  --oci-ref ghcr.io/your/repo/layer1:latest

# pull a snapshot
sporectl pull \
  --oci-ref ghcr.io/your/repo/layer1:latest \
  --out-dir dist
```
