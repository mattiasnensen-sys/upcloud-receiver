# upcloud-receiver

OpenTelemetry Collector receiver for UpCloud managed services metrics.

This repository scaffolds a custom receiver package named `upcloud` and an OpenTelemetry Collector distribution manifest for local and CI builds.

## Scope

- Managed databases metrics via UpCloud API (`/1.3/database/{uuid}/metrics`)
- Managed load balancers metrics via UpCloud API (path template, configurable)

## Repository Layout

- `receiver/upcloudreceiver/` receiver implementation package
- `cmd/otelcol-upcloud/builder-config.yaml` Collector Builder manifest
- `examples/otelcol-config.yaml` sample collector pipeline config
- `docs/ARCHITECTURE.md` architecture and extensibility design
- `docs/BUILD_AND_INSTALL.md` build and installation workflow

## Status

- Stability target: `alpha`
- Managed database metrics path and payload parsing: scaffolded and implemented
- Managed load balancer metrics: scaffolded via provider interface and configurable endpoint template
- Authentication: bearer token and basic auth, including `_file` credential loading options
- Metric naming: OpenTelemetry-style names and units with percent-to-ratio normalization for utilization
- Resource targeting: explicit UUIDs and autodiscovery with include/exclude controls

## Quick Start

1. Build and test receiver package:

```bash
make test
```

2. Build a collector binary including this receiver:

```bash
make build-collector
```

3. Run collector with the sample config:

```bash
./_build/otelcol-upcloud --config ./examples/otelcol-config.yaml
```

## Licensing

Apache-2.0. See `LICENSE`.
