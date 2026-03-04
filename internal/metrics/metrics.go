package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SwitchStatus: 1 = healthy/up, 0 = violated/down
	SwitchStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dms_switch_status",
		Help: "Dead man switch status: 1=healthy, 0=violated",
	}, []string{"name", "mode", "signal"})

	// LastSignalTimestamp: unix timestamp of last observed signal
	LastSignalTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dms_last_signal_timestamp",
		Help: "Unix timestamp of last observed signal",
	}, []string{"name"})

	// ExpectedAtTimestamp: unix timestamp of when next signal is expected
	ExpectedAtTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dms_expected_at_timestamp",
		Help: "Unix timestamp of when next signal is expected",
	}, []string{"name"})

	// StateDurationSeconds: how long the switch has been in its current state
	StateDurationSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dms_state_duration_seconds",
		Help: "How long the switch has been in its current state",
	}, []string{"name", "state"})

	// EvalTotal: counter of evaluation results
	EvalTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "dms_eval_total",
		Help: "Total number of evaluations by result",
	}, []string{"name", "result"})
)
