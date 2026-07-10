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
