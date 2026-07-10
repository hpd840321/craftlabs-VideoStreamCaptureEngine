package decoder

import (
	"math"
	"time"
)

type ExponentialBackoffConfig struct {
	Initial time.Duration
	Max     time.Duration
	Factor  float64
}

func (c ExponentialBackoffConfig) withDefaults() ExponentialBackoffConfig {
	if c.Initial <= 0 {
		c.Initial = 1 * time.Second
	}
	if c.Max <= 0 {
		c.Max = 60 * time.Second
	}
	if c.Factor <= 0 {
		c.Factor = 2.0
	}
	return c
}

type ExponentialBackoff struct {
	config  ExponentialBackoffConfig
	attempt int
}

func NewExponentialBackoff(config ExponentialBackoffConfig) *ExponentialBackoff {
	return &ExponentialBackoff{config: config.withDefaults()}
}

func (b *ExponentialBackoff) NextDelay() time.Duration {
	b.attempt++
	delay := time.Duration(float64(b.config.Initial) * math.Pow(b.config.Factor, float64(b.attempt-1)))
	if delay > b.config.Max {
		delay = b.config.Max
	}
	return delay
}

func (b *ExponentialBackoff) Reset() {
	b.attempt = 0
}

func (b *ExponentialBackoff) Sleep() {
	time.Sleep(b.NextDelay())
}
