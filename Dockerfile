FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w -X main.Version=${VERSION}" \
    -o /out/webhook ./cmd/webhook

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/webhook /webhook

EXPOSE 8888
USER 65532:65532

ENTRYPOINT ["/webhook"]
