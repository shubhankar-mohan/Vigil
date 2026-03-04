package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"vigil/internal/models"

	"github.com/go-chi/chi/v5"
)

func (s *Server) ListSwitches(w http.ResponseWriter, r *http.Request) {
	var switches []models.Switch
	if err := s.db.Order("created_at DESC").Find(&switches).Error; err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to list switches")
		return
	}
	jsonResp(w, http.StatusOK, switches)
}

func (s *Server) GetSwitch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var sw models.Switch
	if err := s.db.First(&sw, id).Error; err != nil {
		jsonError(w, http.StatusNotFound, "switch not found")
		return
	}
	jsonResp(w, http.StatusOK, sw)
}

type createSwitchRequest struct {
	Name                string  `json:"name"`
	Signal              string  `json:"signal"`
	Query               string  `json:"query"`
	Mode                string  `json:"mode"`
	IntervalSeconds     int     `json:"interval_seconds"`
	GraceSeconds        int     `json:"grace_seconds"`
	WindowStart         string  `json:"window_start"`
	WindowEnd           string  `json:"window_end"`
	WindowTZ            string  `json:"window_tz"`
	MinSamples          int     `json:"min_samples"`
	ToleranceMultiplier float64 `json:"tolerance_multiplier"`
}

func (s *Server) CreateSwitch(w http.ResponseWriter, r *http.Request) {
	var req createSwitchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Signal == "" || req.Query == "" || req.Mode == "" {
		jsonError(w, http.StatusBadRequest, "name, signal, query, and mode are required")
		return
	}

	if req.Signal != models.SignalPrometheus && req.Signal != models.SignalLoki {
		jsonError(w, http.StatusBadRequest, "signal must be 'prometheus' or 'loki'")
		return
	}

	if req.Mode != models.ModeFrequency && req.Mode != models.ModeIrregularity {
		jsonError(w, http.StatusBadRequest, "mode must be 'frequency' or 'irregularity'")
		return
	}

	sw := models.Switch{
		Name:                req.Name,
		Signal:              req.Signal,
		Query:               req.Query,
		Mode:                req.Mode,
		State:               models.StateNew,
		IntervalSeconds:     req.IntervalSeconds,
		GraceSeconds:        req.GraceSeconds,
		WindowStart:         req.WindowStart,
		WindowEnd:           req.WindowEnd,
		WindowTZ:            req.WindowTZ,
		MinSamples:          req.MinSamples,
		ToleranceMultiplier: req.ToleranceMultiplier,
		StateChangedAt:      time.Now(),
	}

	if err := s.db.Create(&sw).Error; err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to create switch: "+err.Error())
		return
	}

	jsonResp(w, http.StatusCreated, sw)
}

func (s *Server) UpdateSwitch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var sw models.Switch
	if err := s.db.First(&sw, id).Error; err != nil {
		jsonError(w, http.StatusNotFound, "switch not found")
		return
	}

	var req createSwitchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Signal != "" {
		updates["signal"] = req.Signal
	}
	if req.Query != "" {
		updates["query"] = req.Query
	}
	if req.Mode != "" {
		updates["mode"] = req.Mode
	}
	if req.IntervalSeconds > 0 {
		updates["interval_seconds"] = req.IntervalSeconds
	}
	if req.GraceSeconds > 0 {
		updates["grace_seconds"] = req.GraceSeconds
	}
	updates["window_start"] = req.WindowStart
	updates["window_end"] = req.WindowEnd
	updates["window_tz"] = req.WindowTZ
	if req.MinSamples > 0 {
		updates["min_samples"] = req.MinSamples
	}
	if req.ToleranceMultiplier > 0 {
		updates["tolerance_multiplier"] = req.ToleranceMultiplier
	}

	if err := s.db.Model(&sw).Updates(updates).Error; err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to update switch")
		return
	}

	s.db.First(&sw, id)
	jsonResp(w, http.StatusOK, sw)
}

func (s *Server) DeleteSwitch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Delete related records first
	s.db.Where("switch_id = ?", id).Delete(&models.EvalHistory{})
	s.db.Where("switch_id = ?", id).Delete(&models.SignalOccurrence{})
	s.db.Delete(&models.Switch{}, id)

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) PauseSwitch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	s.db.Model(&models.Switch{}).Where("id = ?", id).Updates(map[string]interface{}{
		"state":            models.StatePaused,
		"state_changed_at": time.Now(),
	})

	jsonResp(w, http.StatusOK, map[string]string{"status": "paused"})
}

func (s *Server) ResumeSwitch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	s.db.Model(&models.Switch{}).Where("id = ?", id).Updates(map[string]interface{}{
		"state":            models.StateNew,
		"state_changed_at": time.Now(),
	})

	jsonResp(w, http.StatusOK, map[string]string{"status": "resumed"})
}

func (s *Server) TestQuery(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Signal string `json:"signal"`
		Query  string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Signal == "" || req.Query == "" {
		jsonError(w, http.StatusBadRequest, "signal and query are required")
		return
	}

	switch req.Signal {
	case "prometheus":
		val, ts, err := s.promClient.QueryInstant(req.Query)
		if err != nil {
			jsonResp(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		signalTime := time.Unix(int64(val), 0)
		jsonResp(w, http.StatusOK, map[string]interface{}{
			"success":        true,
			"raw_value":      val,
			"query_time":     ts.Format(time.RFC3339),
			"signal_time":    signalTime.Format(time.RFC3339),
			"signal_age":     time.Since(signalTime).Round(time.Second).String(),
			"message":        "Query returned a value. The raw_value is interpreted as a Unix timestamp for frequency mode.",
		})

	case "loki":
		lookback := 24 * time.Hour
		lastOccurrence, err := s.lokiClient.QueryLastOccurrence(req.Query, lookback)
		if err != nil {
			jsonResp(w, http.StatusOK, map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		if lastOccurrence == nil {
			jsonResp(w, http.StatusOK, map[string]interface{}{
				"success": true,
				"message": "Query executed successfully but no matching logs found in the last 24h.",
			})
			return
		}
		jsonResp(w, http.StatusOK, map[string]interface{}{
			"success":         true,
			"last_occurrence": lastOccurrence.Format(time.RFC3339),
			"signal_age":      time.Since(*lastOccurrence).Round(time.Second).String(),
			"message":         "Found matching log entry.",
		})

	default:
		jsonError(w, http.StatusBadRequest, "signal must be 'prometheus' or 'loki'")
	}
}

func (s *Server) GetSwitchHistory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	var history []models.EvalHistory
	s.db.Where("switch_id = ?", id).Order("eval_at DESC").Limit(limit).Find(&history)
	jsonResp(w, http.StatusOK, history)
}

// JSON helpers

func jsonResp(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, message string) {
	jsonResp(w, status, map[string]string{"error": message})
}
