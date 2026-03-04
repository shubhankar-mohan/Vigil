package evaluator

import (
	"context"
	"log"
	"time"

	"vigil/internal/config"
	"vigil/internal/metrics"
	"vigil/internal/models"

	"gorm.io/gorm"
)

type Evaluator struct {
	db         *gorm.DB
	promClient *PromClient
	lokiClient *LokiClient
	cfg        *config.Config
}

func New(db *gorm.DB, cfg *config.Config, promClient *PromClient, lokiClient *LokiClient) *Evaluator {
	return &Evaluator{
		db:         db,
		promClient: promClient,
		lokiClient: lokiClient,
		cfg:        cfg,
	}
}

// Run starts the evaluation loop. Blocks until ctx is cancelled.
func (e *Evaluator) Run(ctx context.Context) {
	ticker := time.NewTicker(e.cfg.EvalInterval)
	defer ticker.Stop()

	// Run immediately on start
	e.evalAll()

	for {
		select {
		case <-ctx.Done():
			log.Println("evaluator stopping")
			return
		case <-ticker.C:
			e.evalAll()
		}
	}
}

func (e *Evaluator) evalAll() {
	var switches []models.Switch
	if err := e.db.Where("state != ?", models.StatePaused).Find(&switches).Error; err != nil {
		log.Printf("evaluator: failed to load switches: %v", err)
		return
	}

	now := time.Now()
	for i := range switches {
		e.evalSwitch(&switches[i], now)
	}
}

func (e *Evaluator) evalSwitch(sw *models.Switch, now time.Time) {
	var result EvalResult

	switch sw.Mode {
	case models.ModeFrequency:
		result = e.evalFrequencySwitch(sw, now)
	case models.ModeIrregularity:
		result = e.evalIrregularitySwitch(sw, now)
	default:
		log.Printf("evaluator: unknown mode %q for switch %q", sw.Mode, sw.Name)
		return
	}

	// Update switch state in DB
	e.applyResult(sw, &result, now)

	// Update Prometheus metrics
	e.updateMetrics(sw, &result, now)
}

func (e *Evaluator) evalFrequencySwitch(sw *models.Switch, now time.Time) EvalResult {
	var lastSignal *time.Time

	switch sw.Signal {
	case models.SignalPrometheus:
		val, _, err := e.promClient.QueryInstant(sw.Query)
		if err != nil {
			log.Printf("evaluator: prom query error for %q: %v", sw.Name, err)
			// Don't change state on query error — retain current state
			return EvalResult{Pass: true, State: sw.State, Details: "query error: " + err.Error()}
		}
		// Treat the value as a unix timestamp
		t := time.Unix(int64(val), 0)
		lastSignal = &t

	case models.SignalLoki:
		// Look back for interval + grace + buffer
		lookback := time.Duration(sw.IntervalSeconds+sw.GraceSeconds)*time.Second + 10*time.Minute
		t, err := e.lokiClient.QueryLastOccurrence(sw.Query, lookback)
		if err != nil {
			log.Printf("evaluator: loki query error for %q: %v", sw.Name, err)
			return EvalResult{Pass: true, State: sw.State, Details: "query error: " + err.Error()}
		}
		lastSignal = t
	}

	return EvalFrequency(sw, lastSignal, now)
}

func (e *Evaluator) evalIrregularitySwitch(sw *models.Switch, now time.Time) EvalResult {
	// First, try to get latest signal and record it
	var latestSignal *time.Time

	switch sw.Signal {
	case models.SignalPrometheus:
		val, _, err := e.promClient.QueryInstant(sw.Query)
		if err == nil {
			t := time.Unix(int64(val), 0)
			latestSignal = &t
		}
	case models.SignalLoki:
		t, err := e.lokiClient.QueryLastOccurrence(sw.Query, 24*time.Hour)
		if err == nil {
			latestSignal = t
		}
	}

	// Record new occurrence if it's different from the last recorded one
	if latestSignal != nil {
		e.recordOccurrence(sw.ID, *latestSignal)
	}

	// Load all recorded occurrences
	var occRecords []models.SignalOccurrence
	e.db.Where("switch_id = ?", sw.ID).Order("occurred_at ASC").Find(&occRecords)

	occurrences := make([]time.Time, len(occRecords))
	for i, r := range occRecords {
		occurrences[i] = r.OccurredAt
	}

	return EvalIrregularity(sw, occurrences, now)
}

// recordOccurrence adds a signal occurrence if it's new (not a duplicate).
func (e *Evaluator) recordOccurrence(switchID uint, at time.Time) {
	// Check if we already have this timestamp (within 1 second tolerance)
	var count int64
	e.db.Model(&models.SignalOccurrence{}).
		Where("switch_id = ? AND occurred_at BETWEEN ? AND ?", switchID, at.Add(-time.Second), at.Add(time.Second)).
		Count(&count)

	if count == 0 {
		e.db.Create(&models.SignalOccurrence{
			SwitchID:   switchID,
			OccurredAt: at,
		})
	}
}

func (e *Evaluator) applyResult(sw *models.Switch, result *EvalResult, now time.Time) {
	oldState := sw.State
	updates := map[string]interface{}{
		"last_signal_at":   result.LastSignalAt,
		"next_expected_at": result.NextExpected,
	}

	if result.State != "" && result.State != oldState {
		updates["state"] = result.State
		updates["state_changed_at"] = now
	}

	if result.Pass {
		updates["eval_pass_count"] = gorm.Expr("eval_pass_count + 1")
	} else {
		updates["eval_fail_count"] = gorm.Expr("eval_fail_count + 1")
	}

	e.db.Model(sw).Updates(updates)

	// Record eval history
	resultStr := "pass"
	if !result.Pass {
		resultStr = "fail"
	}
	e.db.Create(&models.EvalHistory{
		SwitchID: sw.ID,
		EvalAt:   now,
		Result:   resultStr,
		State:    result.State,
		SignalAt: result.LastSignalAt,
		Details:  result.Details,
	})

	// Log state transitions
	if result.State != "" && result.State != oldState {
		log.Printf("switch %q: %s -> %s (%s)", sw.Name, oldState, result.State, result.Details)
	}
}

func (e *Evaluator) updateMetrics(sw *models.Switch, result *EvalResult, now time.Time) {
	// Switch status
	status := float64(1)
	if result.State == models.StateDown {
		status = 0
	}
	metrics.SwitchStatus.WithLabelValues(sw.Name, sw.Mode, sw.Signal).Set(status)

	// Last signal timestamp
	if result.LastSignalAt != nil {
		metrics.LastSignalTimestamp.WithLabelValues(sw.Name).Set(float64(result.LastSignalAt.Unix()))
	}

	// Expected at timestamp
	if result.NextExpected != nil {
		metrics.ExpectedAtTimestamp.WithLabelValues(sw.Name).Set(float64(result.NextExpected.Unix()))
	}

	// State duration
	var stateChangedAt time.Time
	e.db.Model(sw).Select("state_changed_at").First(&stateChangedAt)
	if !sw.StateChangedAt.IsZero() {
		metrics.StateDurationSeconds.WithLabelValues(sw.Name, sw.State).Set(now.Sub(sw.StateChangedAt).Seconds())
	}

	// Eval counter
	resultStr := "pass"
	if !result.Pass {
		resultStr = "fail"
	}
	metrics.EvalTotal.WithLabelValues(sw.Name, resultStr).Inc()
}
