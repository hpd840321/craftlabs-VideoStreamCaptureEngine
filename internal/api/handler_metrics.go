package api

import "net/http"

func (s *Server) handleMetricsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, 405, "method not allowed", nil)
		return
	}

	writeJSON(w, 0, "ok", map[string]interface{}{
		"online_streams":  42,
		"total_streams":   50,
		"frames_today":    847000,
		"avg_fps":         23.5,
		"active_alerts":   3,
		"unacknowledged":  2,
		"fps_trend": []float64{24.5, 25.0, 24.8, 25.0, 23.2, 25.0, 0, 24.5},
	})
}
