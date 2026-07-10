package tracker

import "image"

type ObjectTracker struct{}

func (o *ObjectTracker) Track(frame image.Image) []Detection { return nil }
func (o *ObjectTracker) Close() error                        { return nil }
