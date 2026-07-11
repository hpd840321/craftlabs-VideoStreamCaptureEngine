package tracker

import (
	"embed"
	"image"
	"image/color"
	"os"

	pigo "github.com/esimov/pigo/core"
)

//go:embed cascade/facefinder
var defaultCascade embed.FS

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
	classifier     *pigo.Pigo
	cfg            FaceTrackerConfig
	trackMgr       *trackManager
	lastGray       *image.Gray
	frameCount     int
	lastDetections []Detection
}

func NewFaceTracker(cfg FaceTrackerConfig) (*FaceTracker, error) {
	cfg = cfg.withDefaults()

	pg := pigo.NewPigo()

	var cascadeData []byte
	var err error
	if cfg.CascadeFile != "" {
		cascadeData, err = os.ReadFile(cfg.CascadeFile)
	} else {
		cascadeData, err = defaultCascade.ReadFile("cascade/facefinder")
	}
	if err != nil {
		return nil, err
	}

	classifier, err := pg.Unpack(cascadeData)
	if err != nil {
		return nil, err
	}

	return &FaceTracker{
		classifier: classifier,
		cfg:        cfg,
		trackMgr:   newTrackManager(cfg.IOUThreshold, cfg.TrackMaxLost),
	}, nil
}

func (ft *FaceTracker) Track(frame image.Image) []Detection {
	ft.frameCount++

	gray := toGrayImage(frame)
	if ft.lastGray != nil && !frameChanged(ft.lastGray, gray) && ft.lastDetections != nil {
		ft.lastGray = gray
		return ft.lastDetections
	}
	ft.lastGray = gray

	if ft.frameCount%ft.cfg.DetectEvery != 0 {
		return ft.lastDetections
	}

	bounds := frame.Bounds()
	ratio := 640.0 / float64(bounds.Dx())
	detH := int(float64(bounds.Dy()) * ratio)
	resized := resizeRGBA(frame, 640, detH)

	grayPixels := pigo.RgbToGrayscale(resized)

	params := pigo.CascadeParams{
		MinSize:     ft.cfg.MinSize,
		MaxSize:     ft.cfg.MaxSize,
		ShiftFactor: 0.1,
		ScaleFactor: 1.05,
		ImageParams: pigo.ImageParams{
			Pixels: grayPixels,
			Rows:   detH,
			Cols:   640,
			Dim:    640,
		},
	}

	faces := ft.classifier.RunCascade(params, 0.0)

	dets := make([]Detection, 0)
	for _, face := range faces {
		if float64(face.Q) > ft.cfg.MinConfidence {
			invRatio := float64(bounds.Dx()) / 640.0
			dets = append(dets, Detection{
				Class: "face",
				BBox: image.Rect(
					int(float64(face.Col)*invRatio),
					int(float64(face.Row)*invRatio),
					int(float64(face.Col+face.Scale)*invRatio),
					int(float64(face.Row+face.Scale)*invRatio),
				),
				Confidence: float64(face.Q),
			})
		}
	}

	dets = ft.trackMgr.match(dets)
	ft.lastDetections = dets
	return dets
}

func (ft *FaceTracker) LastDetections() []Detection { return ft.lastDetections }
func (ft *FaceTracker) Close() error                { return nil }

func toGrayImage(img image.Image) *image.Gray {
	bounds := img.Bounds()
	gray := image.NewGray(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			gray.Set(x, y, img.At(x, y))
		}
	}
	return gray
}

func frameChanged(prev, curr *image.Gray) bool {
	bounds := prev.Bounds()
	changed := 0
	total := 0
	step := 4
	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			diff := int(curr.GrayAt(x, y).Y) - int(prev.GrayAt(x, y).Y)
			if diff < 0 {
				diff = -diff
			}
			if diff > 20 {
				changed++
			}
			total++
		}
	}
	if total == 0 {
		return false
	}
	return float64(changed)/float64(total) > 0.01
}

func resizeRGBA(img image.Image, w, h int) *image.RGBA {
	bounds := img.Bounds()
	resized := image.NewRGBA(image.Rect(0, 0, w, h))
	rx := float64(bounds.Dx()) / float64(w)
	ry := float64(bounds.Dy()) / float64(h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			srcX := int(float64(x) * rx)
			srcY := int(float64(y) * ry)
			r, g, b, a := img.At(srcX+bounds.Min.X, srcY+bounds.Min.Y).RGBA()
			resized.SetRGBA(x, y, color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)})
		}
	}
	return resized
}
