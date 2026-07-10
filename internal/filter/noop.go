package filter

import "image"

type NoopFilter struct{}

func (n *NoopFilter) Name() string { return "noop" }

func (n *NoopFilter) Apply(frame image.Image, meta FrameMeta) (FilteredFrame, FilterDecision) {
	return FilteredFrame{Image: frame, Meta: meta, Score: 0}, FilterPass
}
