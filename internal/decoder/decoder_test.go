package decoder

import (
	"bytes"
	"testing"
	"time"
)

func TestRawVideoReader_ReadFrame_ValidBGR(t *testing.T) {
	width, height := 2, 2
	frameSize := width * height * 3
	buf := make([]byte, frameSize)
	buf[0], buf[1], buf[2] = 1, 2, 3
	buf[3], buf[4], buf[5] = 4, 5, 6
	buf[6], buf[7], buf[8] = 7, 8, 9
	buf[9], buf[10], buf[11] = 10, 11, 12

	reader := NewRawVideoReader(width, height)
	img, err := reader.ReadFrame(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}

	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		t.Errorf("image bounds = %v, want %dx%d", img.Bounds(), width, height)
	}
}

func TestRawVideoReader_ReadFrame_IncompleteData(t *testing.T) {
	reader := NewRawVideoReader(4, 4)
	buf := make([]byte, 10)

	_, err := reader.ReadFrame(bytes.NewReader(buf))
	if err == nil {
		t.Fatal("expected error for incomplete frame data")
	}
}

func TestNewRawVideoReader_CalculatesFrameSize(t *testing.T) {
	reader := NewRawVideoReader(1920, 1080)
	expected := 1920 * 1080 * 3
	if reader.frameSize != expected {
		t.Errorf("frameSize = %d, want %d", reader.frameSize, expected)
	}
}

func TestFFmpegCommandBuilder_BasicRawVideo(t *testing.T) {
	cfg := FFmpegConfig{
		RTSPURL:     "rtsp://10.0.0.1:554/stream",
		CaptureFPS:  25,
		DecodeScale: "1920x1080",
	}

	args := BuildFFmpegArgs(cfg)
	argStr := joinArgs(args)

	for _, want := range []string{"ffmpeg", "-rtsp_transport tcp", "-f rawvideo", "-pix_fmt bgr24", "pipe:1", "fps=25", "scale=1920x1080"} {
		if !contains(argStr, want) {
			t.Errorf("args missing '%s': %s", want, argStr)
		}
	}
}

func TestFFmpegCommandBuilder_CustomParams(t *testing.T) {
	cfg := FFmpegConfig{
		RTSPURL:     "rtsp://localhost/stream",
		CaptureFPS:  15,
		DecodeScale: "1280x720",
	}

	args := BuildFFmpegArgs(cfg)
	argStr := joinArgs(args)

	if !contains(argStr, "fps=15") {
		t.Error("missing fps=15")
	}
	if !contains(argStr, "scale=1280x720") {
		t.Error("missing scale=1280:720")
	}
}

func TestExponentialBackoff_Sequence(t *testing.T) {
	b := NewExponentialBackoff(ExponentialBackoffConfig{
		Initial: 1 * time.Second,
		Max:     60 * time.Second,
		Factor:  2.0,
	})

	expected := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second, 32 * time.Second, 60 * time.Second, 60 * time.Second}
	for i, want := range expected {
		got := b.NextDelay()
		if got != want {
			t.Errorf("attempt %d: NextDelay() = %v, want %v", i+1, got, want)
		}
	}
}

func TestExponentialBackoff_Reset(t *testing.T) {
	b := NewExponentialBackoff(ExponentialBackoffConfig{
		Initial: 1 * time.Second,
		Max:     60 * time.Second,
		Factor:  2.0,
	})
	b.NextDelay()
	b.NextDelay()
	b.Reset()
	if got := b.NextDelay(); got != 1*time.Second {
		t.Errorf("after Reset: %v, want 1s", got)
	}
}

func joinArgs(args []string) string {
	result := ""
	for _, a := range args {
		result += a + " "
	}
	return result
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
