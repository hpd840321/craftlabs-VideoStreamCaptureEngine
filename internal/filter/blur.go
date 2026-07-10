package filter

import (
	"image"
	"math"
)

type BlurFilter struct {
	threshold float64
}

func NewBlurFilter(threshold float64) *BlurFilter {
	if threshold <= 0 {
		threshold = 100
	}
	return &BlurFilter{threshold: threshold}
}

func (b *BlurFilter) Name() string { return "blur" }

func (b *BlurFilter) Apply(frame image.Image, meta FrameMeta) (FilteredFrame, FilterDecision) {
	score := laplacianVariance(frame)
	if score < b.threshold {
		return FilteredFrame{}, FilterDrop
	}
	return FilteredFrame{Image: frame, Meta: meta, Score: score}, FilterPass
}

func laplacianVariance(img image.Image) float64 {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	gray := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			gray[y*w+x] = 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
		}
	}

	kernel := [][]float64{
		{0, 1, 0},
		{1, -4, 1},
		{0, 1, 0},
	}

	n := (w - 2) * (h - 2)
	responses := make([]float64, n)
	idx := 0
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			sum := 0.0
			for ky := -1; ky <= 1; ky++ {
				for kx := -1; kx <= 1; kx++ {
					sum += gray[(y+ky)*w+(x+kx)] * kernel[ky+1][kx+1]
				}
			}
			responses[idx] = sum
			idx++
		}
	}

	mean := 0.0
	for _, r := range responses {
		mean += r
	}
	mean /= float64(n)

	variance := 0.0
	for _, r := range responses {
		d := r - mean
		variance += d * d
	}
	variance /= float64(n)

	return math.Sqrt(variance)
}
