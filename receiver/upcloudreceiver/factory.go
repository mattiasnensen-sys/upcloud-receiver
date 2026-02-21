// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloudreceiver // import "github.com/upcloud-community/opentelemetry-upcloud-receiver/receiver/upcloudreceiver"

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"

	"github.com/upcloud-community/opentelemetry-upcloud-receiver/receiver/upcloudreceiver/internal/metadata"
)

// NewFactory creates a factory for upcloud receiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, metadata.MetricsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		CollectionInterval: defaultCollectionInterval,
		InitialDelay:       defaultInitialDelay,
		API: APIConfig{
			Endpoint: defaultAPIEndpoint,
			Timeout:  defaultAPITimeout,
		},
		ManagedDatabases: ManagedDatabaseConfig{
			Enabled: true,
			Period:  defaultManagedDatabasePeriod,
		},
		ManagedLoadBalancers: ManagedLoadBalancerConfig{
			Enabled:             false,
			Period:              defaultManagedLoadBalancerPeriod,
			MetricsPathTemplate: defaultLoadBalancerMetricsTemplate,
		},
	}
}

func createMetricsReceiver(
	_ context.Context,
	settings receiver.Settings,
	baseCfg component.Config,
	next consumer.Metrics,
) (receiver.Metrics, error) {
	cfg := baseCfg.(*Config)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	client, err := NewHTTPClient(cfg.API, cfg.ManagedLoadBalancers.MetricsPathTemplate)
	if err != nil {
		return nil, err
	}
	return newMetricsReceiver(cfg, settings, next, client), nil
}
