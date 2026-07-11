package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/craftlabs/video-stream-capture-engine/internal/auth"
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, "method not allowed", nil)
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, "invalid request", nil)
		return
	}

	user, err := s.db.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		writeJSON(w, 401, "invalid credentials", nil)
		return
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		writeJSON(w, 401, "invalid credentials", nil)
		return
	}

	token, err := auth.GenerateToken(req.Username, s.jwtKey, 24*time.Hour)
	if err != nil {
		writeJSON(w, 500, "token generation failed", nil)
		return
	}

	writeJSON(w, 0, "ok", map[string]interface{}{
		"token":      token,
		"expires_at": time.Now().Add(24 * time.Hour),
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, "method not allowed", nil)
		return
	}
	token, err := auth.GenerateToken(getUsername(r), s.jwtKey, 24*time.Hour)
	if err != nil {
		writeJSON(w, 500, "token refresh failed", nil)
		return
	}
	writeJSON(w, 0, "ok", map[string]interface{}{
		"token":      token,
		"expires_at": time.Now().Add(24 * time.Hour),
	})
}

func (s *Server) InitAdminUser() error {
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		password = "admin"
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	return s.db.CreateUser(context.Background(), "admin", hash)
}
