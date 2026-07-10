package filter

import (
	"image"
	"math/bits"
	"sync"
)

// DuplicateFilter drops frames visually similar to previous frame using dHash.
type DuplicateFilter struct {
	threshold int
	lastHash  uint64
	hasLast   bool
	mu        sync.Mutex
}

func NewDuplicateFilter(threshold int) *DuplicateFilter {
	return &DuplicateFilter{threshold: threshold}
}

func (d *DuplicateFilter) Name() string { return "duplicate" }

func (d *DuplicateFilter) Apply(frame image.Image, meta FrameMeta) (FilteredFrame, FilterDecision) {
	hash := dhash(frame)

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.hasLast && hammingDistance(hash, d.lastHash) <= d.threshold {
		return FilteredFrame{}, FilterDrop
	}

	d.lastHash = hash
	d.hasLast = true
	return FilteredFrame{Image: frame, Meta: meta, Score: 1.0}, FilterPass
}

func dhash(img image.Image) uint64 {
	bounds := img.Bounds()
	width, height := 9, 8
	var hash uint64

	for y := 0; y < height; y++ {
		for x := 0; x < width-1; x++ {
			srcX1 := bounds.Min.X + x*bounds.Dx()/width
			srcX2 := bounds.Min.X + (x+1)*bounds.Dx()/width
			srcY := bounds.Min.Y + y*bounds.Dy()/height

			r1, g1, b1, _ := img.At(srcX1, srcY).RGBA()
			r2, g2, b2, _ := img.At(srcX2, srcY).RGBA()

			gray1 := (r1*299 + g1*587 + b1*114) / 1000
			gray2 := (r2*299 + g2*587 + b2*114) / 1000

			hash <<= 1
			if gray1 > gray2 {
				hash |= 1
			}
		}
	}

	return hash
}

func hammingDistance(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}
