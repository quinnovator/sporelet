# Reproducible environment for building snapshots
FROM ubuntu:22.04

ARG GO_VERSION=1.22.0
ARG NODE_VERSION=20
ARG PNPM_VERSION=9.12.0
ARG FC_VERSION=1.5.2

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential curl git ca-certificates \
    libssl-dev libelf-dev bison flex bc \
    qemu-utils xz-utils \
    && rm -rf /var/lib/apt/lists/*

# Install Go
RUN curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz | \
    tar -xzC /usr/local && \
    ln -s /usr/local/go/bin/go /usr/local/bin/go && \
    ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

ENV PATH="/usr/local/go/bin:$PATH"

# Install Node.js and pnpm
RUN curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - && \
    apt-get update && apt-get install -y --no-install-recommends nodejs && \
    corepack enable && corepack prepare pnpm@${PNPM_VERSION} --activate && \
    rm -rf /var/lib/apt/lists/*

# Install Firecracker binaries
RUN curl -L https://github.com/firecracker-microvm/firecracker/releases/download/v${FC_VERSION}/firecracker-v${FC_VERSION}-x86_64.tgz | \
    tar -xz && \
    mv release-v${FC_VERSION}-x86_64/firecracker-v${FC_VERSION} /usr/local/bin/firecracker && \
    mv release-v${FC_VERSION}-x86_64/jailer-v${FC_VERSION} /usr/local/bin/jailer && \
    chmod +x /usr/local/bin/firecracker /usr/local/bin/jailer && \
    rm -r release-v${FC_VERSION}-x86_64

WORKDIR /workspace

CMD ["bash"]
