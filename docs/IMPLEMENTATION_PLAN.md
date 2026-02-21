# Implementation Plan

## Phase 0: Scaffold (done)

- Set up receiver package skeleton (`config`, `factory`, `receiver`, `client`, `scrape`)
- Add Apache-2.0 license and OSS docs
- Add collector-builder manifest for local distribution builds

## Phase 1: Managed Databases MVP

- Confirm API contract against `/1.3/database/{uuid}/metrics`
- Add integration tests with recorded fixtures
- Define metric unit mapping where hints are explicit
- Add optional metric renaming map (UpCloud key -> OTel name)

## Phase 2: Managed Load Balancers MVP

- Confirm LB metrics endpoint and payload contract
- Implement parser adaptation if LB payload diverges from database payload
- Add load balancer integration tests and sample configs

## Phase 3: Production Hardening

- Retry/backoff policy for transient 429/5xx
- Per-resource scrape timeout and partial-failure reporting metrics
- Add receiver self-observability metrics (scrape duration, success/failure)
- Add cardinality controls for series labels

## Phase 4: Release and Adoption

- Publish tagged module releases
- Add changelog and release automation
- Build/publish collector container image with receiver included
- Submit upstream proposal to OTel contrib if desired
