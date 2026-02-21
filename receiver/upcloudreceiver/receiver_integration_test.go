// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloudreceiver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

func TestReceiverIntegration_StartAndConsume(t *testing.T) {
	fixture := mustReadFixture(t, "testdata/integration/managed_database_metrics.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer server.Close()

	cfg := &Config{
		CollectionInterval: 50 * time.Millisecond,
		InitialDelay:       0,
		API: APIConfig{
			Endpoint: server.URL,
			Token:    "fixture-token",
			Timeout:  2 * time.Second,
		},
		ManagedDatabases: ManagedDatabaseConfig{
			Enabled: true,
			UUIDs:   []string{"db-uuid"},
			Period:  "5m",
		},
	}

	client, err := NewHTTPClient(cfg.API, cfg.ManagedLoadBalancers.MetricsPathTemplate)
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}

	capture := &metricsCapture{}
	next, err := consumer.NewMetrics(capture.consume)
	if err != nil {
		t.Fatalf("new metrics consumer: %v", err)
	}

	r := newMetricsReceiver(cfg, receiver.Settings{
		ID: component.MustNewID("upcloud"),
		TelemetrySettings: component.TelemetrySettings{
			Logger: zap.NewNop(),
		},
	}, next, client)

	if err := r.Start(context.Background(), nil); err != nil {
		t.Fatalf("receiver start failed: %v", err)
	}
	defer func() {
		_ = r.Shutdown(context.Background())
	}()

	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if capture.count() > 0 {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}

	if capture.count() == 0 {
		t.Fatalf("expected at least one metrics batch to be consumed")
	}

	first := capture.first()
	names := allMetricNames(first)
	if len(names) == 0 {
		t.Fatalf("expected metrics in consumed batch")
	}
}

type metricsCapture struct {
	mu      sync.Mutex
	batches []pmetric.Metrics
}

func (c *metricsCapture) consume(_ context.Context, md pmetric.Metrics) error {
	copied := pmetric.NewMetrics()
	md.CopyTo(copied)
	c.mu.Lock()
	c.batches = append(c.batches, copied)
	c.mu.Unlock()
	return nil
}

func (c *metricsCapture) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.batches)
}

func (c *metricsCapture) first() pmetric.Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.batches) == 0 {
		return pmetric.NewMetrics()
	}
	return c.batches[0]
}
