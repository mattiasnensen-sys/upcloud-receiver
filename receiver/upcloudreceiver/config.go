// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloudreceiver

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/collector/config/configopaque"
)

const (
	defaultAPIEndpoint                 = "https://api.upcloud.com"
	defaultCollectionInterval          = 60 * time.Second
	defaultInitialDelay                = 1 * time.Second
	defaultAPITimeout                  = 10 * time.Second
	defaultManagedDatabasePeriod       = "5m"
	defaultManagedLoadBalancerPeriod   = "5m"
	defaultLoadBalancerMetricsTemplate = "/1.3/load-balancer/{uuid}/metrics"
)

// Config defines the upcloud receiver settings.
type Config struct {
	CollectionInterval   time.Duration             `mapstructure:"collection_interval"`
	InitialDelay         time.Duration             `mapstructure:"initial_delay"`
	API                  APIConfig                 `mapstructure:"api"`
	ManagedDatabases     ManagedDatabaseConfig     `mapstructure:"managed_databases"`
	ManagedLoadBalancers ManagedLoadBalancerConfig `mapstructure:"managed_load_balancers"`
}

// APIConfig defines authentication and endpoint settings.
type APIConfig struct {
	Endpoint string              `mapstructure:"endpoint"`
	Token    configopaque.String `mapstructure:"token"`
	Timeout  time.Duration       `mapstructure:"timeout"`
}

// ManagedDatabaseConfig configures database metrics scraping.
type ManagedDatabaseConfig struct {
	Enabled bool     `mapstructure:"enabled"`
	UUIDs   []string `mapstructure:"uuids"`
	Period  string   `mapstructure:"period"`
	Metrics []string `mapstructure:"metrics"`
}

// ManagedLoadBalancerConfig configures load balancer metrics scraping.
type ManagedLoadBalancerConfig struct {
	Enabled             bool     `mapstructure:"enabled"`
	UUIDs               []string `mapstructure:"uuids"`
	Period              string   `mapstructure:"period"`
	Metrics             []string `mapstructure:"metrics"`
	MetricsPathTemplate string   `mapstructure:"metrics_path_template"`
}

// Validate validates receiver configuration.
func (cfg *Config) Validate() error {
	if cfg.CollectionInterval <= 0 {
		return fmt.Errorf("collection_interval must be > 0")
	}
	if cfg.InitialDelay < 0 {
		return fmt.Errorf("initial_delay must be >= 0")
	}
	if strings.TrimSpace(cfg.API.Endpoint) == "" {
		return fmt.Errorf("api.endpoint is required")
	}
	if _, err := url.ParseRequestURI(cfg.API.Endpoint); err != nil {
		return fmt.Errorf("api.endpoint is invalid: %w", err)
	}
	if strings.TrimSpace(string(cfg.API.Token)) == "" {
		return fmt.Errorf("api.token is required")
	}
	if cfg.API.Timeout <= 0 {
		return fmt.Errorf("api.timeout must be > 0")
	}
	if !cfg.ManagedDatabases.Enabled && !cfg.ManagedLoadBalancers.Enabled {
		return fmt.Errorf("at least one managed service block must be enabled")
	}
	if cfg.ManagedDatabases.Enabled && len(cfg.ManagedDatabases.UUIDs) == 0 {
		return fmt.Errorf("managed_databases.uuids must be set when managed_databases.enabled=true")
	}
	if cfg.ManagedLoadBalancers.Enabled && len(cfg.ManagedLoadBalancers.UUIDs) == 0 {
		return fmt.Errorf("managed_load_balancers.uuids must be set when managed_load_balancers.enabled=true")
	}
	if cfg.ManagedLoadBalancers.Enabled && !strings.Contains(cfg.ManagedLoadBalancers.MetricsPathTemplate, "{uuid}") {
		return fmt.Errorf("managed_load_balancers.metrics_path_template must contain {uuid}")
	}
	return nil
}
