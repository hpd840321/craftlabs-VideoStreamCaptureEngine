package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	StreamStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "vce_stream_status",
			Help: "Stream status: 1=running, 0=stopped",
		},
		[]string{"stream_id"},
	)

	FramesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vce_frames_total",
			Help: "Total frames processed by stream and decision",
		},
		[]string{"stream_id", "decision"},
	)

	FrameLatencyMs = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "vce_frame_latency_ms",
			Help:    "Frame decode-to-output latency in milliseconds",
			Buckets: []float64{10, 25, 50, 100, 250, 500, 1000, 2000},
		},
		[]string{"stream_id"},
	)

	DecodeErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vce_decode_errors_total",
			Help: "Total decode errors by stream and error type",
		},
		[]string{"stream_id", "error_type"},
	)

	ReconnectTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vce_reconnect_total",
			Help: "Total ffmpeg reconnection attempts",
		},
		[]string{"stream_id"},
	)

	FFmpegRestartsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vce_ffmpeg_restarts_total",
			Help: "Total ffmpeg process restarts",
		},
		[]string{"stream_id"},
	)

	KafkaWriteLatencyMs = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "vce_kafka_write_latency_ms",
			Help:    "Kafka write latency in milliseconds",
			Buckets: []float64{5, 10, 25, 50, 100, 250, 500},
		},
		[]string{"stream_id"},
	)

	KafkaWriteErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vce_kafka_write_errors_total",
			Help: "Total Kafka write errors",
		},
		[]string{"stream_id"},
	)
)
