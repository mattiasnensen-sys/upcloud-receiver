// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloudreceiver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

const instrumentationScopeName = "github.com/upcloud-community/opentelemetry-upcloud-receiver/receiver/upcloudreceiver"

const (
	resourceTypeManagedDatabase     = "managed_database"
	resourceTypeManagedLoadBalancer = "managed_load_balancer"
)

func scrapeMetrics(ctx context.Context, client Client, cfg *Config, logger *zap.Logger) (pmetric.Metrics, error) {
	out := pmetric.NewMetrics()
	var errs []error

	if cfg.ManagedDatabases.Enabled {
		targetUUIDs, err := resolveManagedDatabaseUUIDs(ctx, client, cfg.ManagedDatabases)
		if err != nil {
			errs = append(errs, err)
		}
		for _, uuid := range targetUUIDs {
			resp, err := client.GetManagedDatabaseMetrics(ctx, uuid, cfg.ManagedDatabases.Period)
			if err != nil {
				errs = append(errs, fmt.Errorf("managed database %s: %w", uuid, err))
				continue
			}
			appendMetricsPayload(out, resp, resourceTypeManagedDatabase, uuid, cfg.ManagedDatabases.Metrics, logger)
		}
	}

	if cfg.ManagedLoadBalancers.Enabled {
		targetUUIDs, err := resolveManagedLoadBalancerUUIDs(ctx, client, cfg.ManagedLoadBalancers)
		if err != nil {
			errs = append(errs, err)
		}
		for _, uuid := range targetUUIDs {
			resp, err := client.GetManagedLoadBalancerMetrics(ctx, uuid, cfg.ManagedLoadBalancers.Period)
			if err != nil {
				errs = append(errs, fmt.Errorf("managed load balancer %s: %w", uuid, err))
				continue
			}
			appendMetricsPayload(out, resp, resourceTypeManagedLoadBalancer, uuid, cfg.ManagedLoadBalancers.Metrics, logger)
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
	descriptor := descriptorForMetric(resourceType, metricKey)

	m := dest.AppendEmpty()
	m.SetName(descriptor.Name)
	m.SetDescription(metric.Hints.Title)
	m.SetUnit(descriptor.Unit)
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
		value = descriptor.normalizeValue(value)

		dp := g.AppendEmpty()
		dp.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
		dp.SetDoubleValue(value)
		dp.Attributes().PutStr("upcloud.metric.name", metricKey)
		dp.Attributes().PutStr("upcloud.series", metric.Data.Cols[idx].Label)
		if descriptor.PercentToRatio {
			dp.Attributes().PutStr("upcloud.value.normalization", "percent_to_ratio")
		}
	}
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

func resolveManagedDatabaseUUIDs(ctx context.Context, client Client, cfg ManagedDatabaseConfig) ([]string, error) {
	targets := append([]string(nil), cfg.UUIDs...)
	if cfg.AutoDiscover {
		discovered, err := client.ListManagedDatabaseServiceUUIDs(ctx, cfg.DiscoveryPath, cfg.DiscoveryLimit)
		if err != nil {
			return applyExcludeUUIDs(targets, cfg.ExcludeUUIDs), fmt.Errorf("discover managed databases: %w", err)
		}
		targets = append(targets, discovered...)
	}
	return applyExcludeUUIDs(targets, cfg.ExcludeUUIDs), nil
}

func resolveManagedLoadBalancerUUIDs(ctx context.Context, client Client, cfg ManagedLoadBalancerConfig) ([]string, error) {
	targets := append([]string(nil), cfg.UUIDs...)
	if cfg.AutoDiscover {
		discovered, err := client.ListManagedLoadBalancerUUIDs(ctx, cfg.DiscoveryPath)
		if err != nil {
			return applyExcludeUUIDs(targets, cfg.ExcludeUUIDs), fmt.Errorf("discover managed load balancers: %w", err)
		}
		targets = append(targets, discovered...)
	}
	return applyExcludeUUIDs(targets, cfg.ExcludeUUIDs), nil
}

func applyExcludeUUIDs(targets []string, exclude []string) []string {
	targets = dedupe(targets)
	if len(targets) == 0 {
		return targets
	}

	excluded := make(map[string]struct{}, len(exclude))
	for _, id := range exclude {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		excluded[trimmed] = struct{}{}
	}

	filtered := make([]string, 0, len(targets))
	for _, id := range targets {
		if _, skip := excluded[id]; skip {
			continue
		}
		filtered = append(filtered, id)
	}
	sort.Strings(filtered)
	return filtered
}
