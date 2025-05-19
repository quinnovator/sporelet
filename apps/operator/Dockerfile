FROM golang:1.22-bullseye AS builder
WORKDIR /src

# Install ORAS CLI for pulling snapshots
RUN go install oras.land/oras/cmd/oras@v1.1.0

# Copy Go modules
COPY apps/operator/go.mod apps/operator/
COPY packages/fc-snapshot-tools/go.mod packages/fc-snapshot-tools/
COPY cmd/spore-shim/go.mod cmd/spore-shim/

# Copy source
COPY apps/operator apps/operator
COPY packages/fc-snapshot-tools packages/fc-snapshot-tools
COPY cmd/spore-shim cmd/spore-shim

# Build operator binary
RUN cd apps/operator && CGO_ENABLED=0 go build -o /operator .
# Build spore-shim binary
RUN cd cmd/spore-shim && CGO_ENABLED=0 go build -o /spore-shim .

FROM gcr.io/distroless/static
COPY --from=builder /operator /operator
COPY --from=builder /spore-shim /spore-shim
COPY --from=builder /go/bin/oras /usr/local/bin/oras
ENTRYPOINT ["/operator"]
