package output

import (
	"bytes"
	"image/jpeg"
)

type FrameSerializer struct {
	quality int
}

func NewFrameSerializer(quality int) *FrameSerializer {
	if quality < 1 {
		quality = 1
	}
	if quality > 100 {
		quality = 100
	}
	return &FrameSerializer{quality: quality}
}

func (s *FrameSerializer) Serialize(frame *Frame) (*OutputFrame, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, frame.Image, &jpeg.Options{Quality: s.quality}); err != nil {
		return nil, err
	}

	data := buf.Bytes()
	return &OutputFrame{
		StreamID:  frame.Meta.StreamID,
		SeqNum:    frame.Meta.SeqNum,
		Timestamp: frame.Meta.Timestamp,
		Quality:   frame.Score,
		ImageData: data,
		ImageSize: len(data),
	}, nil
}
