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

## 6. Full Contrib Distribution

To preserve all components from the official `otelcol-contrib` distribution while adding this receiver:

```bash
make build-collector-contrib OTELCOL_VERSION=0.146.1
```

This command:

1. Downloads the upstream contrib manifest for the pinned OTel release
2. Injects `upcloudreceiver` into the receivers list
3. Builds `./_build/otelcol-contrib-upcloud` with `ocb`

Container build (official contrib runtime base):

```bash
docker build --build-arg OTELCOL_VERSION=0.146.1 -t ghcr.io/<owner>/otelcol-contrib-upcloud:local .
```

## 7. GitHub Actions Pipelines

- `.github/workflows/ci.yaml`
  - Runs on `pull_request` and `main` pushes
  - Executes linting, tests, full contrib build, config validation, and docker smoke build
  - Does not publish images

- `.github/workflows/release-image.yaml`
  - Runs on semver tags (`v*.*.*`)
  - Re-runs verification gates, then publishes `linux/amd64` and `linux/arm64` images to GHCR
  - Uses semver-only image tags: `X.Y.Z`, `X.Y`, `X`
