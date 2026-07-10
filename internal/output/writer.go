package output

import (
	"image"
	"time"
)

type FrameMeta struct {
	StreamID  string
	SeqNum    int64
	Timestamp time.Time
}

type Frame struct {
	Image image.Image
	Meta  FrameMeta
	Score float64
}

type OutputFrame struct {
	StreamID  string            `json:"stream_id"`
	SeqNum    int64             `json:"seq_num"`
	Timestamp time.Time         `json:"timestamp"`
	Quality   float64           `json:"quality"`
	ImageData []byte            `json:"image_data"`
	ImageSize int               `json:"image_size"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type OutputWriter interface {
	Write(frame *OutputFrame) error
	Close() error
	Backend() string
}
