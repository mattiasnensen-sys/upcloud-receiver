SHELL := /bin/bash
GO_BIN := $(shell go env GOPATH)/bin
OTELCOL_VERSION ?= 0.146.1

.PHONY: fmt test tidy vet ci build-collector build-collector-local build-collector-contrib

fmt:
	gofmt -w $(shell find . -name '*.go' -type f -not -path './_build/*' -not -path './vendor/*')

test:
	go test ./...

tidy:
	go mod tidy

vet:
	go vet ./...

ci: tidy fmt vet test
	git diff --exit-code

build-collector:
	go install go.opentelemetry.io/collector/cmd/builder@v0.146.1
	$(GO_BIN)/builder --config ./cmd/otelcol-upcloud/builder-config.yaml

build-collector-local:
	go install go.opentelemetry.io/collector/cmd/builder@v0.146.1
	$(GO_BIN)/builder --skip-strict-versioning --config ./cmd/otelcol-upcloud/builder-config.yaml

build-collector-contrib:
	./scripts/build-contrib-distribution.sh $(OTELCOL_VERSION)
