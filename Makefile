SHELL := /bin/bash
GO_BIN := $(shell go env GOPATH)/bin

.PHONY: fmt test tidy build-collector build-collector-local

fmt:
	gofmt -w $(shell find . -name '*.go' -type f -not -path './_build/*' -not -path './vendor/*')

test:
	go test ./...

tidy:
	go mod tidy

build-collector:
	go install go.opentelemetry.io/collector/cmd/builder@v0.146.1
	$(GO_BIN)/builder --config ./cmd/otelcol-upcloud/builder-config.yaml

build-collector-local:
	go install go.opentelemetry.io/collector/cmd/builder@v0.146.1
	$(GO_BIN)/builder --skip-strict-versioning --config ./cmd/otelcol-upcloud/builder-config.yaml
