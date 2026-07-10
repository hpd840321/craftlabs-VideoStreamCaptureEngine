package filter

import (
	"image"

	"github.com/craftlabs/video-stream-capture-engine/internal/tracker"
)

type FilterPipeline struct {
	filters []FrameFilter
}

func NewFilterPipeline(filters []FrameFilter) *FilterPipeline {
	return &FilterPipeline{filters: filters}
}

func (p *FilterPipeline) Process(frame image.Image, meta FrameMeta) (*FilteredFrame, bool) {
	current := FilteredFrame{Image: frame, Meta: meta}

	for _, f := range p.filters {
		result, decision := f.Apply(current.Image, current.Meta)

		switch decision {
		case FilterPass:
			current = result
		case FilterDrop, FilterAbort:
			return nil, false
		}
	}

	return &current, true
}

func (p *FilterPipeline) Detections() []tracker.Detection {
	for _, f := range p.filters {
		if tf, ok := f.(*TrackerFilter); ok {
			dets := tf.Detections()
			if len(dets) > 0 {
				return dets
			}
		}
	}
	return nil
}
