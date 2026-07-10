package manager

import (
	"sync"
	"time"
)

type StreamStatus int

const (
	StatusNotFound StreamStatus = iota
	StatusHealthy
	StatusUnhealthy
)

func (s StreamStatus) String() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusUnhealthy:
		return "unhealthy"
	default:
		return "not_found"
	}
}

type streamHealth struct {
	lastHeartbeat time.Time
}

type HealthMonitor struct {
	mu      sync.RWMutex
	streams map[string]*streamHealth
	timeout time.Duration
}

func NewHealthMonitor(timeout time.Duration) *HealthMonitor {
	return &HealthMonitor{
		streams: make(map[string]*streamHealth),
		timeout: timeout,
	}
}

func (h *HealthMonitor) Register(streamID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.streams[streamID] = &streamHealth{lastHeartbeat: time.Now()}
}

func (h *HealthMonitor) Unregister(streamID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.streams, streamID)
}

func (h *HealthMonitor) Heartbeat(streamID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if s, ok := h.streams[streamID]; ok {
		s.lastHeartbeat = time.Now()
	}
}

func (h *HealthMonitor) Check(streamID string) StreamStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()

	s, ok := h.streams[streamID]
	if !ok {
		return StatusNotFound
	}

	if time.Since(s.lastHeartbeat) > h.timeout {
		return StatusUnhealthy
	}

	return StatusHealthy
}

func (h *HealthMonitor) UnhealthyStreams() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var result []string
	for id := range h.streams {
		if time.Since(h.streams[id].lastHeartbeat) > h.timeout {
			result = append(result, id)
		}
	}
	return result
}
