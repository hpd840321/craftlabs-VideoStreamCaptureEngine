package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/craftlabs/video-stream-capture-engine/internal/auth"
	"github.com/craftlabs/video-stream-capture-engine/internal/config"
	"github.com/craftlabs/video-stream-capture-engine/internal/manager"
	"github.com/craftlabs/video-stream-capture-engine/internal/store"
)

type Server struct {
	cfg     *config.Config
	mgr     *manager.StreamManager
	db      *store.DB
	jwtKey  []byte
}

func NewServer(cfg *config.Config, mgr *manager.StreamManager, db *store.DB, jwtSecret string) *Server {
	return &Server{
		cfg:    cfg,
		mgr:    mgr,
		db:     db,
		jwtKey: []byte(jwtSecret),
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// Public
	mux.HandleFunc("/api/login", s.handleLogin)
	mux.HandleFunc("/api/refresh", s.handleRefresh)

	// Protected
	protected := s.authMiddleware(http.DefaultServeMux)
	_ = protected
	mux.HandleFunc("/api/streams", s.authWrap(s.handleStreams))
	mux.HandleFunc("/api/streams/", s.authWrap(s.handleStreamByID))
	mux.HandleFunc("/api/streams/batch", s.authWrap(s.handleBatchImport))
	mux.HandleFunc("/api/events", s.authWrap(s.handleEvents))
	mux.HandleFunc("/api/events/ack", s.authWrap(s.handleAckEvents))
	mux.HandleFunc("/api/config", s.authWrap(s.handleConfig))
	mux.HandleFunc("/api/metrics/summary", s.authWrap(s.handleMetricsSummary))
}

func (s *Server) authWrap(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			writeJSON(w, 401, "unauthorized", nil)
			return
		}
		claims, err := auth.ValidateToken(token, s.jwtKey)
		if err != nil {
			writeJSON(w, 401, "invalid token", nil)
			return
		}
		ctx := context.WithValue(r.Context(), ctxKeyUsername, claims.Username)
		handler(w, r.WithContext(ctx))
	}
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" || strings.Contains(r.URL.Path, "/api/login") || strings.Contains(r.URL.Path, "/health") || strings.Contains(r.URL.Path, "/metrics") {
			next.ServeHTTP(w, r)
			return
		}
		_, err := auth.ValidateToken(token, s.jwtKey)
		if err != nil {
			writeJSON(w, 401, "invalid token", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if code >= 400 {
		w.WriteHeader(code)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    code,
		"message": message,
		"data":    data,
	})
}

func getUsername(r *http.Request) string {
	if v := r.Context().Value(ctxKeyUsername); v != nil {
		return v.(string)
	}
	return ""
}

type ctxKey string

const ctxKeyUsername ctxKey = "username"

func init() {
	_ = slog.Default()
}
