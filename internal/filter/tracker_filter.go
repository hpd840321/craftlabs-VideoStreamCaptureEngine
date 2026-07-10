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
	return FilteredFrame{Image: frame, Meta: meta, Score: 1.0}, FilterPass
}

func (t *TrackerFilter) Close() error { return t.tracker.Close() }

func (t *TrackerFilter) Detections() []tracker.Detection {
	if ft, ok := t.tracker.(*tracker.FaceTracker); ok {
		return ft.LastDetections()
	}
	return nil
}
