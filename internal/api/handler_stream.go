package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/craftlabs/video-stream-capture-engine/internal/config"
)

type streamInfo struct {
	ID           string              `json:"id"`
	Group        string              `json:"group"`
	Status       string              `json:"status"`
	FPS          float64             `json:"fps"`
	Resolution   string              `json:"resolution"`
	FramesTotal  int64               `json:"frames_total"`
	LatencyMs    string              `json:"latency_ms"`
	Uptime       string              `json:"uptime"`
	RTSPURL      string              `json:"rtsp_url"`
	OutputTopic  string              `json:"output_topic"`
	CaptureFPS   int                 `json:"capture_fps"`
	DecodeScale  string              `json:"decode_scale"`
	Filters      []config.FilterSpec `json:"filters"`
	Restart      config.RestartPolicy `json:"restart"`
}

type streamListResponse struct {
	Total int          `json:"total"`
	Page  int          `json:"page"`
	Size  int          `json:"size"`
	Items []streamInfo `json:"items"`
}

func (s *Server) handleStreams(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listStreams(w, r)
	case http.MethodPost:
		s.createStream(w, r)
	default:
		writeJSON(w, 405, "method not allowed", nil)
	}
}

func (s *Server) handleStreamByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/streams/")
	id = strings.TrimSuffix(id, "/")
	id = strings.TrimPrefix(id, "/")

	// Check for action sub-paths
	if strings.HasSuffix(r.URL.Path, "/start") {
		id = strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/streams/"), "/start")
		s.handleStreamAction(w, r, id, "start")
		return
	}
	if strings.HasSuffix(r.URL.Path, "/stop") {
		id = strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/streams/"), "/stop")
		s.handleStreamAction(w, r, id, "stop")
		return
	}
	if strings.HasSuffix(r.URL.Path, "/restart") {
		id = strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/streams/"), "/restart")
		s.handleStreamAction(w, r, id, "restart")
		return
	}
	if strings.HasSuffix(r.URL.Path, "/frame") {
		id = strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/streams/"), "/frame")
		s.handleStreamFrame(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getStream(w, r, id)
	case http.MethodPut:
		s.updateStream(w, r, id)
	case http.MethodDelete:
		s.deleteStream(w, r, id)
	default:
		writeJSON(w, 405, "method not allowed", nil)
	}
}

func (s *Server) listStreams(w http.ResponseWriter, r *http.Request) {
	page := 1
	size := 20

	var mu sync.Mutex
	items := make([]streamInfo, 0)
	for _, sc := range s.cfg.Streams {
		info := streamInfo{
			ID:          sc.ID,
			Group:       sc.Group,
			Resolution:  sc.DecodeScale,
			RTSPURL:     sc.RTSPURL,
			OutputTopic: sc.OutputTopic,
			CaptureFPS:  sc.CaptureFPS,
			DecodeScale: sc.DecodeScale,
			Filters:     sc.Filters,
			Restart:     sc.Restart,
			Status:      "stopped",
		}
		if info.OutputTopic == "" {
			info.OutputTopic = sc.ID
		}
		mu.Lock()
		items = append(items, info)
		mu.Unlock()
	}
	_ = page
	_ = size

	writeJSON(w, 0, "ok", streamListResponse{
		Total: len(items),
		Page:  1,
		Size:  len(items),
		Items: items,
	})
}

func (s *Server) getStream(w http.ResponseWriter, r *http.Request, id string) {
	for _, sc := range s.cfg.Streams {
		if sc.ID == id {
			info := streamInfo{
				ID:          sc.ID,
				Group:       sc.Group,
				Resolution:  sc.DecodeScale,
				RTSPURL:     sc.RTSPURL,
				OutputTopic: sc.OutputTopic,
				CaptureFPS:  sc.CaptureFPS,
				DecodeScale: sc.DecodeScale,
				Filters:     sc.Filters,
				Restart:     sc.Restart,
				Status:      "running",
				FPS:         25.0,
				FramesTotal: 0,
				LatencyMs:   "42ms",
				Uptime:      "3d 14h",
			}
			if info.OutputTopic == "" {
				info.OutputTopic = sc.ID
			}
			writeJSON(w, 0, "ok", info)
			return
		}
	}
	writeJSON(w, 404, "stream not found", nil)
}

func (s *Server) createStream(w http.ResponseWriter, r *http.Request) {
	var sc config.StreamConfig
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		writeJSON(w, 400, "invalid body", nil)
		return
	}
	s.cfg.Streams = append(s.cfg.Streams, sc)
	slog.Info("stream created", "id", sc.ID)
	writeJSON(w, 0, "created", sc)
}

func (s *Server) updateStream(w http.ResponseWriter, r *http.Request, id string) {
	for i, sc := range s.cfg.Streams {
		if sc.ID == id {
			var updated config.StreamConfig
			if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
				writeJSON(w, 400, "invalid body", nil)
				return
			}
			updated.ID = id
			s.cfg.Streams[i] = updated
			writeJSON(w, 0, "updated", updated)
			return
		}
	}
	writeJSON(w, 404, "stream not found", nil)
}

func (s *Server) deleteStream(w http.ResponseWriter, r *http.Request, id string) {
	for i, sc := range s.cfg.Streams {
		if sc.ID == id {
			s.cfg.Streams = append(s.cfg.Streams[:i], s.cfg.Streams[i+1:]...)
			writeJSON(w, 0, "deleted", nil)
			return
		}
	}
	writeJSON(w, 404, "stream not found", nil)
}

func (s *Server) handleStreamAction(w http.ResponseWriter, r *http.Request, id, action string) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, "method not allowed", nil)
		return
	}
	slog.Info("stream action", "id", id, "action", action)
	writeJSON(w, 0, "ok", map[string]string{"action": action, "stream_id": id})
}

func (s *Server) handleStreamFrame(w http.ResponseWriter, r *http.Request, id string) {
	writeJSON(w, 0, "ok", map[string]string{"frame": "mock_base64_frame_data", "stream_id": id})
}

func (s *Server) handleBatchImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, 405, "method not allowed", nil)
		return
	}
	var streams []config.StreamConfig
	if err := json.NewDecoder(r.Body).Decode(&streams); err != nil {
		writeJSON(w, 400, "invalid body", nil)
		return
	}
	s.cfg.Streams = append(s.cfg.Streams, streams...)
	writeJSON(w, 0, "imported", map[string]int{"imported": len(streams)})
}
