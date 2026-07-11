package decoder

import (
	"bytes"
	"testing"
)

func TestRawVideoReader_ReadFrame_LargeFrame(t *testing.T) {
	width, height := 640, 480
	reader := NewRawVideoReader(width, height)
	frameSize := width * height * 3
	buf := make([]byte, frameSize)
	for i := range buf {
		buf[i] = byte(i % 256)
	}

	img, err := reader.ReadFrame(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}

	if img.Bounds().Dx() != width || img.Bounds().Dy() != height {
		t.Errorf("image bounds = %v, want %dx%d", img.Bounds(), width, height)
	}
}

func TestRawVideoReader_ZeroSizeFrame(t *testing.T) {
	reader := NewRawVideoReader(0, 0)
	img, err := reader.ReadFrame(bytes.NewReader([]byte{}))
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}
	if !img.Bounds().Empty() {
		t.Errorf("expected empty image, got %v", img.Bounds())
	}
}

func TestRawVideoReader_BGRToRGBAPixelOrder(t *testing.T) {
	width, height := 1, 3
	reader := NewRawVideoReader(width, height)
	// pixel0: B=10,G=20,R=30; pixel1: B=40,G=50,R=60; pixel2: B=70,G=80,R=90
	buf := []byte{10, 20, 30, 40, 50, 60, 70, 80, 90}

	img, err := reader.ReadFrame(bytes.NewReader(buf))
	if err != nil {
		t.Fatalf("ReadFrame() error = %v", err)
	}

	expected := []struct{ r, g, b uint32 }{
		{30, 20, 10},
		{60, 50, 40},
		{90, 80, 70},
	}
	for y := 0; y < height; y++ {
		r, g, b, a := img.At(0, y).RGBA()
		e := expected[y]
		if r>>8 != e.r || g>>8 != e.g || b>>8 != e.b {
			t.Errorf("pixel(%d) = (%d,%d,%d), want (%d,%d,%d)",
				y, r>>8, g>>8, b>>8, e.r, e.g, e.b)
		}
		if a>>8 != 255 {
			t.Errorf("pixel(%d) alpha = %d, want 255", y, a>>8)
		}
	}
}
