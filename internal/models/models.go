package models

import (
	"time"
)

// Switch states
const (
	StateNew      = "new"
	StateUp       = "up"
	StateDown     = "down"
	StateGrace    = "grace"
	StateLearning = "learning"
	StatePaused   = "paused"
)

// Signal types
const (
	SignalPrometheus = "prometheus"
	SignalLoki       = "loki"
)

// Detection modes
const (
	ModeFrequency    = "frequency"
	ModeIrregularity = "irregularity"
)

type Switch struct {
	ID        uint   `gorm:"primarykey" json:"id"`
	Name      string `gorm:"uniqueIndex;not null" json:"name"`
	Signal    string `gorm:"not null" json:"signal"`   // "prometheus" or "loki"
	Query     string `gorm:"not null" json:"query"`     // PromQL or LogQL
	Mode      string `gorm:"not null" json:"mode"`      // "frequency" or "irregularity"
	State     string `gorm:"not null;default:new" json:"state"`
	AutoCreated bool `gorm:"default:false" json:"auto_created"`

	// Frequency mode fields
	IntervalSeconds int `json:"interval_seconds"` // expected every N seconds
	GraceSeconds    int `json:"grace_seconds"`    // grace period after expected time

	// Time window (optional) — empty means all day
	WindowStart string `json:"window_start"` // "09:00"
	WindowEnd   string `json:"window_end"`   // "11:00"
	WindowTZ    string `json:"window_tz"`    // "Asia/Kolkata", defaults to UTC

	// Irregularity mode fields
	MinSamples          int     `json:"min_samples"`          // minimum data points before activating
	ToleranceMultiplier float64 `json:"tolerance_multiplier"` // e.g. 2.0 = 2x median

	// Runtime state
	LastSignalAt    *time.Time `json:"last_signal_at"`
	NextExpectedAt  *time.Time `json:"next_expected_at"`
	StateChangedAt  time.Time  `json:"state_changed_at"`
	EvalPassCount   int64      `json:"eval_pass_count"`
	EvalFailCount   int64      `json:"eval_fail_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type EvalHistory struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	SwitchID  uint      `gorm:"index;not null" json:"switch_id"`
	EvalAt    time.Time `gorm:"not null" json:"eval_at"`
	Result    string    `gorm:"not null" json:"result"` // "pass" or "fail"
	State     string    `gorm:"not null" json:"state"`
	SignalAt  *time.Time `json:"signal_at"`
	Details   string    `json:"details"` // human-readable reason
}

type AutoDiscoveryRule struct {
	ID            uint   `gorm:"primarykey" json:"id"`
	LokiSelector  string `gorm:"not null" json:"loki_selector"`  // e.g. {job="diagon-alley"}
	Pattern       string `json:"pattern"`                         // e.g. "[CRON]*"
	Active        bool   `gorm:"default:true" json:"active"`
	MinSamples    int    `gorm:"default:4" json:"min_samples"`
	ToleranceMultiplier float64 `gorm:"default:2.0" json:"tolerance_multiplier"`
	LastScanAt    *time.Time `json:"last_scan_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// SignalOccurrence stores timestamps for irregularity detection
type SignalOccurrence struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	SwitchID  uint      `gorm:"index;not null" json:"switch_id"`
	OccurredAt time.Time `gorm:"not null" json:"occurred_at"`
}

// Alert channel types
const (
	ChannelSlack     = "slack"
	ChannelWebhook   = "webhook"
	ChannelDiscord   = "discord"
	ChannelPagerDuty = "pagerduty"
	ChannelTelegram  = "telegram"
)

// AlertChannel configures a notification destination
type AlertChannel struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Name      string    `gorm:"uniqueIndex;not null" json:"name"`
	Type      string    `gorm:"not null" json:"type"`    // slack, webhook, discord, pagerduty, telegram
	Config    string    `gorm:"not null" json:"config"`  // JSON config blob
	Enabled   bool      `gorm:"default:true" json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AlertLog records each notification sent
type AlertLog struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	AlertChannelID uint      `gorm:"index;not null" json:"alert_channel_id"`
	SwitchID       uint      `gorm:"index;not null" json:"switch_id"`
	SwitchName     string    `gorm:"not null" json:"switch_name"`
	OldState       string    `json:"old_state"`
	NewState       string    `json:"new_state"`
	Success        bool      `json:"success"`
	Error          string    `json:"error,omitempty"`
	SentAt         time.Time `gorm:"not null" json:"sent_at"`
}
