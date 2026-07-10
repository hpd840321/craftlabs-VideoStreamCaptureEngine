package decoder

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"log/slog"
	"os/exec"
	"time"

	"github.com/craftlabs/video-stream-capture-engine/internal/filter"
	"github.com/craftlabs/video-stream-capture-engine/internal/output"
)

type DecoderWorker struct {
	streamID   string
	rtspURL    string
	reader     FrameReader
	pipeline   *filter.FilterPipeline
	serializer *output.FrameSerializer
	output     output.OutputWriter
	ctx        context.Context
	cancel     context.CancelFunc
	seqNum     int64
}

func NewDecoderWorker(
	streamID, rtspURL string,
	reader FrameReader,
	pipeline *filter.FilterPipeline,
	serializer *output.FrameSerializer,
	output output.OutputWriter,
) *DecoderWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &DecoderWorker{
		streamID:   streamID,
		rtspURL:    rtspURL,
		reader:     reader,
		pipeline:   pipeline,
		serializer: serializer,
		output:     output,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (w *DecoderWorker) Stop() {
	w.cancel()
}

func (w *DecoderWorker) Run(ffmpegCfg FFmpegConfig, backoff *ExponentialBackoff) error {
	defer w.cancel()

	for {
		select {
		case <-w.ctx.Done():
			return nil
		default:
		}

		args := BuildFFmpegArgs(ffmpegCfg)
		cmd := exec.CommandContext(w.ctx, args[0], args[1:]...)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("create stdout pipe: %w", err)
		}

		if err := cmd.Start(); err != nil {
			slog.Error("ffmpeg start failed", "stream", w.streamID, "error", err)
			backoff.Sleep()
			continue
		}

		slog.Info("ffmpeg started", "stream", w.streamID, "pid", cmd.Process.Pid)
		backoff.Reset()

		if err := w.readLoop(stdout); err != nil {
			slog.Warn("frame read loop exited", "stream", w.streamID, "error", err)
		}

		if waitErr := cmd.Wait(); waitErr != nil {
			slog.Warn("ffmpeg exited", "stream", w.streamID, "error", waitErr)
		}
	}
}

func (w *DecoderWorker) readLoop(stdout interface{ Read([]byte) (int, error) }) error {
	for {
		select {
		case <-w.ctx.Done():
			return nil
		default:
		}

		img, err := w.reader.ReadFrame(stdout)
		if err != nil {
			return fmt.Errorf("read frame: %w", err)
		}

		if err := w.processFrame(img, time.Now()); err != nil {
			slog.Error("process frame failed", "stream", w.streamID, "error", err)
		}
	}
}

func (w *DecoderWorker) processFrame(img image.Image, ts time.Time) error {
	w.seqNum++
	meta := filter.FrameMeta{
		StreamID:  w.streamID,
		SeqNum:    w.seqNum,
		Timestamp: ts,
	}

	result, ok := w.pipeline.Process(img, meta)
	if !ok {
		return nil
	}

	outFrame, err := w.serializer.Serialize(&output.Frame{
		Image: result.Image,
		Meta: output.FrameMeta{
			StreamID:  result.Meta.StreamID,
			SeqNum:    result.Meta.SeqNum,
			Timestamp: result.Meta.Timestamp,
		},
		Score: result.Score,
	})
	if err != nil {
		return fmt.Errorf("serialize frame: %w", err)
	}

	// Enrich metadata with tracker detections
	if dets := w.pipeline.Detections(); len(dets) > 0 {
		if outFrame.Metadata == nil {
			outFrame.Metadata = make(map[string]string)
		}
		data, _ := json.Marshal(dets)
		outFrame.Metadata["detections"] = string(data)
		outFrame.Metadata["detection_count"] = fmt.Sprintf("%d", len(dets))
	}

	return w.output.Write(outFrame)
}
