package grafana

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	apiToken   string
	httpClient *http.Client
}

func NewClient(baseURL, apiToken string) *Client {
	if baseURL == "" || apiToken == "" {
		return nil
	}
	return &Client{
		baseURL:    baseURL,
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

type annotation struct {
	Text string   `json:"text"`
	Tags []string `json:"tags"`
	Time int64    `json:"time"` // epoch ms
}

// PushStateChange sends an annotation to Grafana when a switch changes state.
func (c *Client) PushStateChange(switchName, oldState, newState, details string) {
	if c == nil {
		return
	}

	text := fmt.Sprintf("Switch **%s**: %s → %s\n\n%s", switchName, oldState, newState, details)

	a := annotation{
		Text: text,
		Tags: []string{"vigil", "dms", switchName, newState},
		Time: time.Now().UnixMilli(),
	}

	body, err := json.Marshal(a)
	if err != nil {
		log.Printf("grafana: marshal annotation: %v", err)
		return
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/annotations", bytes.NewReader(body))
	if err != nil {
		log.Printf("grafana: create request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("grafana: push annotation: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("grafana: annotation returned status %d", resp.StatusCode)
	}
}
