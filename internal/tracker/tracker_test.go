package tracker

import (
	"image"
	"testing"
)

func TestObjectTracker_Track(t *testing.T) {
	ot := &ObjectTracker{}
	dets := ot.Track(image.NewRGBA(image.Rect(0, 0, 10, 10)))
	if len(dets) != 0 {
		t.Errorf("Track() returned %d detections, want 0", len(dets))
	}
}

func TestObjectTracker_Close(t *testing.T) {
	ot := &ObjectTracker{}
	if err := ot.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
