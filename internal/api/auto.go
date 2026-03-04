package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"vigil/internal/models"

	"github.com/go-chi/chi/v5"
)

func (s *Server) ListAutoRules(w http.ResponseWriter, r *http.Request) {
	var rules []models.AutoDiscoveryRule
	s.db.Order("created_at DESC").Find(&rules)

	// Also count auto-created switches
	var autoSwitchCount int64
	s.db.Model(&models.Switch{}).Where("auto_created = ?", true).Count(&autoSwitchCount)

	var activeLearning int64
	s.db.Model(&models.Switch{}).Where("auto_created = ? AND state = ?", true, models.StateLearning).Count(&activeLearning)

	jsonResp(w, http.StatusOK, map[string]interface{}{
		"rules":               rules,
		"auto_switch_count":   autoSwitchCount,
		"learning_count":      activeLearning,
	})
}

func (s *Server) CreateAutoRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LokiSelector        string  `json:"loki_selector"`
		Pattern             string  `json:"pattern"`
		MinSamples          int     `json:"min_samples"`
		ToleranceMultiplier float64 `json:"tolerance_multiplier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.LokiSelector == "" {
		jsonError(w, http.StatusBadRequest, "loki_selector is required")
		return
	}

	rule := models.AutoDiscoveryRule{
		LokiSelector:        req.LokiSelector,
		Pattern:             req.Pattern,
		Active:              true,
		MinSamples:          req.MinSamples,
		ToleranceMultiplier: req.ToleranceMultiplier,
	}
	if rule.MinSamples == 0 {
		rule.MinSamples = 4
	}
	if rule.ToleranceMultiplier == 0 {
		rule.ToleranceMultiplier = 2.0
	}

	if err := s.db.Create(&rule).Error; err != nil {
		jsonError(w, http.StatusInternalServerError, "failed to create rule: "+err.Error())
		return
	}

	jsonResp(w, http.StatusCreated, rule)
}

func (s *Server) UpdateAutoRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var rule models.AutoDiscoveryRule
	if err := s.db.First(&rule, id).Error; err != nil {
		jsonError(w, http.StatusNotFound, "rule not found")
		return
	}

	var req struct {
		LokiSelector        string  `json:"loki_selector"`
		Pattern             string  `json:"pattern"`
		Active              *bool   `json:"active"`
		MinSamples          int     `json:"min_samples"`
		ToleranceMultiplier float64 `json:"tolerance_multiplier"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if req.LokiSelector != "" {
		updates["loki_selector"] = req.LokiSelector
	}
	if req.Pattern != "" {
		updates["pattern"] = req.Pattern
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}
	if req.MinSamples > 0 {
		updates["min_samples"] = req.MinSamples
	}
	if req.ToleranceMultiplier > 0 {
		updates["tolerance_multiplier"] = req.ToleranceMultiplier
	}

	s.db.Model(&rule).Updates(updates)
	s.db.First(&rule, id)
	jsonResp(w, http.StatusOK, rule)
}

func (s *Server) DeleteAutoRule(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		jsonError(w, http.StatusBadRequest, "invalid id")
		return
	}

	s.db.Delete(&models.AutoDiscoveryRule{}, id)
	w.WriteHeader(http.StatusNoContent)
}
