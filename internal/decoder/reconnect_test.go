package decoder

import (
	"testing"
	"time"
)

func TestExponentialBackoff_MaxDelayConsistency(t *testing.T) {
	b := NewExponentialBackoff(ExponentialBackoffConfig{
		Initial: 1 * time.Second,
		Max:     10 * time.Second,
		Factor:  3.0,
	})

	// After reaching max, should stay at max
	for i := 0; i < 5; i++ {
		delay := b.NextDelay()
		if delay > 10*time.Second {
			t.Errorf("attempt %d: delay = %v, want <= 10s", i+1, delay)
		}
	}
}

func TestExponentialBackoff_CustomConfig(t *testing.T) {
	b := NewExponentialBackoff(ExponentialBackoffConfig{
		Initial: 500 * time.Millisecond,
		Max:     30 * time.Second,
		Factor:  1.5,
	})

	expected := []time.Duration{
		500 * time.Millisecond,
		750 * time.Millisecond,
		1125 * time.Millisecond,
	}

	for i, want := range expected {
		got := b.NextDelay()
		if got != want {
			t.Errorf("attempt %d: delay = %v, want %v", i+1, got, want)
		}
	}
}

func TestExponentialBackoff_ResetAfterMax(t *testing.T) {
	b := NewExponentialBackoff(ExponentialBackoffConfig{
		Initial: 1 * time.Second,
		Max:     5 * time.Second,
		Factor:  2.0,
	})

	// Drive to max
	for i := 0; i < 5; i++ {
		b.NextDelay()
	}
	// Reset
	b.Reset()
	got := b.NextDelay()
	if got != 1*time.Second {
		t.Errorf("after reset: delay = %v, want 1s", got)
	}
}
