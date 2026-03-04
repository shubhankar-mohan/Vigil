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

type PromClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

func NewPromClient(baseURL, username, password string) *PromClient {
	return &PromClient{
		baseURL:    baseURL,
		username:   username,
		password:   password,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *PromClient) doGet(rawURL string) (*http.Response, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	return c.httpClient.Do(req)
}

// promResponse matches Prometheus HTTP API response format
type promResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string           `json:"resultType"`
		Result     []promResult     `json:"result"`
	} `json:"data"`
	Error string `json:"error"`
}

type promResult struct {
	Metric map[string]string `json:"metric"`
	Value  [2]interface{}    `json:"value"` // [timestamp, "value"]
}

// QueryInstant runs an instant PromQL query and returns the first scalar value.
func (c *PromClient) QueryInstant(query string) (float64, time.Time, error) {
	u, err := url.Parse(c.baseURL + "/api/v1/query")
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("parse url: %w", err)
	}

	q := u.Query()
	q.Set("query", query)
	u.RawQuery = q.Encode()

	resp, err := c.doGet(u.String())
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("prometheus query: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("read response: %w", err)
	}

	var pr promResponse
	if err := json.Unmarshal(body, &pr); err != nil {
		return 0, time.Time{}, fmt.Errorf("unmarshal: %w", err)
	}

	if pr.Status != "success" {
		return 0, time.Time{}, fmt.Errorf("prometheus error: %s", pr.Error)
	}

	if len(pr.Data.Result) == 0 {
		return 0, time.Time{}, fmt.Errorf("no results for query: %s", query)
	}

	// Extract value — pr.Data.Result[0].Value is [timestamp, "stringValue"]
	valStr, ok := pr.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, time.Time{}, fmt.Errorf("unexpected value type")
	}
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("parse float: %w", err)
	}

	// Extract timestamp
	tsFloat, ok := pr.Data.Result[0].Value[0].(float64)
	if !ok {
		return 0, time.Time{}, fmt.Errorf("unexpected timestamp type")
	}
	ts := time.Unix(int64(tsFloat), 0)

	return val, ts, nil
}
