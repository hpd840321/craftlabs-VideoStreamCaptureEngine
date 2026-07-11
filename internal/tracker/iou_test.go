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

func TestMatchDetections_TrackCleanup(t *testing.T) {
	tm := newTrackManager(0.3, 2)
	dets := []Detection{{Class: "face", BBox: image.Rect(10, 10, 50, 50), Confidence: 0.9}}
	matched := tm.match(dets)
	if matched[0].TrackID != 1 {
		t.Errorf("first ID = %d, want 1", matched[0].TrackID)
	}

	// Two frames with no detections → track should be cleaned
	tm.match(nil)
	tm.match(nil)

	// New detection should get new track ID
	dets2 := []Detection{{Class: "face", BBox: image.Rect(100, 100, 150, 150), Confidence: 0.9}}
	matched2 := tm.match(dets2)
	if matched2[0].TrackID != 2 {
		t.Errorf("after cleanup, new track ID = %d, want 2", matched2[0].TrackID)
	}
}

func TestIOU_Contained(t *testing.T) {
	a := image.Rect(0, 0, 200, 200)
	b := image.Rect(50, 80, 150, 160)
	i := iou(a, b)
	// b is contained within a: intersection = area(b), union = area(a)
	expected := float64(area(b)) / float64(area(a))
	if i != expected {
		t.Errorf("IOU contained = %f, want %f", i, expected)
	}
}

func TestIOU_ZeroArea(t *testing.T) {
	a := image.Rect(0, 0, 0, 0)
	b := image.Rect(0, 0, 100, 100)
	if i := iou(a, b); i != 0.0 {
		t.Errorf("IOU zero area = %f, want 0.0", i)
	}
}

func TestMatchDetections_MultipleTracks(t *testing.T) {
	tm := newTrackManager(0.3, 5)

	// Add two separate tracks
	dets1 := []Detection{
		{Class: "face", BBox: image.Rect(10, 10, 50, 50), Confidence: 0.9},
		{Class: "face", BBox: image.Rect(100, 100, 140, 140), Confidence: 0.8},
	}
	matched1 := tm.match(dets1)
	if len(matched1) != 2 {
		t.Fatalf("expected 2 detections, got %d", len(matched1))
	}
	if matched1[0].TrackID == matched1[1].TrackID {
		t.Error("two separate faces should have different track IDs")
	}

	// Same positions (slightly moved) should keep same track IDs
	dets2 := []Detection{
		{Class: "face", BBox: image.Rect(12, 12, 52, 52), Confidence: 0.9},
		{Class: "face", BBox: image.Rect(102, 102, 142, 142), Confidence: 0.8},
	}
	matched2 := tm.match(dets2)
	if len(matched2) != 2 {
		t.Fatalf("expected 2 detections, got %d", len(matched2))
	}
	if matched2[0].TrackID != matched1[0].TrackID {
		t.Errorf("track 0 should keep ID %d, got %d", matched1[0].TrackID, matched2[0].TrackID)
	}
	if matched2[1].TrackID != matched1[1].TrackID {
		t.Errorf("track 1 should keep ID %d, got %d", matched1[1].TrackID, matched2[1].TrackID)
	}
}

func TestMatchDetections_TrackLossAndCleanup(t *testing.T) {
	tm := newTrackManager(0.3, 3)

	// Create a track
	dets := []Detection{{Class: "face", BBox: image.Rect(10, 10, 50, 50), Confidence: 0.9}}
	tm.match(dets)

	// Lose it for maxLost frames
	for i := 0; i < 4; i++ {
		tm.match(nil) // no detections — track loses frames
	}

	// New detection should get a new track ID
	dets2 := []Detection{{Class: "face", BBox: image.Rect(10, 10, 50, 50), Confidence: 0.9}}
	matched := tm.match(dets2)
	if matched[0].TrackID == 1 {
		t.Error("stale track should have been cleaned up, new detection should get new track ID")
	}
}
