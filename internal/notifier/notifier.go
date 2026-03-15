package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"vigil/internal/models"

	"gorm.io/gorm"
)

// StateChange represents a switch state transition
type StateChange struct {
	SwitchID   uint
	SwitchName string
	OldState   string
	NewState   string
	Details    string
	Timestamp  time.Time
}

// Notifier sends alerts on state changes
type Notifier struct {
	db     *gorm.DB
	client *http.Client
}

func New(db *gorm.DB) *Notifier {
	return &Notifier{
		db:     db,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Notify sends alerts to all enabled channels for a state change
func (n *Notifier) Notify(sc StateChange) {
	var channels []models.AlertChannel
	n.db.Where("enabled = ?", true).Find(&channels)

	for _, ch := range channels {
		go n.send(ch, sc)
	}
}

func (n *Notifier) send(ch models.AlertChannel, sc StateChange) {
	var err error

	switch ch.Type {
	case models.ChannelSlack:
		err = n.sendSlack(ch.Config, sc)
	case models.ChannelDiscord:
		err = n.sendDiscord(ch.Config, sc)
	case models.ChannelWebhook:
		err = n.sendWebhook(ch.Config, sc)
	case models.ChannelPagerDuty:
		err = n.sendPagerDuty(ch.Config, sc)
	case models.ChannelTelegram:
		err = n.sendTelegram(ch.Config, sc)
	default:
		err = fmt.Errorf("unknown channel type: %s", ch.Type)
	}

	// Log the alert
	alertLog := models.AlertLog{
		AlertChannelID: ch.ID,
		SwitchID:       sc.SwitchID,
		SwitchName:     sc.SwitchName,
		OldState:       sc.OldState,
		NewState:       sc.NewState,
		Success:        err == nil,
		SentAt:         time.Now(),
	}
	if err != nil {
		alertLog.Error = err.Error()
		log.Printf("alert failed [%s/%s] for switch %q: %v", ch.Type, ch.Name, sc.SwitchName, err)
	} else {
		log.Printf("alert sent [%s/%s] for switch %q: %s -> %s", ch.Type, ch.Name, sc.SwitchName, sc.OldState, sc.NewState)
	}
	n.db.Create(&alertLog)
}

// SendTest sends a test notification to a channel
func (n *Notifier) SendTest(ch models.AlertChannel) error {
	sc := StateChange{
		SwitchName: "test-switch",
		OldState:   "up",
		NewState:   "down",
		Details:    "This is a test alert from Vigil",
		Timestamp:  time.Now(),
	}

	switch ch.Type {
	case models.ChannelSlack:
		return n.sendSlack(ch.Config, sc)
	case models.ChannelDiscord:
		return n.sendDiscord(ch.Config, sc)
	case models.ChannelWebhook:
		return n.sendWebhook(ch.Config, sc)
	case models.ChannelPagerDuty:
		return n.sendPagerDuty(ch.Config, sc)
	case models.ChannelTelegram:
		return n.sendTelegram(ch.Config, sc)
	default:
		return fmt.Errorf("unknown channel type: %s", ch.Type)
	}
}

func stateEmoji(state string) string {
	switch state {
	case "up":
		return "✅"
	case "down":
		return "🔴"
	case "grace":
		return "⚠️"
	default:
		return "ℹ️"
	}
}

func alertMessage(sc StateChange) string {
	return fmt.Sprintf("%s Switch *%s* changed: *%s* → *%s*\n%s",
		stateEmoji(sc.NewState), sc.SwitchName, sc.OldState, sc.NewState, sc.Details)
}

// ── Slack ──

type slackConfig struct {
	WebhookURL string `json:"webhook_url"`
}

func (n *Notifier) sendSlack(cfgJSON string, sc StateChange) error {
	var cfg slackConfig
	if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
		return fmt.Errorf("invalid slack config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("slack webhook_url is required")
	}

	payload := map[string]interface{}{
		"text": alertMessage(sc),
		"blocks": []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": alertMessage(sc),
				},
			},
			{
				"type": "context",
				"elements": []map[string]string{
					{"type": "mrkdwn", "text": fmt.Sprintf("*Time:* %s", sc.Timestamp.Format(time.RFC3339))},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := n.client.Post(cfg.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	respText := string(respBody)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack returned HTTP %d: %s", resp.StatusCode, respText)
	}
	// Slack returns exactly "ok" on success; anything else (e.g. "no_service",
	// "invalid_token", or an HTML redirect page) indicates failure.
	if respText != "ok" {
		return fmt.Errorf("slack rejected the message: %s", respText)
	}
	return nil
}

// ── Discord ──

type discordConfig struct {
	WebhookURL string `json:"webhook_url"`
}

func (n *Notifier) sendDiscord(cfgJSON string, sc StateChange) error {
	var cfg discordConfig
	if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
		return fmt.Errorf("invalid discord config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("discord webhook_url is required")
	}

	color := 0x40c057 // green
	if sc.NewState == "down" {
		color = 0xf03e3e // red
	} else if sc.NewState == "grace" {
		color = 0xfab005 // yellow
	}

	payload := map[string]interface{}{
		"embeds": []map[string]interface{}{
			{
				"title":       fmt.Sprintf("%s %s → %s", stateEmoji(sc.NewState), sc.OldState, sc.NewState),
				"description": fmt.Sprintf("**Switch:** %s\n%s", sc.SwitchName, sc.Details),
				"color":       color,
				"timestamp":   sc.Timestamp.Format(time.RFC3339),
			},
		},
	}
	return n.postJSON(cfg.WebhookURL, payload)
}

// ── Generic Webhook ──

type webhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

func (n *Notifier) sendWebhook(cfgJSON string, sc StateChange) error {
	var cfg webhookConfig
	if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
		return fmt.Errorf("invalid webhook config: %w", err)
	}
	if cfg.URL == "" {
		return fmt.Errorf("webhook url is required")
	}
	if cfg.Method == "" {
		cfg.Method = "POST"
	}

	payload := map[string]interface{}{
		"switch_name": sc.SwitchName,
		"switch_id":   sc.SwitchID,
		"old_state":   sc.OldState,
		"new_state":   sc.NewState,
		"details":     sc.Details,
		"timestamp":   sc.Timestamp.Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(cfg.Method, cfg.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("webhook returned %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// ── PagerDuty ──

type pagerdutyConfig struct {
	RoutingKey string `json:"routing_key"`
}

func (n *Notifier) sendPagerDuty(cfgJSON string, sc StateChange) error {
	var cfg pagerdutyConfig
	if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
		return fmt.Errorf("invalid pagerduty config: %w", err)
	}
	if cfg.RoutingKey == "" {
		return fmt.Errorf("pagerduty routing_key is required")
	}

	action := "resolve"
	severity := "info"
	if sc.NewState == "down" {
		action = "trigger"
		severity = "critical"
	} else if sc.NewState == "grace" {
		action = "trigger"
		severity = "warning"
	}

	payload := map[string]interface{}{
		"routing_key":  cfg.RoutingKey,
		"event_action": action,
		"dedup_key":    fmt.Sprintf("vigil-%s", sc.SwitchName),
		"payload": map[string]interface{}{
			"summary":   fmt.Sprintf("Vigil: %s is %s", sc.SwitchName, sc.NewState),
			"severity":  severity,
			"source":    "vigil",
			"component": sc.SwitchName,
			"custom_details": map[string]string{
				"old_state": sc.OldState,
				"new_state": sc.NewState,
				"details":   sc.Details,
			},
		},
	}
	return n.postJSON("https://events.pagerduty.com/v2/enqueue", payload)
}

// ── Telegram ──

type telegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

func (n *Notifier) sendTelegram(cfgJSON string, sc StateChange) error {
	var cfg telegramConfig
	if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
		return fmt.Errorf("invalid telegram config: %w", err)
	}
	if cfg.BotToken == "" || cfg.ChatID == "" {
		return fmt.Errorf("telegram bot_token and chat_id are required")
	}

	text := fmt.Sprintf("%s *%s*: `%s` → `%s`\n_%s_",
		stateEmoji(sc.NewState), sc.SwitchName, sc.OldState, sc.NewState, sc.Details)

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	payload := map[string]interface{}{
		"chat_id":    cfg.ChatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	return n.postJSON(url, payload)
}

// ── helpers ──

func (n *Notifier) postJSON(url string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := n.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
