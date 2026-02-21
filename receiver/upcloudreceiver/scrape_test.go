// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloudreceiver

import (
	"context"
	"math"
	"testing"

	"go.uber.org/zap"
)

type fakeClient struct {
	dbResp MetricsResponse
	lbResp MetricsResponse
}

func (f *fakeClient) GetManagedDatabaseMetrics(context.Context, string, string) (MetricsResponse, error) {
	return f.dbResp, nil
}

func (f *fakeClient) GetManagedLoadBalancerMetrics(context.Context, string, string) (MetricsResponse, error) {
	return f.lbResp, nil
}

func TestScrapeMetricsManagedDatabase(t *testing.T) {
	cfg := &Config{
		CollectionInterval: 60,
		InitialDelay:       0,
		API:                APIConfig{Endpoint: "https://api.upcloud.com", Token: "token", Timeout: 10},
		ManagedDatabases: ManagedDatabaseConfig{
			Enabled: true,
			UUIDs:   []string{"db-uuid"},
			Period:  "5m",
		},
	}

	client := &fakeClient{
		dbResp: MetricsResponse{
			"cpu_usage": {
				Hints: MetricsHints{Title: "CPU usage %"},
				Data: MetricsData{
					Cols: []MetricsColumn{
						{Label: "time", Type: "date"},
						{Label: "primary", Type: "number"},
						{Label: "replica", Type: "number"},
					},
					Rows: [][]any{
						{"2026-02-21T08:00:00Z", 2.2, 2.5},
					},
				},
			},
		},
	}

	metrics, err := scrapeMetrics(context.Background(), client, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("unexpected scrape error: %v", err)
	}

	if metrics.ResourceMetrics().Len() != 1 {
		t.Fatalf("expected 1 resource metrics, got %d", metrics.ResourceMetrics().Len())
	}

	rm := metrics.ResourceMetrics().At(0)
	sm := rm.ScopeMetrics().At(0)
	if sm.Metrics().Len() != 1 {
		t.Fatalf("expected 1 metric, got %d", sm.Metrics().Len())
	}

	m := sm.Metrics().At(0)
	if m.Name() != "upcloud.managed_database.cpu.utilization" {
		t.Fatalf("unexpected metric name: %s", m.Name())
	}
	if m.Unit() != "1" {
		t.Fatalf("unexpected metric unit: %s", m.Unit())
	}

	if m.Gauge().DataPoints().Len() != 2 {
		t.Fatalf("expected 2 datapoints (primary + replica), got %d", m.Gauge().DataPoints().Len())
	}

	first := m.Gauge().DataPoints().At(0)
	if math.Abs(first.DoubleValue()-0.022) > 0.0000001 {
		t.Fatalf("expected normalized value 0.022, got %f", first.DoubleValue())
	}
}
