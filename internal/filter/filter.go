package filter

import (
	"image"
	"time"
)

type FilterDecision int

const (
	FilterPass  FilterDecision = iota
	FilterDrop
	FilterAbort
)

type FrameMeta struct {
	StreamID  string
	SeqNum    int64
	Timestamp time.Time
	ImageSize int
}

type FilteredFrame struct {
	Image image.Image
	Meta  FrameMeta
	Score float64
}

type FrameFilter interface {
	Name() string
	Apply(frame image.Image, meta FrameMeta) (FilteredFrame, FilterDecision)
}
