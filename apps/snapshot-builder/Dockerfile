FROM alpine:3.20 AS builder
RUN apk add --no-cache build-base go git curl e2fsprogs qemu-img gzip

COPY packages/fc-snapshot-tools /src/fc-snapshot-tools
RUN cd /src/fc-snapshot-tools && go build -o /usr/local/bin/fc-tools ./cmd/...

COPY packages/compose-preheater /src/compose-preheater
RUN cd /src/compose-preheater && go build -o /usr/local/bin/compose-preheater .