# Architecture

## Goals

- Provide a generic UpCloud receiver that can poll metrics per managed resource type
- Keep resource-type logic modular so managed database and managed load balancer can evolve independently
- Emit OpenTelemetry metrics with consistent resource and metric attributes

## Component Topology

- `factory.go`
  - Exposes `NewFactory()` and default config
- `config.go`
  - Defines API auth config, polling config, and per-resource settings
- `receiver.go`
  - Owns receiver lifecycle (`Start`, `Shutdown`) and poll loop
- `client.go`
  - UpCloud HTTP client and response models
- `scrape.go`
  - Transforms UpCloud API responses into `pmetric.Metrics`

## Data Flow

1. Receiver poll loop triggers scrape at `collection_interval`
2. For each enabled resource type:
   - Managed databases: call `/1.3/database/{uuid}/metrics`
   - Managed load balancers: call `metrics_path_template` with `{uuid}` replacement
3. Parse payload `metric_key -> data(cols, rows)`
4. Convert to OTel gauges with attributes:
   - `cloud.provider=upcloud`
   - `upcloud.resource.type`
   - `upcloud.resource.uuid`
   - `upcloud.metric.name`
   - `upcloud.series`
5. Forward to next metrics consumer in Collector pipeline

## Extensibility Pattern

Each resource type is represented by:

- Config block (`managed_databases`, `managed_load_balancers`)
- Client method (`GetManagedDatabaseMetrics`, `GetManagedLoadBalancerMetrics`)
- Scrape branch in `scrapeMetrics`

Adding new managed services follows the same pattern without changing receiver lifecycle code.
