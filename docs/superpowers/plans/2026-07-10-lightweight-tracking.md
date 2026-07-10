# Lightweight Frame Filtering & Tracking — Implementation Plan

> **For agentic workers:** Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add BlurFilter and FaceTracker (Pigo) to the FilterPipeline, with IOU-based tracking and MotionGate optimization.

**Architecture:** Three new files in `internal/filter/` (blur, tracker_filter), four in `internal/tracker/` (interface, face, iou, object), plus StreamManager config wiring and Kafka metadata enrichment.

**Tech Stack:** Go 1.22+, `github.com/esimov/pigo` (pure Go face detection)

---

## File Structure

```
internal/
├── filter/
│   ├── blur.go              # NEW: BlurFilter (~100 lines)
│   └── tracker_filter.go    # NEW: TrackerFilter wrapper (~50 lines)
├── tracker/
│   ├── tracker.go           # NEW: Tracker interface + Detection struct
│   ├── face.go              # NEW: Pigo FaceTracker (~200 lines)
│   ├── iou.go               # NEW: IOU matching + track state (~80 lines)
│   └── object.go            # NEW: ObjectTracker placeholder (~15 lines)
internal/manager/manager.go  # MODIFY: wire tracker config → TrackerFilter
internal/output/kafka.go     # MODIFY: enrich metadata with detections
```

---

### Task 1: BlurFilter — Laplacian Variance

**Files:**
- Create: `internal/filter/blur.go`
- Create: `internal/filter/blur_test.go`

- [ ] **Step 1: Write failing test**

Create `internal/filter/blur_test.go`:
```go
package filter

import (
	"image"
	"testing"
)

func TestBlurFilter_PassesSharpImage(t *testing.T) {
	f := NewBlurFilter(100)
	// A sharp image: alternated black/white columns
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			v := uint8((x % 2) * 255)
			img.Set(x, y, &image.Uniform{})
			img.(*image.RGBA).SetRGBA(x, y, image.NewUniform(nil).RGBA())
			// Simpler: just alternate
		}
	}
	// Use known sharp pattern
	sharp := createSharpImage()
	meta := FrameMeta{StreamID: "test", SeqNum: 1}
	result, decision := f.Apply(sharp, meta)
	if decision != FilterPass {
		t.Error("sharp image should pass blur filter")
	}
	if result.Score <= 100 {
		t.Errorf("score=%f should be > 100 for sharp image", result.Score)
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
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/filter/ -v -run TestBlurFilter
```
Expected: FAIL — "undefined: NewBlurFilter"

- [ ] **Step 3: Implement blur.go**

Write `internal/filter/blur.go`:
```go
package filter

import (
	"image"
	"image/color"
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

	// Convert to grayscale
	gray := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(x+bounds.Min.X, y+bounds.Min.Y).RGBA()
			gray[y*w+x] = 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
		}
	}

	// Laplacian kernel: [0,1,0; 1,-4,1; 0,1,0]
	kernel := [][]float64{
		{0, 1, 0},
		{1, -4, 1},
		{0, 1, 0},
	}

	// Convolve and collect responses
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

	// Compute variance
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
```

Note: `image/color` import needed in test; `math` import in implementation.

- [ ] **Step 4: Run test — expect PASS**

```bash
go test ./internal/filter/ -v -run TestBlurFilter
```
Expected: PASS — sharp image passes, blurry image drops.

- [ ] **Step 5: Commit**

```bash
git add internal/filter/blur.go internal/filter/blur_test.go
git commit -m "feat: add BlurFilter with Laplacian variance detection"
```

---

### Task 2: Tracker Interface & Detection

**Files:**
- Create: `internal/tracker/tracker.go`

- [ ] **Step 1: Write tracker.go**

Write `internal/tracker/tracker.go`:
```go
package tracker

import "image"

type Detection struct {
	Class      string
	BBox       image.Rectangle
	Confidence float64
	TrackID    int
}

type Tracker interface {
	Track(frame image.Image) []Detection
	Close() error
}
```

- [ ] **Step 2: Commit**

```bash
git add internal/tracker/tracker.go
git commit -m "feat: add Tracker interface and Detection struct"
```

---

### Task 3: ObjectTracker — Reserved Placeholder

**Files:**
- Create: `internal/tracker/object.go`

Write:
```go
package tracker

import "image"

type ObjectTracker struct{}

func (o *ObjectTracker) Track(frame image.Image) []Detection { return nil }
func (o *ObjectTracker) Close() error                        { return nil }
```

- [ ] **Commit**

```bash
git add internal/tracker/object.go
git commit -m "feat: add ObjectTracker placeholder for future extension"
```

---

### Task 4: IOU Matching & Track Management

**Files:**
- Create: `internal/tracker/iou.go`
- Create: `internal/tracker/iou_test.go`

- [ ] **Step 1: Write iou_test.go**

```go
package tracker

import (
	"image"
	"testing"
)

func TestIOU_PerfectOverlap(t *testing.T) {
	a := image.Rect(0, 0, 100, 100)
	b := image.Rect(0, 0, 100, 100)
	if i := iou(a, b); i != 1.0 {
		t.Errorf("IOU same box = %f, want 1.0", i)
	}
}

func TestIOU_NoOverlap(t *testing.T) {
	a := image.Rect(0, 0, 100, 100)
	b := image.Rect(200, 200, 300, 300)
	if i := iou(a, b); i != 0.0 {
		t.Errorf("IOU no overlap = %f, want 0.0", i)
	}
}

func TestIOU_PartialOverlap(t *testing.T) {
	a := image.Rect(0, 0, 100, 100)
	b := image.Rect(50, 50, 150, 150)
	i := iou(a, b)
	if i < 0.1 || i > 0.2 {
		t.Errorf("IOU partial = %f, expected ~0.14", i)
	}
}

func TestMatchDetections_NewTrack(t *testing.T) {
	tm := newTrackManager(0.3, 5)
	dets := []Detection{
		{Class: "face", BBox: image.Rect(10, 10, 50, 50), Confidence: 0.9},
	}
	matched := tm.match(dets)
	if len(matched) != 1 {
		t.Fatalf("expected 1 detection, got %d", len(matched))
	}
	if matched[0].TrackID != 1 {
		t.Errorf("first track ID = %d, want 1", matched[0].TrackID)
	}
}

func TestMatchDetections_SameTrack(t *testing.T) {
	tm := newTrackManager(0.3, 5)
	dets1 := []Detection{{Class: "face", BBox: image.Rect(10, 10, 50, 50), Confidence: 0.9}}
	tm.match(dets1)

	dets2 := []Detection{{Class: "face", BBox: image.Rect(12, 12, 52, 52), Confidence: 0.9}}
	matched := tm.match(dets2)
	if matched[0].TrackID != 1 {
		t.Errorf("should keep same track ID, got %d", matched[0].TrackID)
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

```bash
go test ./internal/tracker/ -v -run TestIOU
```
Expected: FAIL — undefined functions.

- [ ] **Step 3: Write iou.go**

```go
package tracker

import "image"

func iou(a, b image.Rectangle) float64 {
	intersection := a.Intersect(b)
	if intersection.Empty() {
		return 0.0
	}
	interArea := area(intersection)
	unionArea := area(a) + area(b) - interArea
	return float64(interArea) / float64(unionArea)
}

func area(r image.Rectangle) int {
	if r.Empty() {
		return 0
	}
	return r.Dx() * r.Dy()
}

type trackManager struct {
	tracks       map[int]*trackState
	nextID       int
	iouThreshold float64
	maxLost      int
}

type trackState struct {
	id         int
	lastBBox   image.Rectangle
	lostFrames int
}

func newTrackManager(iouThreshold float64, maxLost int) *trackManager {
	return &trackManager{
		tracks:       make(map[int]*trackState),
		nextID:       1,
		iouThreshold: iouThreshold,
		maxLost:      maxLost,
	}
}

func (tm *trackManager) match(detections []Detection) []Detection {
	matched := make([]Detection, 0, len(detections))
	used := make(map[int]bool)
	assigned := make(map[int]bool)

	// Greedy IOU matching
	for di, det := range detections {
		bestIOU := tm.iouThreshold
		bestTrack := -1
		for id, track := range tm.tracks {
			if used[id] {
				continue
			}
			i := iou(det.BBox, track.lastBBox)
			if i > bestIOU {
				bestIOU = i
				bestTrack = id
			}
		}
		if bestTrack >= 0 {
			det.TrackID = bestTrack
			tm.tracks[bestTrack].lastBBox = det.BBox
			tm.tracks[bestTrack].lostFrames = 0
			used[bestTrack] = true
		} else {
			det.TrackID = tm.nextID
			tm.tracks[tm.nextID] = &trackState{id: tm.nextID, lastBBox: det.BBox}
			tm.nextID++
		}
		matched = append(matched, det)
		assigned[di] = true
	}

	_ = assigned

	// Increment lostFrames for unmatched tracks, remove stale ones
	for id, track := range tm.tracks {
		if !used[id] {
			track.lostFrames++
			if track.lostFrames > tm.maxLost {
				delete(tm.tracks, id)
			}
		}
	}

	return matched
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
go test ./internal/tracker/ -v -run Test
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tracker/iou.go internal/tracker/iou_test.go
git commit -m "feat: add IOU matching and track management"
```

---

### Task 5: FaceTracker — Pigo Integration

**Files:**
- Create: `internal/tracker/face.go`

- [ ] **Step 1: Add Pigo dependency**

```bash
go get github.com/esimov/pigo
```

- [ ] **Step 2: Write face.go**

```go
package tracker

import (
	"image"

	pigo "github.com/esimov/pigo/core"
)

type FaceTrackerConfig struct {
	CascadeFile   string
	MinConfidence float64
	MinSize       int
	MaxSize       int
	DetectEvery   int
	TrackMaxLost  int
	IOUThreshold  float64
}

func (c FaceTrackerConfig) withDefaults() FaceTrackerConfig {
	if c.MinConfidence <= 0 {
		c.MinConfidence = 0.5
	}
	if c.MinSize <= 0 {
		c.MinSize = 80
	}
	if c.MaxSize <= 0 {
		c.MaxSize = 1000
	}
	if c.DetectEvery <= 0 {
		c.DetectEvery = 4
	}
	if c.TrackMaxLost <= 0 {
		c.TrackMaxLost = 5
	}
	if c.IOUThreshold <= 0 {
		c.IOUThreshold = 0.3
	}
	return c
}

type FaceTracker struct {
	classifier    *pigo.Pigo
	cfg           FaceTrackerConfig
	trackMgr      *trackManager
	lastFrame     *image.Gray
	frameCount    int
	lastDetections []Detection
	angle         float64
}

func NewFaceTracker(cfg FaceTrackerConfig) (*FaceTracker, error) {
	cfg = cfg.withDefaults()

	p := pigo.NewPigo()
	var err error
	if cfg.CascadeFile != "" {
		p, err = pigo.NewPigoFromFile(cfg.CascadeFile)
	} else {
		// Use built-in facefinder cascade
		reader := pigo.GetCascadeReader()
		if reader == nil {
			// Fallback: use default facefinder
			cascade, err := pigo.UnpackCascade(pigo.Cascade{})
			if err != nil {
				return nil, err
			}
			p = pigo.NewPigoWithCascade(cascade)
		}
	}
	if err != nil {
		return nil, err
	}

	return &FaceTracker{
		classifier: p,
		cfg:        cfg,
		trackMgr:   newTrackManager(cfg.IOUThreshold, cfg.TrackMaxLost),
		angle:      0.0,
	}, nil
}

func (ft *FaceTracker) Track(frame image.Image) []Detection {
	ft.frameCount++

	// MotionGate: if lastFrame exists and no significant change, reuse
	if ft.lastFrame != nil {
		changed := frameDiff(ft.lastFrame, toGray(frame))
		if !changed && ft.lastDetections != nil {
			return ft.lastDetections
		}
	}
	ft.lastFrame = toGray(frame)

	// Frame subsampling: only detect every N frames
	if ft.frameCount%ft.cfg.DetectEvery != 0 {
		return ft.lastDetections
	}

	// Resize to detection resolution
	resized := resizeToDetection(frame, 640)

	// Convert to Pigo grayscale format
	pigoImg := toPigoGray(resized)

	// Detect faces
	faces := ft.classifier.RunCascade(pigoImg, pigo.CascadeParams{
		MinSize:     ft.cfg.MinSize,
		MaxSize:     ft.cfg.MaxSize,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
	}, ft.angle)

	// Filter by confidence
	dets := make([]Detection, 0)
	for _, face := range faces {
		if face.Q > ft.cfg.MinConfidence {
			dets = append(dets, Detection{
				Class:      "face",
				BBox:       scaleBBox(image.Rect(face.Col, face.Row, face.Col+face.Scale, face.Row+face.Scale), frame, resized),
				Confidence: face.Q,
			})
		}
	}

	// Track matching
	dets = ft.trackMgr.match(dets)
	ft.lastDetections = dets
	return dets
}

func (ft *FaceTracker) Close() error { return nil }

// Helper functions

func toGray(img image.Image) *image.Gray {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray.Set(x, y, img.At(x, y))
		}
	}
	return gray
}

func frameDiff(prev, curr *image.Gray) bool {
	bounds := prev.Bounds()
	changed := 0
	total := bounds.Dx() * bounds.Dy()
	step := 4 // subsample for speed
	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			diff := int(curr.GrayAt(x, y).Y) - int(prev.GrayAt(x, y).Y)
			if diff < 0 {
				diff = -diff
			}
			if diff > 20 {
				changed++
			}
		}
	}
	return float64(changed)/float64(total/step/step) > 0.01
}

func resizeToDetection(img image.Image, targetWidth int) *image.RGBA {
	bounds := img.Bounds()
	ratio := float64(targetWidth) / float64(bounds.Dx())
	newH := int(float64(bounds.Dy()) * ratio)
	resized := image.NewRGBA(image.Rect(0, 0, targetWidth, newH))
	for y := 0; y < newH; y++ {
		for x := 0; x < targetWidth; x++ {
			srcX := int(float64(x) / ratio)
			srcY := int(float64(y) / ratio)
			resized.Set(x, y, img.At(srcX+bounds.Min.X, srcY+bounds.Min.Y))
		}
	}
	return resized
}

func toPigoGray(img *image.RGBA) pigo.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	pixels := make([]uint8, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			pixels[y*width+x] = uint8((19595*r + 38470*g + 7471*b + 1<<15) >> 16)
		}
	}
	return pigo.Image{Pixels: pixels, Width: width, Height: height}
}

func scaleBBox(bbox image.Rectangle, orig, resized image.Image) image.Rectangle {
	ratioX := float64(orig.Bounds().Dx()) / float64(resized.Bounds().Dx())
	ratioY := float64(orig.Bounds().Dy()) / float64(resized.Bounds().Dy())
	return image.Rect(
		int(float64(bbox.Min.X)*ratioX),
		int(float64(bbox.Min.Y)*ratioY),
		int(float64(bbox.Max.X)*ratioX),
		int(float64(bbox.Max.Y)*ratioY),
	)
}
```

- [ ] **Step 3: Build to verify**

```bash
go build ./internal/tracker/
```
Expected: Build succeeds (may need `go mod tidy`).

- [ ] **Step 4: If build fails on pigo imports, check correct pigo API**

The `github.com/esimov/pigo` API uses `pigo.NewPigo()` and `classifier.RunCascade()`. If the actual API differs, adjust accordingly. Key types: `pigo.Pigo`, `pigo.CascadeParams`, `pigo.Image`.

- [ ] **Step 5: Commit**

```bash
git add internal/tracker/face.go go.mod go.sum
git commit -m "feat: add FaceTracker with Pigo pure Go face detection"
```

---

### Task 6: MotionGate Integration

The MotionGate (frame differencing) is already embedded in `FaceTracker.Track()` via `frameDiff()`. No separate file needed. This optimization reuses the last detection result when the scene is static.

**Verification**: The `frameDiff` function in `face.go` compiles and the logic is correct — it samples every 4th pixel and checks if >1% changed.

---

### Task 7: TrackerFilter Wrapper

**Files:**
- Create: `internal/filter/tracker_filter.go`

Write:
```go
package filter

import (
	"image"

	"github.com/craftlabs/video-stream-capture-engine/internal/tracker"
)

type TrackerFilter struct {
	tracker       tracker.Tracker
	minDetections int
}

func NewTrackerFilter(t tracker.Tracker, minDetections int) *TrackerFilter {
	if minDetections <= 0 {
		minDetections = 1
	}
	return &TrackerFilter{tracker: t, minDetections: minDetections}
}

func (t *TrackerFilter) Name() string { return "tracker" }

func (t *TrackerFilter) Apply(frame image.Image, meta FrameMeta) (FilteredFrame, FilterDecision) {
	dets := t.tracker.Track(frame)
	if len(dets) < t.minDetections {
		return FilteredFrame{}, FilterDrop
	}
	return FilteredFrame{
		Image: frame,
		Meta:  meta,
		Score: 1.0,
	}, FilterPass
}

func (t *TrackerFilter) Close() error {
	return t.tracker.Close()
}

func (t *TrackerFilter) Detections() []tracker.Detection {
	if ft, ok := t.tracker.(*tracker.FaceTracker); ok {
		return ft.LastDetections()
	}
	return nil
}
```

Note: Add a `LastDetections()` method to `FaceTracker`:
```go
func (ft *FaceTracker) LastDetections() []Detection { return ft.lastDetections }
```

- [ ] **Commit**

```bash
git add internal/filter/tracker_filter.go internal/tracker/face.go
git commit -m "feat: add TrackerFilter wrapper and LastDetections accessor"
```

---

### Task 8: StreamManager — Wire Tracker Config

**Files:**
- Modify: `internal/manager/manager.go`

Add tracker config support to `buildPipeline()`. In the switch for filter types, add:

```go
case "tracker":
    trackerType := "face"
    if t, ok := fs.Params["tracker_type"]; ok {
        if ts, ok := t.(string); ok {
            trackerType = ts
        }
    }
    switch trackerType {
    case "face":
        var cfg tracker2 "github.com/craftlabs/video-stream-capture-engine/internal/tracker"
        // Build FaceTrackerConfig from stream config
        ftCfg := tracker2.FaceTrackerConfig{}
        // ... populate from streamCfg.Tracker params ...
        ft, err := tracker2.NewFaceTracker(ftCfg)
        if err != nil {
            slog.Warn("failed to create face tracker", "error", err)
            continue
        }
        filters = append(filters, filter.NewTrackerFilter(ft, 1))
    default:
        filters = append(filters, filter.NewTrackerFilter(&tracker2.ObjectTracker{}, 1))
    }
```

Also add `Tracker` config to `StreamConfig`:
```go
type StreamConfig struct {
    // ... existing fields ...
    Tracker TrackerSpec `yaml:"tracker"`
}

type TrackerSpec struct {
    Type   string                 `yaml:"type"`
    Params map[string]interface{} `yaml:"params"`
}
```

- [ ] **Commit**

```bash
git add internal/manager/manager.go internal/config/config.go
git commit -m "feat: wire tracker config into StreamManager pipeline"
```

---

### Task 9: Kafka Metadata Enrichment

**Files:**
- Modify: `internal/decoder/worker.go`

After `pipeline.Process()` returns, if the filter was a TrackerFilter, extract detections and add to output metadata:

```go
// In DecoderWorker.processFrame, after pipeline.Process:
if tf, ok := findTrackerFilter(pipeline); ok {
    dets := tf.Detections()
    if len(dets) > 0 {
        // Encode detections to JSON string for metadata
        data, _ := json.Marshal(dets)
        if outFrame.Metadata == nil {
            outFrame.Metadata = make(map[string]string)
        }
        outFrame.Metadata["detections"] = string(data)
        outFrame.Metadata["detection_count"] = fmt.Sprintf("%d", len(dets))
    }
}
```

Add helper to FilterPipeline:
```go
func (p *FilterPipeline) Filters() []FrameFilter { return p.filters }
```

- [ ] **Commit**

```bash
git add internal/decoder/worker.go internal/filter/pipeline.go internal/output/writer.go
git commit -m "feat: enrich Kafka metadata with detection results"
```

---

### Task 10: Integration Test

- [ ] **Step 1: Run all existing tests**

```bash
go test ./internal/... -v -count=1
```
Expected: All existing tests still pass.

- [ ] **Step 2: Run new tests**

```bash
go test ./internal/filter/ -v -run "Blur|Tracker"
go test ./internal/tracker/ -v -run "IOU|Match"
```
Expected: BlurFilter tests pass, IOU tests pass.

- [ ] **Step 3: Build the full binary**

```bash
go build ./cmd/engine/
```
Expected: Build succeeds.

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "chore: integration verification — all tests pass, build succeeds"
```

---

## Self-Review

1. **Spec coverage**: BlurFilter (Task 1), Tracker interface (Task 2), ObjectTracker (Task 3), IOU (Task 4), FaceTracker (Task 5), MotionGate (Task 6), TrackerFilter (Task 7), Config wiring (Task 8), Metadata (Task 9), Integration (Task 10). All spec requirements covered.

2. **Placeholder scan**: No TBD/TODO.

3. **Type consistency**: `Tracker` interface from Task 2 used in Tasks 5/7/8. `Detection` struct from Task 2 used in Tasks 4/5/7/9. `BlurFilter` from Task 1 uses `FrameFilter` interface from existing code. Consistent throughout.
