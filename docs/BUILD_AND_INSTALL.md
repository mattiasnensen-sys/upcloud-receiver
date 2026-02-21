# Build and Install Plan

## 1. Receiver Package Development

- Implement and test receiver in `receiver/upcloudreceiver`
- Validate config parsing, auth behavior, and payload-to-OTel mapping with unit tests

## 2. Build a Custom Collector Distribution

Use OpenTelemetry Collector Builder (`ocb`) with `cmd/otelcol-upcloud/builder-config.yaml`.

```bash
go install go.opentelemetry.io/collector/cmd/builder@v0.146.1
builder --config ./cmd/otelcol-upcloud/builder-config.yaml
```

Binary output path is configured in `dist.output_path`.

## 3. Local Iteration with Receiver Source Path

The builder config includes a local `path` override for the receiver module.
This allows local development without publishing tags.

## 4. Open-Source Release Workflow

1. Tag and release receiver module
2. Update builder config component version to released tag
3. Build distribution binary for Linux/amd64 and Linux/arm64
4. Publish checksums and release notes

## 5. Production Installation

- Build and publish collector image containing the custom binary
- Deploy with Helm or Kubernetes manifests
- Configure receiver block in collector config
- Roll out with canary deployment and verify metric cardinality and scrape latency
