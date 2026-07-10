package decoder

import (
	"fmt"
	"image"
	"io"
)

// FrameReader reads a single decoded video frame from a byte stream.
type FrameReader interface {
	ReadFrame(r io.Reader) (image.Image, error)
}

// RawVideoReader reads raw BGR pixel data of known dimensions from a pipe.
// Each frame is exactly width * height * 3 bytes.
type RawVideoReader struct {
	width     int
	height    int
	frameSize int
}

func NewRawVideoReader(width, height int) *RawVideoReader {
	return &RawVideoReader{
		width:     width,
		height:    height,
		frameSize: width * height * 3,
	}
}

func (r *RawVideoReader) ReadFrame(rd io.Reader) (image.Image, error) {
	buf := make([]byte, r.frameSize)
	if _, err := io.ReadFull(rd, buf); err != nil {
		return nil, fmt.Errorf("read raw frame: %w", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, r.width, r.height))
	for y := 0; y < r.height; y++ {
		for x := 0; x < r.width; x++ {
			offset := (y*r.width + x) * 3
			b := buf[offset]
			g := buf[offset+1]
			r_ := buf[offset+2]
			img.SetRGBA(x, y, image.RGBA{R: r_, G: g, B: b, A: 255})
		}
	}

	return img, nil
}
