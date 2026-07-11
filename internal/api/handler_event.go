package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/craftlabs/video-stream-capture-engine/internal/store"
)

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, 405, "method not allowed", nil)
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	size, _ := strconv.Atoi(q.Get("size"))

	events, total, err := s.db.ListEvents(r.Context(), store.EventFilter{
		Level:    q.Get("level"),
		StreamID: q.Get("stream_id"),
		Page:     page,
		Size:     size,
	})
	if err != nil {
		writeJSON(w, 500, "query failed", nil)
		return
	}
	if events == nil {
		events = []store.Event{}
	}

	writeJSON(w, 0, "ok", map[string]interface{}{
		"total": total,
		"page":  page,
		"size":  size,
		"items": events,
	})
}

func (s *Server) handleAckEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, "method not allowed", nil)
		return
	}

	var req struct {
		IDs []int `json:"ids"`
		All bool  `json:"all"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, "invalid body", nil)
		return
	}

	var err error
	if req.All {
		err = s.db.AckAllEvents(r.Context())
	} else {
		err = s.db.AckEvents(r.Context(), req.IDs)
	}
	if err != nil {
		writeJSON(w, 500, "ack failed", nil)
		return
	}
	writeJSON(w, 0, "acknowledged", nil)
}
