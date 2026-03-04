package evaluator

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type LokiClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

func NewLokiClient(baseURL, username, password string) *LokiClient {
	return &LokiClient{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *LokiClient) doGet(rawURL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	return c.httpClient.Do(req)
}

// lokiResponse matches Loki HTTP API query_range/query response
type lokiResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string           `json:"resultType"`
		Result     []lokiResult     `json:"result"`
	} `json:"data"`
}

type lokiResult struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"` // [[nanosecond_timestamp, log_line], ...]
}

// QueryLastOccurrence queries Loki for the most recent log matching the query.
// Returns the timestamp of the last match.
func (c *LokiClient) QueryLastOccurrence(query string, lookback time.Duration) (*time.Time, error) {
	u, err := url.Parse(c.baseURL + "/loki/api/v1/query_range")
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	now := time.Now()
	q := u.Query()
	q.Set("query", query)
	q.Set("start", fmt.Sprintf("%d", now.Add(-lookback).UnixNano()))
	q.Set("end", fmt.Sprintf("%d", now.UnixNano()))
	q.Set("limit", "1")
	q.Set("direction", "backward")
	u.RawQuery = q.Encode()

	resp, err := c.doGet(u.String())
	if err != nil {
		return nil, fmt.Errorf("loki query: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var lr lokiResponse
	if err := json.Unmarshal(body, &lr); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	if lr.Status != "success" {
		return nil, fmt.Errorf("loki query failed")
	}

	// Find the most recent log entry across all streams
	var latest *time.Time
	for _, result := range lr.Data.Result {
		for _, entry := range result.Values {
			if len(entry) < 1 {
				continue
			}
			nsec, err := strconv.ParseInt(entry[0], 10, 64)
			if err != nil {
				continue
			}
			t := time.Unix(0, nsec)
			if latest == nil || t.After(*latest) {
				latest = &t
			}
		}
	}

	return latest, nil
}

// CountOverTime queries Loki for the count of matching logs in a time range.
func (c *LokiClient) CountOverTime(query string, lookback time.Duration) (int, error) {
	// Wrap query in count_over_time
	countQuery := fmt.Sprintf(`count_over_time(%s [%s])`, query, formatDuration(lookback))

	u, err := url.Parse(c.baseURL + "/loki/api/v1/query")
	if err != nil {
		return 0, fmt.Errorf("parse url: %w", err)
	}

	q := u.Query()
	q.Set("query", countQuery)
	u.RawQuery = q.Encode()

	resp, err := c.doGet(u.String())
	if err != nil {
		return 0, fmt.Errorf("loki query: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read response: %w", err)
	}

	var lr lokiResponse
	if err := json.Unmarshal(body, &lr); err != nil {
		return 0, fmt.Errorf("unmarshal: %w", err)
	}

	if lr.Status != "success" || len(lr.Data.Result) == 0 {
		return 0, nil
	}

	// For instant queries, result contains values
	for _, result := range lr.Data.Result {
		if len(result.Values) > 0 && len(result.Values[0]) > 1 {
			count, err := strconv.Atoi(result.Values[0][1])
			if err == nil {
				return count, nil
			}
		}
	}

	return 0, nil
}

// GetPatterns queries Loki's pattern detection endpoint
func (c *LokiClient) GetPatterns(selector string) ([]PatternResult, error) {
	u, err := url.Parse(c.baseURL + "/loki/api/v1/patterns")
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	q := u.Query()
	q.Set("query", selector)
	u.RawQuery = q.Encode()

	resp, err := c.doGet(u.String())
	if err != nil {
		return nil, fmt.Errorf("loki patterns query: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var patterns []PatternResult
	if err := json.Unmarshal(body, &patterns); err != nil {
		return nil, fmt.Errorf("unmarshal patterns: %w", err)
	}

	return patterns, nil
}

type PatternResult struct {
	Pattern string `json:"pattern"`
	Count   int    `json:"count"`
}

func formatDuration(d time.Duration) string {
	if d >= time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dm", int(d.Minutes()))
}
