package filter

import "image"

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
