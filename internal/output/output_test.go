package output

import (
	"image"
	"image/color"
	"testing"
	"time"
)

func TestFrameSerializer_JPEG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}

	s := NewFrameSerializer(85)
	frame := &Frame{
		Image: img,
		Meta:  FrameMeta{StreamID: "test", SeqNum: 42, Timestamp: time.Unix(1700000000, 0)},
		Score: 0.95,
	}

	out, err := s.Serialize(frame)
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	if out.StreamID != "test" {
		t.Errorf("StreamID = %s, want test", out.StreamID)
	}
	if out.SeqNum != 42 {
		t.Errorf("SeqNum = %d, want 42", out.SeqNum)
	}
	if out.Quality != 0.95 {
		t.Errorf("Quality = %f, want 0.95", out.Quality)
	}
	if len(out.ImageData) == 0 {
		t.Error("ImageData empty")
	}
	if out.ImageSize != len(out.ImageData) {
		t.Errorf("ImageSize = %d, want %d", out.ImageSize, len(out.ImageData))
	}

	if out.ImageData[0] != 0xFF || out.ImageData[1] != 0xD8 {
		t.Error("not JPEG SOI marker")
	}
}

func TestFrameSerializer_QualityBounds(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	serializer1 := NewFrameSerializer(0)
	if serializer1.quality != 1 {
		t.Error("quality should be clamped to 1")
	}
	serializer2 := NewFrameSerializer(101)
	if serializer2.quality != 100 {
		t.Error("quality should be clamped to 100")
	}
	_ = img
}

func TestResolveTopic(t *testing.T) {
	tests := []struct {
		prefix, streamID, outputTopic, want string
	}{
		{"frames", "cam-1", "", "frames.cam-1"},
		{"frames", "cam-1", "custom", "custom"},
		{"", "cam-1", "explicit", "explicit"},
		{"", "cam-1", "", "cam-1"},
	}
	for _, tt := range tests {
		got := ResolveTopic(tt.prefix, tt.streamID, tt.outputTopic)
		if got != tt.want {
			t.Errorf("ResolveTopic(%q,%q,%q) = %q, want %q", tt.prefix, tt.streamID, tt.outputTopic, got, tt.want)
		}
	}
}
