// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloudreceiver

import (
	"fmt"
	"regexp"
	"strings"
)

var invalidMetricChars = regexp.MustCompile(`[^a-z0-9]+`)

type metricDescriptor struct {
	Name           string
	Unit           string
	PercentToRatio bool
}

var managedDatabaseMetricDescriptors = map[string]metricDescriptor{
	"cpu_usage": {
		Name:           "upcloud.managed_database.cpu.utilization",
		Unit:           "1",
		PercentToRatio: true,
	},
	"mem_usage": {
		Name:           "upcloud.managed_database.memory.utilization",
		Unit:           "1",
		PercentToRatio: true,
	},
	"disk_usage": {
		Name:           "upcloud.managed_database.disk.utilization",
		Unit:           "1",
		PercentToRatio: true,
	},
	"load_average": {
		Name: "upcloud.managed_database.system.load_average",
		Unit: "1",
	},
	"diskio_reads": {
		Name: "upcloud.managed_database.disk.io.read_operations",
		Unit: "{operation}/s",
	},
	"diskio_writes": {
		Name: "upcloud.managed_database.disk.io.write_operations",
		Unit: "{operation}/s",
	},
	"net_receive": {
		Name: "upcloud.managed_database.network.receive",
		Unit: "By/s",
	},
	"net_send": {
		Name: "upcloud.managed_database.network.transmit",
		Unit: "By/s",
	},
}

var managedLoadBalancerMetricDescriptors = map[string]metricDescriptor{
	"cpu_usage": {
		Name:           "upcloud.managed_load_balancer.cpu.utilization",
		Unit:           "1",
		PercentToRatio: true,
	},
	"mem_usage": {
		Name:           "upcloud.managed_load_balancer.memory.utilization",
		Unit:           "1",
		PercentToRatio: true,
	},
}

func descriptorForMetric(resourceType string, metricKey string) metricDescriptor {
	metricKey = strings.TrimSpace(metricKey)
	if resourceType == resourceTypeManagedDatabase {
		if descriptor, ok := managedDatabaseMetricDescriptors[metricKey]; ok {
			return descriptor
		}
	}
	if resourceType == resourceTypeManagedLoadBalancer {
		if descriptor, ok := managedLoadBalancerMetricDescriptors[metricKey]; ok {
			return descriptor
		}
	}

	if strings.HasSuffix(metricKey, "_usage") {
		base := strings.TrimSuffix(metricKey, "_usage")
		return metricDescriptor{
			Name:           fmt.Sprintf("upcloud.%s.%s.utilization", resourceType, sanitizeMetricPath(base)),
			Unit:           "1",
			PercentToRatio: true,
		}
	}

	return metricDescriptor{
		Name: fmt.Sprintf("upcloud.%s.%s", resourceType, sanitizeMetricPath(metricKey)),
		Unit: "1",
	}
}

func sanitizeMetricPath(metricKey string) string {
	normalized := strings.ToLower(metricKey)
	normalized = invalidMetricChars.ReplaceAllString(normalized, ".")
	normalized = strings.Trim(normalized, ".")
	normalized = strings.ReplaceAll(normalized, "..", ".")
	if normalized == "" {
		return "unknown"
	}
	return normalized
}

func (d metricDescriptor) normalizeValue(value float64) float64 {
	if d.PercentToRatio {
		return value / 100.0
	}
	return value
}
