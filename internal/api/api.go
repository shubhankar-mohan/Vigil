package api

import (
	"net/http"

	"vigil/internal/evaluator"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

type Server struct {
	db         *gorm.DB
	promClient *evaluator.PromClient
	lokiClient *evaluator.LokiClient
	router     chi.Router
}

func NewServer(db *gorm.DB, promClient *evaluator.PromClient, lokiClient *evaluator.LokiClient) *Server {
	s := &Server{db: db, promClient: promClient, lokiClient: lokiClient}
	s.setupRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) setupRoutes() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware)

	// Prometheus metrics
	r.Handle("/metrics", promhttp.Handler())

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Route("/switches", func(r chi.Router) {
			r.Get("/", s.ListSwitches)
			r.Post("/", s.CreateSwitch)
			r.Get("/{id}", s.GetSwitch)
			r.Put("/{id}", s.UpdateSwitch)
			r.Delete("/{id}", s.DeleteSwitch)
			r.Post("/{id}/pause", s.PauseSwitch)
			r.Post("/{id}/resume", s.ResumeSwitch)
			r.Post("/test-query", s.TestQuery)
			r.Get("/{id}/history", s.GetSwitchHistory)
		})

		r.Get("/dashboard", s.Dashboard)

		r.Route("/auto-rules", func(r chi.Router) {
			r.Get("/", s.ListAutoRules)
			r.Post("/", s.CreateAutoRule)
			r.Put("/{id}", s.UpdateAutoRule)
			r.Delete("/{id}", s.DeleteAutoRule)
		})
	})

	// Serve React static files — fallback to index.html for SPA routing
	fileServer := http.FileServer(http.Dir("web/dist"))
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		// Try serving the static file first
		path := "web/dist" + r.URL.Path
		if _, err := http.Dir(".").Open(path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}
		// Fallback to index.html for SPA routes
		http.ServeFile(w, r, "web/dist/index.html")
	})

	s.router = r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
