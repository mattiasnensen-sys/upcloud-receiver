# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.25
ARG OTELCOL_VERSION=0.146.1

FROM golang:${GO_VERSION}-bookworm AS builder
ARG OTELCOL_VERSION
WORKDIR /workspace

COPY . .
RUN ./scripts/build-contrib-distribution.sh "${OTELCOL_VERSION}"

FROM otel/opentelemetry-collector-contrib:${OTELCOL_VERSION}

# Preserve upstream entrypoint and flags by replacing the binary in-place.
COPY --from=builder /workspace/_build/otelcol-contrib-upcloud /otelcol-contrib
