package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"vigil/internal/models"

	"github.com/go-chi/chi/v5"
)

func (s *Server) ListAlertChannels(w http.ResponseWriter, r *http.Request) {
	var channels []models.AlertChannel
	s.db.Order("created_at DESC").Find(&channels)
	jsonResp(w, http.StatusOK, channels)
}

func (s *Server) CreateAlertChannel(w http.ResponseWriter, r *http.Request) {
	var ch models.AlertChannel
	if err := json.NewDecoder(r.Body).Decode(&ch); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if ch.Name == "" || ch.Type == "" || ch.Config == "" {
		jsonError(w, http.StatusBadRequest, "name, type, and config are required")
		return
	}

	validTypes := map[string]bool{
		models.ChannelSlack:     true,
		models.ChannelWebhook:   true,
		models.ChannelDiscord:   true,
		models.ChannelPagerDuty: true,
		models.ChannelTelegram:  true,
	}
	if !validTypes[ch.Type] {
		jsonError(w, http.StatusBadRequest, "type must be one of: slack, webhook, discord, pagerduty, telegram")
		return
	}

	// Validate config is valid JSON
	var raw json.RawMessage
	if err := json.Unmarshal([]byte(ch.Config), &raw); err != nil {
		jsonError(w, http.StatusBadRequest, "config must be valid JSON")
		return
	}

	ch.Enabled = true
	if err := s.db.Create(&ch).Error; err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to create alert channel: "+err.Error())
		return
	}

	jsonResp(w, http.StatusCreated, ch)
}

func (s *Server) UpdateAlertChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var ch models.AlertChannel
	if err := s.db.First(&ch, id).Error; err != nil {
		jsonError(w, http.StatusNotFound, "alert channel not found")
		return
	}

	var req models.AlertChannel
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	if req.Config != "" {
		var raw json.RawMessage
		if err := json.Unmarshal([]byte(req.Config), &raw); err != nil {
			jsonError(w, http.StatusBadRequest, "config must be valid JSON")
			return
		}
		updates["config"] = req.Config
	}
	updates["enabled"] = req.Enabled

	s.db.Model(&ch).Updates(updates)
	s.db.First(&ch, id)
	jsonResp(w, http.StatusOK, ch)
}

func (s *Server) DeleteAlertChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	s.db.Where("alert_channel_id = ?", id).Delete(&models.AlertLog{})
	s.db.Delete(&models.AlertChannel{}, id)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) TestAlertChannel(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var ch models.AlertChannel
	if err := s.db.First(&ch, id).Error; err != nil {
		jsonError(w, http.StatusNotFound, "alert channel not found")
		return
	}

	if err := s.notifier.SendTest(ch); err != nil {
		jsonResp(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	jsonResp(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Test alert sent successfully",
	})
}

func (s *Server) ListAlertLogs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	var logs []models.AlertLog
	s.db.Order("sent_at DESC").Limit(limit).Find(&logs)
	jsonResp(w, http.StatusOK, logs)
}
