package evaluator

import (
	"fmt"
	"time"

	"vigil/internal/models"
)

// EvalResult represents the outcome of evaluating a single switch
type EvalResult struct {
	Pass         bool
	State        string // new state to transition to
	LastSignalAt *time.Time
	NextExpected *time.Time
	Details      string
}

// EvalFrequency evaluates a switch in frequency mode.
// It checks whether a signal was received within the expected interval + grace period.
func EvalFrequency(sw *models.Switch, lastSignal *time.Time, now time.Time) EvalResult {
	interval := time.Duration(sw.IntervalSeconds) * time.Second
	grace := time.Duration(sw.GraceSeconds) * time.Second

	// Check time window
	wc, err := CheckWindow(sw.WindowStart, sw.WindowEnd, sw.WindowTZ, now)
	if err != nil {
		return EvalResult{
			Pass:    true, // don't trigger on config error
			State:   sw.State,
			Details: fmt.Sprintf("window check error: %v", err),
		}
	}

	// No signal ever seen
	if lastSignal == nil {
		if sw.State == models.StateNew {
			return EvalResult{
				Pass:    true,
				State:   models.StateNew,
				Details: "no signal received yet",
			}
		}
		// Was previously up but now no signal — evaluate as missing
		return evalMissing(sw, nil, now, interval, grace, wc)
	}

	// Calculate next expected time
	nextExpected := lastSignal.Add(interval)

	// If we have a window, the next expected is at window_end + grace
	if wc.InWindow {
		deadline := nextExpected.Add(grace)
		if now.After(deadline) {
			return EvalResult{
				Pass:         false,
				State:        models.StateDown,
				LastSignalAt: lastSignal,
				NextExpected: &nextExpected,
				Details:      fmt.Sprintf("signal overdue: last=%s, expected every %s, grace=%s", lastSignal.Format(time.RFC3339), interval, grace),
			}
		}

		// Within grace period
		if now.After(nextExpected) {
			return EvalResult{
				Pass:         true,
				State:        models.StateGrace,
				LastSignalAt: lastSignal,
				NextExpected: &nextExpected,
				Details:      fmt.Sprintf("in grace period: last=%s, deadline=%s", lastSignal.Format(time.RFC3339), deadline.Format(time.RFC3339)),
			}
		}

		// All good
		return EvalResult{
			Pass:         true,
			State:        models.StateUp,
			LastSignalAt: lastSignal,
			NextExpected: &nextExpected,
			Details:      "signal within expected interval",
		}
	}

	// Outside window — if windowed, check against window boundaries
	if sw.WindowStart != "" && sw.WindowEnd != "" {
		// Outside the window, check if signal was seen during today's window
		if lastSignal.Before(wc.WindowStart) {
			// No signal during today's window — check if window + grace has passed
			deadline := wc.WindowEnd.Add(grace)
			if now.After(deadline) {
				return EvalResult{
					Pass:         false,
					State:        models.StateDown,
					LastSignalAt: lastSignal,
					NextExpected: &wc.WindowEnd,
					Details:      fmt.Sprintf("no signal during window %s-%s, grace expired", sw.WindowStart, sw.WindowEnd),
				}
			}
		}
		// Either signal was during window, or we haven't passed grace yet
		return EvalResult{
			Pass:         true,
			State:        sw.State, // maintain current state outside window
			LastSignalAt: lastSignal,
			Details:      "outside monitoring window",
		}
	}

	return EvalResult{
		Pass:         true,
		State:        models.StateUp,
		LastSignalAt: lastSignal,
		NextExpected: &nextExpected,
		Details:      "signal ok",
	}
}

func evalMissing(sw *models.Switch, lastSignal *time.Time, now time.Time, interval, grace time.Duration, wc WindowCheck) EvalResult {
	if !wc.InWindow && sw.WindowStart != "" {
		return EvalResult{
			Pass:         true,
			State:        sw.State,
			LastSignalAt: lastSignal,
			Details:      "outside monitoring window, no signal yet",
		}
	}

	return EvalResult{
		Pass:         false,
		State:        models.StateDown,
		LastSignalAt: lastSignal,
		Details:      "no signal received, expected within interval",
	}
}
