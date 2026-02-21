// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package upcloudreceiver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"
)

// Client fetches metrics from UpCloud managed services APIs.
type Client interface {
	GetManagedDatabaseMetrics(ctx context.Context, uuid string, period string) (MetricsResponse, error)
	GetManagedLoadBalancerMetrics(ctx context.Context, uuid string, period string) (MetricsResponse, error)
}

type httpClient struct {
	baseURL                  *url.URL
	auth                     requestAuth
	client                   *http.Client
	loadBalancerPathTemplate string
}

type requestAuth struct {
	bearerToken string
	username    string
	password    string
}

// NewHTTPClient creates a new UpCloud API client.
func NewHTTPClient(api APIConfig, loadBalancerPathTemplate string) (Client, error) {
	baseURL, err := url.Parse(strings.TrimRight(api.Endpoint, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse api endpoint: %w", err)
	}
	auth, err := resolveRequestAuth(api)
	if err != nil {
		return nil, err
	}
	return &httpClient{
		baseURL:                  baseURL,
		auth:                     auth,
		client:                   &http.Client{Timeout: api.Timeout},
		loadBalancerPathTemplate: loadBalancerPathTemplate,
	}, nil
}

func (c *httpClient) GetManagedDatabaseMetrics(ctx context.Context, uuid string, period string) (MetricsResponse, error) {
	escapedUUID := url.PathEscape(uuid)
	endpointPath := path.Join("/1.3/database", escapedUUID, "metrics")
	return c.getMetrics(ctx, endpointPath, period)
}

func (c *httpClient) GetManagedLoadBalancerMetrics(ctx context.Context, uuid string, period string) (MetricsResponse, error) {
	escapedUUID := url.PathEscape(uuid)
	endpointPath := strings.ReplaceAll(c.loadBalancerPathTemplate, "{uuid}", escapedUUID)
	return c.getMetrics(ctx, endpointPath, period)
}

func (c *httpClient) getMetrics(ctx context.Context, endpointPath string, period string) (MetricsResponse, error) {
	requestURL, err := c.baseURL.Parse(endpointPath)
	if err != nil {
		return nil, fmt.Errorf("build URL: %w", err)
	}

	if strings.TrimSpace(period) != "" {
		query := requestURL.Query()
		query.Set("period", period)
		requestURL.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	c.auth.apply(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var parsed MetricsResponse
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return parsed, nil
}

// MetricsResponse models UpCloud metrics payloads.
type MetricsResponse map[string]MetricsItem

// MetricsItem is one metric entry from the API response.
type MetricsItem struct {
	Data  MetricsData  `json:"data"`
	Hints MetricsHints `json:"hints"`
}

// MetricsData contains columns and rows for one metric.
type MetricsData struct {
	Cols []MetricsColumn `json:"cols"`
	Rows [][]any         `json:"rows"`
}

// MetricsColumn describes a metric column.
type MetricsColumn struct {
	Label string `json:"label"`
	Type  string `json:"type"`
}

// MetricsHints contains optional display metadata from API.
type MetricsHints struct {
	Title string `json:"title"`
}

func nowTimestamp(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}

func (a requestAuth) apply(req *http.Request) {
	if strings.TrimSpace(a.bearerToken) != "" {
		req.Header.Set("Authorization", "Bearer "+a.bearerToken)
		return
	}
	req.SetBasicAuth(a.username, a.password)
}

func resolveRequestAuth(api APIConfig) (requestAuth, error) {
	if token, err := resolveSecret(string(api.Token), api.TokenFile, "api.token", "api.token_file"); err != nil {
		return requestAuth{}, err
	} else if token != "" {
		return requestAuth{bearerToken: token}, nil
	}

	password, err := resolveSecret(string(api.Password), api.PasswordFile, "api.password", "api.password_file")
	if err != nil {
		return requestAuth{}, err
	}
	return requestAuth{
		username: api.Username,
		password: password,
	}, nil
}

func resolveSecret(inlineValue string, filePath string, inlineName string, fileName string) (string, error) {
	value := strings.TrimSpace(inlineValue)
	trimmedFile := strings.TrimSpace(filePath)
	if value != "" && trimmedFile != "" {
		return "", fmt.Errorf("%s and %s are mutually exclusive", inlineName, fileName)
	}
	if value != "" {
		return value, nil
	}
	if trimmedFile == "" {
		return "", nil
	}

	raw, err := os.ReadFile(trimmedFile)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", fileName, err)
	}
	secret := strings.TrimSpace(string(raw))
	if secret == "" {
		return "", fmt.Errorf("%s is empty", fileName)
	}
	return secret, nil
}
