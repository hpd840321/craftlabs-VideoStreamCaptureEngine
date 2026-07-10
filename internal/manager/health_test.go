package manager

import (
	"testing"
	"time"
)

func TestHealthMonitor_RegisterAndCheck(t *testing.T) {
	hm := NewHealthMonitor(1 * time.Second)
	hm.Register("stream-1")

	if hm.Check("stream-1") != StatusHealthy {
		t.Errorf("fresh stream should be healthy")
	}
	hm.Heartbeat("stream-1")
	if hm.Check("stream-1") != StatusHealthy {
		t.Error("should be healthy after heartbeat")
	}
}

func TestHealthMonitor_Timeout(t *testing.T) {
	hm := NewHealthMonitor(10 * time.Millisecond)
	hm.Register("stream-1")
	hm.Heartbeat("stream-1")
	time.Sleep(20 * time.Millisecond)

	if hm.Check("stream-1") != StatusUnhealthy {
		t.Errorf("timed-out stream should be unhealthy")
	}
}

func TestHealthMonitor_Unregister(t *testing.T) {
	hm := NewHealthMonitor(1 * time.Second)
	hm.Register("stream-1")
	hm.Unregister("stream-1")

	if hm.Check("stream-1") != StatusNotFound {
		t.Errorf("unregistered stream should not be found")
	}
}

func TestHealthMonitor_ActiveHeartbeat(t *testing.T) {
	hm := NewHealthMonitor(30 * time.Millisecond)
	hm.Register("stream-1")
	hm.Heartbeat("stream-1")
	time.Sleep(10 * time.Millisecond)
	hm.Heartbeat("stream-1")
	time.Sleep(10 * time.Millisecond)
	hm.Heartbeat("stream-1")

	if hm.Check("stream-1") != StatusHealthy {
		t.Error("should stay healthy with active heartbeats")
	}
}

func TestHealthMonitor_UnhealthyStreams(t *testing.T) {
	hm := NewHealthMonitor(5 * time.Millisecond)
	hm.Register("s1")
	hm.Register("s2")
	hm.Heartbeat("s1")
	hm.Heartbeat("s2")
	time.Sleep(10 * time.Millisecond)

	unhealthy := hm.UnhealthyStreams()
	if len(unhealthy) != 2 {
		t.Errorf("expected 2 unhealthy, got %d: %v", len(unhealthy), unhealthy)
	}
}
