package filter

import (
	"image"
	"image/color"
	"testing"
)

func TestBlurFilter_PassesSharpImage(t *testing.T) {
	f := NewBlurFilter(100)
	sharp := createSharpImage()
	_, decision := f.Apply(sharp, FrameMeta{StreamID: "test", SeqNum: 1})
	if decision != FilterPass {
		t.Error("sharp image should pass blur filter")
	}
}

func TestBlurFilter_DropsBlurryImage(t *testing.T) {
	f := NewBlurFilter(100)
	blurry := createBlurryImage()
	_, decision := f.Apply(blurry, FrameMeta{StreamID: "test", SeqNum: 1})
	if decision != FilterDrop {
		t.Error("blurry image should be dropped")
	}
}

func TestBlurFilter_Name(t *testing.T) {
	f := NewBlurFilter(100)
	if f.Name() != "blur" {
		t.Errorf("Name() = %s, want blur", f.Name())
	}
}

func createSharpImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			v := uint8(((x / 10) % 2) * 255)
			img.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return img
}

func createBlurryImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 200, 200))
	for y := 0; y < 200; y++ {
		for x := 0; x < 200; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	return img
}
