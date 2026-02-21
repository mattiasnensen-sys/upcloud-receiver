// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloudreceiver

import "testing"

func TestCreateDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	if cfg.API.Endpoint == "" {
		t.Fatalf("default api endpoint must be set")
	}
	if cfg.CollectionInterval <= 0 {
		t.Fatalf("default collection interval must be > 0")
	}
	if !cfg.ManagedDatabases.Enabled {
		t.Fatalf("managed_databases should be enabled by default")
	}
	if !cfg.ManagedDatabases.AutoDiscover {
		t.Fatalf("managed_databases auto_discover should be enabled by default")
	}
}
