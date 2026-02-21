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
	"sort"
	"strconv"
	"strings"
	"time"
)

// Client fetches metrics from UpCloud managed services APIs.
type Client interface {
	ListManagedDatabaseServiceUUIDs(ctx context.Context, discoveryPath string, limit int) ([]string, error)
	ListManagedLoadBalancerUUIDs(ctx context.Context, discoveryPath string) ([]string, error)
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

func (c *httpClient) ListManagedDatabaseServiceUUIDs(ctx context.Context, discoveryPath string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = defaultDiscoveryLimit
	}

	seen := map[string]struct{}{}
	var discovered []string
	offset := 0
	for {
		query := url.Values{}
		query.Set("limit", strconv.Itoa(limit))
		query.Set("offset", strconv.Itoa(offset))

		payload, _, err := c.getJSON(ctx, discoveryPath, query)
		if err != nil {
			return nil, err
		}

		page := extractUUIDs(payload)
		newItems := 0
		for _, id := range page {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			discovered = append(discovered, id)
			newItems++
		}

		if len(page) < limit || newItems == 0 {
			break
		}
		offset += limit
	}

	sort.Strings(discovered)
	return discovered, nil
}

func (c *httpClient) ListManagedLoadBalancerUUIDs(ctx context.Context, discoveryPath string) ([]string, error) {
	payload, _, err := c.getJSON(ctx, discoveryPath, nil)
	if err != nil {
		return nil, err
	}
	ids := extractUUIDs(payload)
	sort.Strings(ids)
	return dedupeSorted(ids), nil
}

func (c *httpClient) getMetrics(ctx context.Context, endpointPath string, period string) (MetricsResponse, error) {
	query := url.Values{}
	if strings.TrimSpace(period) != "" {
		query.Set("period", period)
	}

	payload, _, err := c.getJSON(ctx, endpointPath, query)
	if err != nil {
		return nil, err
	}

	serialized, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal metrics response: %w", err)
	}

	var parsed MetricsResponse
	if err := json.Unmarshal(serialized, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal metrics response: %w", err)
	}
	return parsed, nil
}

func (c *httpClient) getJSON(ctx context.Context, endpointPath string, query url.Values) (any, http.Header, error) {
	requestURL, err := c.baseURL.Parse(endpointPath)
	if err != nil {
		return nil, nil, fmt.Errorf("build URL: %w", err)
	}
	if query != nil {
		requestURL.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	c.auth.apply(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request %s: %w", endpointPath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, endpointPath)
	}

	var payload any
	decoder := json.NewDecoder(resp.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, nil, fmt.Errorf("decode response: %w", err)
	}
	return payload, resp.Header.Clone(), nil
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

func extractUUIDs(payload any) []string {
	switch root := payload.(type) {
	case []any:
		return extractUUIDsFromArray(root)
	case map[string]any:
		ids := make([]string, 0)
		if uuid, ok := root["uuid"].(string); ok {
			ids = append(ids, strings.TrimSpace(uuid))
		}
		for _, value := range root {
			arr, ok := value.([]any)
			if !ok {
				continue
			}
			ids = append(ids, extractUUIDsFromArray(arr)...)
		}
		return dedupe(ids)
	default:
		return nil
	}
}

func extractUUIDsFromArray(items []any) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		raw, ok := obj["uuid"]
		if !ok {
			continue
		}
		uuid, ok := raw.(string)
		if !ok {
			continue
		}
		uuid = strings.TrimSpace(uuid)
		if uuid == "" {
			continue
		}
		ids = append(ids, uuid)
	}
	return dedupe(ids)
}

func dedupe(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func dedupeSorted(values []string) []string {
	if len(values) <= 1 {
		return values
	}
	out := values[:1]
	for i := 1; i < len(values); i++ {
		if values[i] == values[i-1] {
			continue
		}
		out = append(out, values[i])
	}
	return out
}
