package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, 0, "ok", map[string]interface{}{
			"engine":     s.cfg.Engine,
			"output":     s.cfg.Output,
			"database":   s.cfg.Database,
			"serializer": s.cfg.Output.Serializer,
		})
	case http.MethodPut:
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, 400, "invalid body", nil)
			return
		}
		writeJSON(w, 0, "updated", req)
	default:
		writeJSON(w, 405, "method not allowed", nil)
	}
}
