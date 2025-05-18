# compose-preheater

> Docker Compose warm‑up utility for Sporelet snapshot builds

`compose-preheater` starts services defined in a Docker Compose file and waits
until they report healthy before returning. It is used during snapshot creation
so that containerd and any required sidecars are already running when the VM
snapshot is taken.

## Installation

```
go install github.com/quinnovator/sporelet/packages/compose-preheater@latest
```

## Usage

```
compose-preheater -f docker-compose.yml -timeout 90s
```

By default it looks for `docker-compose.yml` in the working directory and waits
up to 60 seconds for all services to become healthy.
