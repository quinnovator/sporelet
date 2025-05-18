FROM golang:1.22-bullseye AS builder
WORKDIR /src

# Copy Go modules
COPY apps/sporectl/go.mod apps/sporectl/
COPY packages/fc-snapshot-tools/go.mod packages/fc-snapshot-tools/

# Copy source
COPY apps/sporectl apps/sporectl
COPY packages/fc-snapshot-tools packages/fc-snapshot-tools

# Build sporectl binary
RUN cd apps/sporectl && CGO_ENABLED=0 go build -o /sporectl .

FROM gcr.io/distroless/static
COPY --from=builder /sporectl /sporectl
ENTRYPOINT ["/sporectl"]
