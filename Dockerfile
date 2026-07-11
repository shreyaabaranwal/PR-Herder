# ---- Build stage ----
FROM golang:1.25-bookworm AS builder

WORKDIR /build

# Cache dependency downloads separately from source changes -- go.mod/go.sum
# rarely change, so this layer stays cached across most rebuilds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0: static binary, no libc dependency -- required for the
# distroless base image below, which has no shell or dynamic linker.
RUN CGO_ENABLED=0 GOOS=linux go build -o /build/prherder ./cmd/prherder

# ---- Final stage ----
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /build/prherder /app/prherder
COPY migrations /app/migrations

# distroless/static-debian12:nonroot already runs as a non-root user by
# default (uid 65532) -- no explicit USER directive needed.

EXPOSE 8090

ENTRYPOINT ["/app/prherder"]
