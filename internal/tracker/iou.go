package tracker

import "image"

func iou(a, b image.Rectangle) float64 {
	intersection := a.Intersect(b)
	if intersection.Empty() {
		return 0.0
	}
	interArea := area(intersection)
	unionArea := area(a) + area(b) - interArea
	if unionArea == 0 {
		return 0.0
	}
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
	used := make(map[int]bool)

	for di := range detections {
		bestIOU := tm.iouThreshold
		bestTrack := -1
		for id, track := range tm.tracks {
			if used[id] {
				continue
			}
			i := iou(detections[di].BBox, track.lastBBox)
			if i > bestIOU {
				bestIOU = i
				bestTrack = id
			}
		}
		if bestTrack >= 0 {
			detections[di].TrackID = bestTrack
			tm.tracks[bestTrack].lastBBox = detections[di].BBox
			tm.tracks[bestTrack].lostFrames = 0
			used[bestTrack] = true
		} else {
			detections[di].TrackID = tm.nextID
			tm.tracks[tm.nextID] = &trackState{id: tm.nextID, lastBBox: detections[di].BBox}
			tm.nextID++
		}
	}

	// Increment lost frames for unmatched tracks, cleanup stale
	for id, track := range tm.tracks {
		if !used[id] {
			track.lostFrames++
			if track.lostFrames > tm.maxLost {
				delete(tm.tracks, id)
			}
		}
	}

	return detections
}
