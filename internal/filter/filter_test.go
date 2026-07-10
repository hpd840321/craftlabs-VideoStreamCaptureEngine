package filter

import (
	"image"
	"image/color"
	"testing"
	"time"
)

func TestNoopFilter_PassesFrame(t *testing.T) {
	f := &NoopFilter{}
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	meta := FrameMeta{StreamID: "test", SeqNum: 1, Timestamp: time.Now()}

	result, decision := f.Apply(img, meta)
	if decision != FilterPass {
		t.Errorf("decision = %v, want FilterPass", decision)
	}
	if result.Image != img {
		t.Error("image should pass through unchanged")
	}
	if result.Meta.StreamID != "test" {
		t.Errorf("StreamID = %s, want test", result.Meta.StreamID)
	}
}

func TestNoopFilter_Name(t *testing.T) {
	f := &NoopFilter{}
	if f.Name() != "noop" {
		t.Errorf("Name() = %s, want noop", f.Name())
	}
}

func TestDuplicateFilter_PassesUniqueFrame(t *testing.T) {
	f := NewDuplicateFilter(10)
	img1 := createTestImage(0, 0, 100)
	result, decision := f.Apply(img1, FrameMeta{StreamID: "s1", SeqNum: 1})
	if decision != FilterPass {
		t.Errorf("first frame: %v, want FilterPass", decision)
	}
	if result.Score != 1.0 {
		t.Errorf("score = %f, want 1.0", result.Score)
	}

	img2 := createTestImage(200, 200, 100)
	_, decision2 := f.Apply(img2, FrameMeta{StreamID: "s1", SeqNum: 2})
	if decision2 != FilterPass {
		t.Errorf("different frame: %v, want FilterPass", decision2)
	}
}

func TestDuplicateFilter_DropsDuplicate(t *testing.T) {
	f := NewDuplicateFilter(20) // Use higher threshold for test reliability
	img1 := createTestImage(0, 0, 100)
	f.Apply(img1, FrameMeta{StreamID: "s1", SeqNum: 1})

	img2 := createTestImage(0, 0, 100)
	_, decision := f.Apply(img2, FrameMeta{StreamID: "s1", SeqNum: 2})
	if decision != FilterDrop {
		t.Errorf("duplicate frame: %v, want FilterDrop", decision)
	}
}

func TestHammingDistance(t *testing.T) {
	tests := []struct{ a, b uint64; want int }{
		{0, 0, 0},
		{1, 0, 1},
		{0xFF, 0x00, 8},
	}
	for _, tt := range tests {
		got := hammingDistance(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("hammingDistance(%#x, %#x) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestFilterPipeline_AllPass(t *testing.T) {
	p := NewFilterPipeline([]FrameFilter{&NoopFilter{}, &NoopFilter{}})
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	result, ok := p.Process(img, FrameMeta{StreamID: "s1"})
	if !ok {
		t.Fatal("expected frame to pass")
	}
	if result.Image != img {
		t.Error("image should pass unchanged")
	}
}

func TestFilterPipeline_DropStopsChain(t *testing.T) {
	dropFilter := &dropAlwaysFilter{}
	p := NewFilterPipeline([]FrameFilter{dropFilter, &NoopFilter{}})
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	_, ok := p.Process(img, FrameMeta{StreamID: "s1"})
	if ok {
		t.Fatal("expected frame to be dropped")
	}
}

func createTestImage(offsetX, offsetY, size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			r := uint8((x + offsetX) % 256)
			g := uint8((y + offsetY) % 256)
			b := uint8(((x + offsetX + y + offsetY) / 2) % 256)
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

type dropAlwaysFilter struct{}

func (d *dropAlwaysFilter) Name() string                         { return "drop" }
func (d *dropAlwaysFilter) Apply(_ image.Image, _ FrameMeta) (FilteredFrame, FilterDecision) {
	return FilteredFrame{}, FilterDrop
}
