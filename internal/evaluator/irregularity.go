package evaluator

import (
	"fmt"
	"math"
	"sort"
	"time"

	"vigil/internal/models"
)

// EvalIrregularity evaluates a switch in irregularity mode.
// Uses historical signal occurrences to predict the next expected signal.
func EvalIrregularity(sw *models.Switch, occurrences []time.Time, now time.Time) EvalResult {
	minSamples := sw.MinSamples
	if minSamples < 3 {
		minSamples = 3
	}

	tolerance := sw.ToleranceMultiplier
	if tolerance <= 0 {
		tolerance = 2.0
	}

	if len(occurrences) < minSamples {
		return EvalResult{
			Pass:    true,
			State:   models.StateLearning,
			Details: fmt.Sprintf("learning: have %d/%d data points", len(occurrences), minSamples),
		}
	}

	// Sort occurrences chronologically
	sort.Slice(occurrences, func(i, j int) bool {
		return occurrences[i].Before(occurrences[j])
	})

	// Compute intervals between consecutive occurrences
	intervals := make([]time.Duration, 0, len(occurrences)-1)
	for i := 1; i < len(occurrences); i++ {
		intervals = append(intervals, occurrences[i].Sub(occurrences[i-1]))
	}

	// Compute median interval
	median := medianDuration(intervals)

	// Time since last occurrence
	lastSignal := occurrences[len(occurrences)-1]
	elapsed := now.Sub(lastSignal)

	// Predicted next = last + median
	predictedNext := lastSignal.Add(median)

	// Deadline = predicted + (tolerance * median - median) = last + tolerance * median
	deadline := lastSignal.Add(time.Duration(float64(median) * tolerance))

	if now.After(deadline) {
		return EvalResult{
			Pass:         false,
			State:        models.StateDown,
			LastSignalAt: &lastSignal,
			NextExpected: &predictedNext,
			Details:      fmt.Sprintf("overdue: elapsed=%s, median_interval=%s, tolerance=%.1fx, deadline=%s", elapsed, median, tolerance, deadline.Format(time.RFC3339)),
		}
	}

	if now.After(predictedNext) {
		return EvalResult{
			Pass:         true,
			State:        models.StateGrace,
			LastSignalAt: &lastSignal,
			NextExpected: &predictedNext,
			Details:      fmt.Sprintf("past predicted but within tolerance: elapsed=%s, median=%s", elapsed, median),
		}
	}

	return EvalResult{
		Pass:         true,
		State:        models.StateUp,
		LastSignalAt: &lastSignal,
		NextExpected: &predictedNext,
		Details:      fmt.Sprintf("on schedule: next predicted at %s (median=%s)", predictedNext.Format(time.RFC3339), median),
	}
}

func medianDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	n := len(sorted)
	if n%2 == 0 {
		return time.Duration(math.Round(float64(sorted[n/2-1]+sorted[n/2]) / 2))
	}
	return sorted[n/2]
}
