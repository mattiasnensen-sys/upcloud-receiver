// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloudreceiver

import (
	"context"
	"sync"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

type metricsReceiver struct {
	cfg      *Config
	settings receiver.Settings
	next     consumer.Metrics
	client   Client

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newMetricsReceiver(cfg *Config, settings receiver.Settings, next consumer.Metrics, client Client) receiver.Metrics {
	return &metricsReceiver{
		cfg:      cfg,
		settings: settings,
		next:     next,
		client:   client,
	}
}

func (r *metricsReceiver) Start(_ context.Context, _ component.Host) error {
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.run(ctx)
	}()
	return nil
}

func (r *metricsReceiver) Shutdown(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (r *metricsReceiver) run(ctx context.Context) {
	if r.cfg.InitialDelay > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(r.cfg.InitialDelay):
		}
	}

	// Immediate first scrape after initial delay.
	r.scrapeAndConsume(ctx)

	ticker := time.NewTicker(r.cfg.CollectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.scrapeAndConsume(ctx)
		}
	}
}

func (r *metricsReceiver) scrapeAndConsume(ctx context.Context) {
	metrics, err := scrapeMetrics(ctx, r.client, r.cfg, r.settings.Logger)
	if err != nil {
		r.settings.Logger.Error("UpCloud scrape failed", zap.Error(err))
		return
	}
	if metrics.ResourceMetrics().Len() == 0 {
		return
	}
	if err := r.next.ConsumeMetrics(ctx, metrics); err != nil {
		r.settings.Logger.Error("Failed to consume UpCloud metrics", zap.Error(err))
	}
}
