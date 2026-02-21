// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloudreceiver

import "testing"

func TestDescriptorForMetric_KnownManagedDatabaseMetric(t *testing.T) {
	d := descriptorForMetric(resourceTypeManagedDatabase, "cpu_usage")
	if d.Name != "upcloud.managed_database.cpu.utilization" {
		t.Fatalf("unexpected name: %s", d.Name)
	}
	if d.Unit != "1" {
		t.Fatalf("unexpected unit: %s", d.Unit)
	}
	if !d.PercentToRatio {
		t.Fatalf("expected percent-to-ratio normalization")
	}
}

func TestDescriptorForMetric_UsageFallback(t *testing.T) {
	d := descriptorForMetric(resourceTypeManagedLoadBalancer, "frontend_usage")
	if d.Name != "upcloud.managed_load_balancer.frontend.utilization" {
		t.Fatalf("unexpected name: %s", d.Name)
	}
	if !d.PercentToRatio {
		t.Fatalf("expected percent-to-ratio normalization")
	}
}

func TestDescriptorForMetric_GenericFallback(t *testing.T) {
	d := descriptorForMetric(resourceTypeManagedLoadBalancer, "backend-connections.total")
	if d.Name != "upcloud.managed_load_balancer.backend.connections.total" {
		t.Fatalf("unexpected name: %s", d.Name)
	}
	if d.Unit != "1" {
		t.Fatalf("unexpected unit: %s", d.Unit)
	}
}
