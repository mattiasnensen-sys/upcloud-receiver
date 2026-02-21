// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloudreceiver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

func TestHTTPClientIntegration_BearerTokenFromFile(t *testing.T) {
	tokenFile := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFile, []byte("fixture-token\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	dbFixture := mustReadFixture(t, "testdata/integration/managed_database_metrics.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer fixture-token" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		if got := r.URL.Query().Get("period"); got != "5m" {
			t.Fatalf("unexpected period query: %q", got)
		}
		if r.URL.Path != "/1.3/database/db-uuid/metrics" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(dbFixture)
	}))
	defer server.Close()

	client, err := NewHTTPClient(APIConfig{
		Endpoint:  server.URL,
		TokenFile: tokenFile,
		Timeout:   2 * time.Second,
	}, defaultLoadBalancerMetricsTemplate)
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}

	if _, err := client.GetManagedDatabaseMetrics(context.Background(), "db-uuid", "5m"); err != nil {
		t.Fatalf("get managed database metrics: %v", err)
	}
}

func TestHTTPClientIntegration_ListManagedDatabaseServiceUUIDs(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1.3/database" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		calls = append(calls, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("offset") {
		case "0":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"uuid": "db-1"},
				{"uuid": "db-2"},
			})
		default:
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		}
	}))
	defer server.Close()

	client, err := NewHTTPClient(APIConfig{
		Endpoint: server.URL,
		Token:    "fixture-token",
		Timeout:  2 * time.Second,
	}, defaultLoadBalancerMetricsTemplate)
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}

	ids, err := client.ListManagedDatabaseServiceUUIDs(context.Background(), "/1.3/database", 2)
	if err != nil {
		t.Fatalf("list managed database uuids: %v", err)
	}
	if len(ids) != 2 || ids[0] != "db-1" || ids[1] != "db-2" {
		t.Fatalf("unexpected discovered ids: %v", ids)
	}
	if len(calls) != 2 {
		t.Fatalf("expected two paginated calls, got %d", len(calls))
	}
}

func TestHTTPClientIntegration_ListManagedLoadBalancerUUIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1.3/load-balancer" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"load_balancers": []map[string]any{
				{"uuid": "lb-2"},
				{"uuid": "lb-1"},
			},
		})
	}))
	defer server.Close()

	client, err := NewHTTPClient(APIConfig{
		Endpoint: server.URL,
		Token:    "fixture-token",
		Timeout:  2 * time.Second,
	}, defaultLoadBalancerMetricsTemplate)
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}

	ids, err := client.ListManagedLoadBalancerUUIDs(context.Background(), "/1.3/load-balancer")
	if err != nil {
		t.Fatalf("list managed load balancer uuids: %v", err)
	}
	if len(ids) != 2 || ids[0] != "lb-1" || ids[1] != "lb-2" {
		t.Fatalf("unexpected discovered ids: %v", ids)
	}
}

func TestHTTPClientIntegration_LoadBalancerSnapshotConversion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/1.3/load-balancer/lb-uuid/metrics" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"frontends": []map[string]any{
				{
					"name":                "https-443",
					"updated_at":          "2026-02-21T12:01:47.746303Z",
					"total_http_requests": 12,
					"request_rate":        2,
				},
			},
			"backends": []map[string]any{
				{
					"name":                "api-backend",
					"updated_at":          "2026-02-21T12:01:47.746303Z",
					"current_sessions":    3,
					"total_request_bytes": 1024,
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewHTTPClient(APIConfig{
		Endpoint: server.URL,
		Token:    "fixture-token",
		Timeout:  2 * time.Second,
	}, "/1.3/load-balancer/{uuid}/metrics")
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}

	metrics, err := client.GetManagedLoadBalancerMetrics(context.Background(), "lb-uuid", "hour")
	if err != nil {
		t.Fatalf("get managed load balancer metrics: %v", err)
	}

	if len(metrics) == 0 {
		t.Fatalf("expected converted metrics from load balancer snapshot")
	}
	if _, ok := metrics["frontend.total_http_requests"]; !ok {
		t.Fatalf("expected frontend.total_http_requests metric key")
	}
	if _, ok := metrics["backend.current_sessions"]; !ok {
		t.Fatalf("expected backend.current_sessions metric key")
	}
}

func TestHTTPClientIntegration_BasicAuthFromPasswordFile(t *testing.T) {
	passwordFile := filepath.Join(t.TempDir(), "password")
	if err := os.WriteFile(passwordFile, []byte("fixture-password\n"), 0o600); err != nil {
		t.Fatalf("write password file: %v", err)
	}

	lbFixture := mustReadFixture(t, "testdata/integration/managed_load_balancer_metrics.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Fatalf("expected basic auth")
		}
		if user != "fixture-user" || pass != "fixture-password" {
			t.Fatalf("unexpected basic auth credentials: %s/%s", user, pass)
		}
		if got := r.URL.Query().Get("period"); got != "10m" {
			t.Fatalf("unexpected period query: %q", got)
		}
		if r.URL.Path != "/1.3/load-balancer/lb-uuid/metrics" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(lbFixture)
	}))
	defer server.Close()

	client, err := NewHTTPClient(APIConfig{
		Endpoint:     server.URL,
		Username:     "fixture-user",
		PasswordFile: passwordFile,
		Timeout:      2 * time.Second,
	}, "/1.3/load-balancer/{uuid}/metrics")
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}

	if _, err := client.GetManagedLoadBalancerMetrics(context.Background(), "lb-uuid", "10m"); err != nil {
		t.Fatalf("get managed load balancer metrics: %v", err)
	}
}

func TestScrapeMetricsIntegration_DatabaseAndLoadBalancer(t *testing.T) {
	dbFixture := mustReadFixture(t, "testdata/integration/managed_database_metrics.json")
	lbFixture := mustReadFixture(t, "testdata/integration/managed_load_balancer_metrics.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/1.3/database/"):
			_, _ = w.Write(dbFixture)
		case strings.HasPrefix(r.URL.Path, "/1.3/load-balancer/"):
			_, _ = w.Write(lbFixture)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := &Config{
		CollectionInterval: 10 * time.Second,
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
		ManagedLoadBalancers: ManagedLoadBalancerConfig{
			Enabled:             true,
			UUIDs:               []string{"lb-uuid"},
			Period:              "5m",
			MetricsPathTemplate: "/1.3/load-balancer/{uuid}/metrics",
		},
	}

	client, err := NewHTTPClient(cfg.API, cfg.ManagedLoadBalancers.MetricsPathTemplate)
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}

	metrics, err := scrapeMetrics(context.Background(), client, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("scrape metrics: %v", err)
	}

	names := allMetricNames(metrics)
	sort.Strings(names)

	want := []string{
		"upcloud.managed_database.cpu.utilization",
		"upcloud.managed_database.disk.io.read_operations",
		"upcloud.managed_load_balancer.backend.connections",
		"upcloud.managed_load_balancer.cpu.utilization",
	}
	sort.Strings(want)

	if len(names) != len(want) {
		t.Fatalf("unexpected metric count: got=%d want=%d names=%v", len(names), len(want), names)
	}

	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("unexpected metric names: got=%v want=%v", names, want)
		}
	}
}

func TestScrapeMetricsIntegration_AutoDiscover(t *testing.T) {
	dbFixture := mustReadFixture(t, "testdata/integration/managed_database_metrics.json")
	lbFixture := mustReadFixture(t, "testdata/integration/managed_load_balancer_metrics.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/1.3/database":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"uuid": "db-uuid"},
			})
		case "/1.3/load-balancer":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"load_balancers": []map[string]any{
					{"uuid": "lb-uuid"},
				},
			})
		case "/1.3/database/db-uuid/metrics":
			_, _ = w.Write(dbFixture)
		case "/1.3/load-balancer/lb-uuid/metrics":
			_, _ = w.Write(lbFixture)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cfg := &Config{
		CollectionInterval: 10 * time.Second,
		InitialDelay:       0,
		API: APIConfig{
			Endpoint: server.URL,
			Token:    "fixture-token",
			Timeout:  2 * time.Second,
		},
		ManagedDatabases: ManagedDatabaseConfig{
			Enabled:        true,
			AutoDiscover:   true,
			DiscoveryPath:  "/1.3/database",
			DiscoveryLimit: 100,
			Period:         "5m",
		},
		ManagedLoadBalancers: ManagedLoadBalancerConfig{
			Enabled:             true,
			AutoDiscover:        true,
			DiscoveryPath:       "/1.3/load-balancer",
			Period:              "5m",
			MetricsPathTemplate: "/1.3/load-balancer/{uuid}/metrics",
		},
	}

	client, err := NewHTTPClient(cfg.API, cfg.ManagedLoadBalancers.MetricsPathTemplate)
	if err != nil {
		t.Fatalf("new http client: %v", err)
	}

	metrics, err := scrapeMetrics(context.Background(), client, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("scrape metrics: %v", err)
	}

	names := allMetricNames(metrics)
	if len(names) == 0 {
		t.Fatalf("expected discovered metrics")
	}
}

func TestNewHTTPClient_InvalidCredentialFile(t *testing.T) {
	_, err := NewHTTPClient(APIConfig{
		Endpoint:  "https://api.upcloud.com",
		TokenFile: "/non-existent/token",
		Timeout:   2 * time.Second,
	}, defaultLoadBalancerMetricsTemplate)
	if err == nil {
		t.Fatalf("expected error for missing credential file")
	}
}

func mustReadFixture(t *testing.T, relativePath string) []byte {
	t.Helper()
	b, err := os.ReadFile(relativePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", relativePath, err)
	}
	return b
}

func allMetricNames(metrics pmetric.Metrics) []string {
	names := make([]string, 0)
	rms := metrics.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		sms := rms.At(i).ScopeMetrics()
		for j := 0; j < sms.Len(); j++ {
			ms := sms.At(j).Metrics()
			for k := 0; k < ms.Len(); k++ {
				names = append(names, ms.At(k).Name())
			}
		}
	}
	return names
}
