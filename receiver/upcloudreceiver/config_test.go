// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloudreceiver

import "testing"

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid database config",
			cfg: Config{
				CollectionInterval: 30,
				InitialDelay:       1,
				API: APIConfig{
					Endpoint: "https://api.upcloud.com",
					Token:    "token",
					Timeout:  10,
				},
				ManagedDatabases: ManagedDatabaseConfig{
					Enabled: true,
					UUIDs:   []string{"db-uuid"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid token file auth",
			cfg: Config{
				CollectionInterval: 30,
				InitialDelay:       1,
				API: APIConfig{
					Endpoint:  "https://api.upcloud.com",
					TokenFile: "/tmp/upcloud-token",
					Timeout:   10,
				},
				ManagedDatabases: ManagedDatabaseConfig{
					Enabled: true,
					UUIDs:   []string{"db-uuid"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid basic auth",
			cfg: Config{
				CollectionInterval: 30,
				InitialDelay:       1,
				API: APIConfig{
					Endpoint: "https://api.upcloud.com",
					Username: "user",
					Password: "pass",
					Timeout:  10,
				},
				ManagedDatabases: ManagedDatabaseConfig{
					Enabled: true,
					UUIDs:   []string{"db-uuid"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing token",
			cfg: Config{
				CollectionInterval: 30,
				API: APIConfig{
					Endpoint: "https://api.upcloud.com",
					Timeout:  10,
				},
				ManagedDatabases: ManagedDatabaseConfig{Enabled: true, UUIDs: []string{"db-uuid"}},
			},
			wantErr: true,
		},
		{
			name: "invalid both token and token file",
			cfg: Config{
				CollectionInterval: 30,
				API: APIConfig{
					Endpoint:  "https://api.upcloud.com",
					Token:     "token",
					TokenFile: "/tmp/upcloud-token",
					Timeout:   10,
				},
				ManagedDatabases: ManagedDatabaseConfig{Enabled: true, UUIDs: []string{"db-uuid"}},
			},
			wantErr: true,
		},
		{
			name: "invalid bearer and basic mixed",
			cfg: Config{
				CollectionInterval: 30,
				API: APIConfig{
					Endpoint: "https://api.upcloud.com",
					Token:    "token",
					Username: "user",
					Password: "pass",
					Timeout:  10,
				},
				ManagedDatabases: ManagedDatabaseConfig{Enabled: true, UUIDs: []string{"db-uuid"}},
			},
			wantErr: true,
		},
		{
			name: "invalid basic missing username",
			cfg: Config{
				CollectionInterval: 30,
				API: APIConfig{
					Endpoint: "https://api.upcloud.com",
					Password: "pass",
					Timeout:  10,
				},
				ManagedDatabases: ManagedDatabaseConfig{Enabled: true, UUIDs: []string{"db-uuid"}},
			},
			wantErr: true,
		},
		{
			name: "enabled database without uuids",
			cfg: Config{
				CollectionInterval: 30,
				API:                APIConfig{Endpoint: "https://api.upcloud.com", Token: "token", Timeout: 10},
				ManagedDatabases:   ManagedDatabaseConfig{Enabled: true},
			},
			wantErr: true,
		},
		{
			name: "invalid load balancer template",
			cfg: Config{
				CollectionInterval: 30,
				API:                APIConfig{Endpoint: "https://api.upcloud.com", Token: "token", Timeout: 10},
				ManagedLoadBalancers: ManagedLoadBalancerConfig{
					Enabled:             true,
					UUIDs:               []string{"lb-uuid"},
					MetricsPathTemplate: "/1.3/load-balancer/static/metrics",
				},
			},
			wantErr: true,
		},
		{
			name: "no resources enabled",
			cfg: Config{
				CollectionInterval: 30,
				API:                APIConfig{Endpoint: "https://api.upcloud.com", Token: "token", Timeout: 10},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
