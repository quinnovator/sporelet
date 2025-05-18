FROM golang:1.22-bullseye AS builder
WORKDIR /src

# Copy Go modules
COPY apps/operator/go.mod apps/operator/
COPY packages/fc-snapshot-tools/go.mod packages/fc-snapshot-tools/

# Copy source
COPY apps/operator apps/operator
COPY packages/fc-snapshot-tools packages/fc-snapshot-tools

# Build operator binary
RUN cd apps/operator && CGO_ENABLED=0 go build -o /operator .

FROM gcr.io/distroless/static
COPY --from=builder /operator /operator
ENTRYPOINT ["/operator"]
