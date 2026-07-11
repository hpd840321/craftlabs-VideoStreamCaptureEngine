package metrics

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetrics_Registered(t *testing.T) {
	StreamStatus.WithLabelValues("probe").Set(0)
	FramesTotal.WithLabelValues("probe", "pass")
	FrameLatencyMs.WithLabelValues("probe")
	DecodeErrorsTotal.WithLabelValues("probe", "test")
	ReconnectTotal.WithLabelValues("probe")
	FFmpegRestartsTotal.WithLabelValues("probe")
	KafkaWriteLatencyMs.WithLabelValues("probe")
	KafkaWriteErrorsTotal.WithLabelValues("probe")

	metrics, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Gather() error = %v", err)
	}

	expectedNames := map[string]bool{
		"vce_stream_status":            false,
		"vce_frames_total":             false,
		"vce_frame_latency_ms":         false,
		"vce_decode_errors_total":      false,
		"vce_reconnect_total":          false,
		"vce_ffmpeg_restarts_total":    false,
		"vce_kafka_write_latency_ms":   false,
		"vce_kafka_write_errors_total": false,
	}

	for _, mf := range metrics {
		if _, ok := expectedNames[mf.GetName()]; ok {
			expectedNames[mf.GetName()] = true
		}
	}

	for name, found := range expectedNames {
		if !found {
			t.Errorf("metric %s not found in gathered metrics", name)
		}
	}
}

func TestStreamStatus_LabelValues(t *testing.T) {
	StreamStatus.Reset()
	StreamStatus.WithLabelValues("cam-1").Set(1)
	StreamStatus.WithLabelValues("cam-2").Set(0)

	err := testutil.CollectAndCompare(StreamStatus,
		strings.NewReader(`
# HELP vce_stream_status Stream status: 1=running, 0=stopped
# TYPE vce_stream_status gauge
vce_stream_status{stream_id="cam-1"} 1
vce_stream_status{stream_id="cam-2"} 0
`), "vce_stream_status")
	if err != nil {
		t.Errorf("StreamStatus metric mismatch: %v", err)
	}
}

func TestFramesTotal_Increment(t *testing.T) {
	FramesTotal.Reset()
	FramesTotal.WithLabelValues("test-cam", "pass").Inc()
	FramesTotal.WithLabelValues("test-cam", "pass").Inc()

	err := testutil.CollectAndCompare(FramesTotal,
		strings.NewReader(`
# HELP vce_frames_total Total frames processed by stream and decision
# TYPE vce_frames_total counter
vce_frames_total{decision="pass",stream_id="test-cam"} 2
`), "vce_frames_total")
	if err != nil {
		t.Errorf("FramesTotal metric mismatch: %v", err)
	}
}

func TestDecodeErrorsTotal_LabelCombinations(t *testing.T) {
	DecodeErrorsTotal.Reset()
	DecodeErrorsTotal.WithLabelValues("cam-1", "network").Inc()
	DecodeErrorsTotal.WithLabelValues("cam-1", "decode").Inc()
	DecodeErrorsTotal.WithLabelValues("cam-1", "network").Inc()

	err := testutil.CollectAndCompare(DecodeErrorsTotal,
		strings.NewReader(`
# HELP vce_decode_errors_total Total decode errors by stream and error type
# TYPE vce_decode_errors_total counter
vce_decode_errors_total{error_type="decode",stream_id="cam-1"} 1
vce_decode_errors_total{error_type="network",stream_id="cam-1"} 2
`), "vce_decode_errors_total")
	if err != nil {
		t.Errorf("DecodeErrorsTotal metric mismatch: %v", err)
	}
}
