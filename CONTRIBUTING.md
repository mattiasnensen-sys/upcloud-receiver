# Contributing

## Requirements

- Go 1.23+
- `gofmt`
- Optional: OpenTelemetry Collector Builder (`ocb`)

## Development

```bash
make fmt
make test
```

## Pull Requests

- Keep changes focused and small
- Add tests for behavior changes
- Keep receiver config backwards-compatible when possible
- Do not introduce project-specific branding or internal names
