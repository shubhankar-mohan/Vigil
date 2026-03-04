package api

import (
	"net/http"

	"vigil/internal/models"
)

type DashboardResponse struct {
	Total    int64            `json:"total"`
	Up       int64            `json:"up"`
	Down     int64            `json:"down"`
	Grace    int64            `json:"grace"`
	Learning int64            `json:"learning"`
	Paused   int64            `json:"paused"`
	New      int64            `json:"new"`
	Switches []models.Switch  `json:"switches"`
}

func (s *Server) Dashboard(w http.ResponseWriter, r *http.Request) {
	var switches []models.Switch
	s.db.Order("state ASC, name ASC").Find(&switches)

	resp := DashboardResponse{
		Total:    int64(len(switches)),
		Switches: switches,
	}

	for _, sw := range switches {
		switch sw.State {
		case models.StateUp:
			resp.Up++
		case models.StateDown:
			resp.Down++
		case models.StateGrace:
			resp.Grace++
		case models.StateLearning:
			resp.Learning++
		case models.StatePaused:
			resp.Paused++
		case models.StateNew:
			resp.New++
		}
	}

	jsonResp(w, http.StatusOK, resp)
}
