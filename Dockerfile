# syntax=docker/dockerfile:1

FROM golang:1.24-alpine AS builder
WORKDIR /src

ENV GOFLAGS=-mod=vendor

COPY go.mod ./
COPY vendor ./vendor
COPY cmd ./cmd
COPY pkg ./pkg

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/agentpark ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /out/agentpark /agentpark
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/agentpark"]
