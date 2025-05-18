FROM golang:1.22-bullseye AS builder
ARG FC_VERSION=1.5.2
ENV DEBIAN_FRONTEND=noninteractive

# Tools needed to assemble the snapshot
RUN apt-get update && apt-get install -y --no-install-recommends \
    debootstrap docker.io docker-compose-plugin \
    qemu-utils e2fsprogs gzip curl git ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Build fc-snapshot-tools
COPY packages/fc-snapshot-tools /src/fc-snapshot-tools
RUN cd /src/fc-snapshot-tools && go build -o /usr/local/bin/fc-tools ./cmd/...

# Build compose-preheater
COPY packages/compose-preheater /src/compose-preheater
RUN cd /src/compose-preheater && go build -o /usr/local/bin/compose-preheater .

# Install Firecracker binaries
RUN curl -L https://github.com/firecracker-microvm/firecracker/releases/download/v${FC_VERSION}/firecracker-v${FC_VERSION}-x86_64.tgz \
    | tar -xz && \
    mv release-v${FC_VERSION}-x86_64/firecracker-v${FC_VERSION} /usr/local/bin/firecracker && \
    mv release-v${FC_VERSION}-x86_64/jailer-v${FC_VERSION} /usr/local/bin/jailer && \
    chmod +x /usr/local/bin/firecracker /usr/local/bin/jailer && \
    rm -r release-v${FC_VERSION}-x86_64

# Add snapshot build scripts
COPY apps/snapshot-builder/build.sh /snapshot-builder/build.sh
COPY apps/snapshot-builder/hack /snapshot-builder/hack
RUN chmod +x /snapshot-builder/build.sh /snapshot-builder/hack/*.sh

WORKDIR /snapshot-builder
ENV SNAP_DIR=/snapshot
VOLUME ["/snapshot"]
ENTRYPOINT ["/snapshot-builder/build.sh"]
