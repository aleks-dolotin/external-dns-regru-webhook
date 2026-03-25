# ---- Build stage ----
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w -X main.Version=${VERSION}" \
    -o /out/sidecar ./cmd/sidecar

# ---- Runtime stage ----
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/sidecar /sidecar

# Default config path (mount via ConfigMap).
ENV REGADAPTER_MAPPINGS_PATH=/etc/reg-adapter/mappings.yaml

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/sidecar"]
