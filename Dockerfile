# ---- Build stage ----
FROM golang:1.25-bookworm AS builder

WORKDIR /build


COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /build/prherder ./cmd/prherder


FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /build/prherder /app/prherder
COPY migrations /app/migrations


EXPOSE 8090

ENTRYPOINT ["/app/prherder"]
