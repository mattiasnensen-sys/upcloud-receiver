// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloudreceiver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

const instrumentationScopeName = "github.com/upcloud-community/opentelemetry-upcloud-receiver/receiver/upcloudreceiver"

func scrapeMetrics(ctx context.Context, client Client, cfg *Config, logger *zap.Logger) (pmetric.Metrics, error) {
	out := pmetric.NewMetrics()
	var errs []error

	if cfg.ManagedDatabases.Enabled {
		for _, uuid := range cfg.ManagedDatabases.UUIDs {
			resp, err := client.GetManagedDatabaseMetrics(ctx, uuid, cfg.ManagedDatabases.Period)
			if err != nil {
				errs = append(errs, fmt.Errorf("managed database %s: %w", uuid, err))
				continue
			}
			appendMetricsPayload(out, resp, "managed_database", uuid, cfg.ManagedDatabases.Metrics, logger)
		}
	}

	if cfg.ManagedLoadBalancers.Enabled {
		for _, uuid := range cfg.ManagedLoadBalancers.UUIDs {
			resp, err := client.GetManagedLoadBalancerMetrics(ctx, uuid, cfg.ManagedLoadBalancers.Period)
			if err != nil {
				errs = append(errs, fmt.Errorf("managed load balancer %s: %w", uuid, err))
				continue
			}
			appendMetricsPayload(out, resp, "managed_load_balancer", uuid, cfg.ManagedLoadBalancers.Metrics, logger)
		}
	}

	if len(errs) > 0 {
		return out, errors.Join(errs...)
	}
	return out, nil
}

func appendMetricsPayload(
	out pmetric.Metrics,
	payload MetricsResponse,
	resourceType string,
	resourceUUID string,
	allowlist []string,
	logger *zap.Logger,
) {
	allowed := toAllowlist(allowlist)

	rm := out.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutStr("cloud.provider", "upcloud")
	rm.Resource().Attributes().PutStr("upcloud.resource.type", resourceType)
	rm.Resource().Attributes().PutStr("upcloud.resource.uuid", resourceUUID)

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName(instrumentationScopeName)
	metrics := sm.Metrics()

	for metricKey, metric := range payload {
		if len(allowed) > 0 {
			if _, ok := allowed[metricKey]; !ok {
				continue
			}
		}
		appendMetric(metricKey, metric, resourceType, metrics, logger)
	}
}

func appendMetric(metricKey string, metric MetricsItem, resourceType string, dest pmetric.MetricSlice, logger *zap.Logger) {
	if len(metric.Data.Cols) < 2 || len(metric.Data.Rows) == 0 {
		return
	}

	row := metric.Data.Rows[len(metric.Data.Rows)-1]
	if len(row) < 2 {
		return
	}

	timestamp := extractTime(row[0])

	m := dest.AppendEmpty()
	m.SetName(buildMetricName(resourceType, metricKey))
	m.SetDescription(metric.Hints.Title)
	m.SetUnit("1")
	m.SetEmptyGauge()
	g := m.Gauge().DataPoints()

	for idx := 1; idx < len(metric.Data.Cols) && idx < len(row); idx++ {
		value, ok := toFloat64(row[idx])
		if !ok {
			logger.Debug("Skipping non-numeric metric value",
				zap.String("metric", metricKey),
				zap.Int("column", idx),
			)
			continue
		}

		dp := g.AppendEmpty()
		dp.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
		dp.SetDoubleValue(value)
		dp.Attributes().PutStr("upcloud.metric.name", metricKey)
		dp.Attributes().PutStr("upcloud.series", metric.Data.Cols[idx].Label)
	}
}

func buildMetricName(resourceType string, metricKey string) string {
	sanitized := sanitizeMetricFragment(metricKey)
	resource := sanitizeMetricFragment(resourceType)
	return fmt.Sprintf("upcloud.%s.%s", resource, sanitized)
}

func sanitizeMetricFragment(s string) string {
	if s == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		";", "_",
		",", "_",
	)
	return replacer.Replace(strings.ToLower(s))
}

func extractTime(v any) time.Time {
	s, ok := v.(string)
	if !ok {
		return nowTimestamp(time.Time{})
	}
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nowTimestamp(time.Time{})
	}
	return parsed.UTC()
}

func toAllowlist(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]struct{}, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		result[trimmed] = struct{}{}
	}
	return result
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case int32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case uint32:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	case string:
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}
