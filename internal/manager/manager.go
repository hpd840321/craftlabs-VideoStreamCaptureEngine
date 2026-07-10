package manager

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/craftlabs/video-stream-capture-engine/internal/config"
	"github.com/craftlabs/video-stream-capture-engine/internal/decoder"
	"github.com/craftlabs/video-stream-capture-engine/internal/filter"
	"github.com/craftlabs/video-stream-capture-engine/internal/output"
)

type StreamManager struct {
	cfg         *config.Config
	health      *HealthMonitor
	mu          sync.Mutex
	workers     map[string]*decoder.DecoderWorker
	kafkaWriter output.OutputWriter
}

func NewStreamManager(cfg *config.Config) (*StreamManager, error) {
	kafkaWriter, err := output.NewKafkaWriter(
		cfg.Output.Kafka.Brokers,
		cfg.Output.Kafka.TopicPrefix,
		cfg.Output.Kafka.Compression,
		cfg.Output.Kafka.RequiredAcks,
	)
	if err != nil {
		return nil, fmt.Errorf("create kafka writer: %w", err)
	}

	return &StreamManager{
		cfg:         cfg,
		health:      NewHealthMonitor(cfg.Engine.HealthCheckTimeout),
		workers:     make(map[string]*decoder.DecoderWorker),
		kafkaWriter: kafkaWriter,
	}, nil
}

func (m *StreamManager) Start(ctx context.Context) error {
	sem := make(chan struct{}, m.cfg.Engine.MaxConcurrentStarts)

	for _, streamCfg := range m.cfg.Streams {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sem <- struct{}{}:
		}

		go func(sc config.StreamConfig) {
			defer func() { <-sem }()

			if err := m.startStream(sc); err != nil {
				slog.Error("failed to start stream", "stream", sc.ID, "error", err)
			}
		}(streamCfg)
	}

	return nil
}

func (m *StreamManager) startStream(cfg config.StreamConfig) error {
	width, height := parseScale(cfg.DecodeScale)
	reader := decoder.NewRawVideoReader(width, height)
	pipeline := m.buildPipeline(cfg)
	serializer := output.NewFrameSerializer(m.cfg.Output.Serializer.Quality)

	topic := output.ResolveTopic(m.cfg.Output.Kafka.TopicPrefix, cfg.ID, cfg.OutputTopic)
	var streamOutput output.OutputWriter

	if topic != m.cfg.Output.Kafka.TopicPrefix && cfg.OutputTopic != "" {
		var err error
		streamOutput, err = output.NewKafkaWriter(
			m.cfg.Output.Kafka.Brokers,
			"",
			m.cfg.Output.Kafka.Compression,
			m.cfg.Output.Kafka.RequiredAcks,
		)
		if err != nil {
			return fmt.Errorf("create per-stream kafka writer: %w", err)
		}
		if kw, ok := streamOutput.(*output.KafkaWriter); ok {
			kw.SetTopic(topic)
		}
	} else {
		streamOutput = m.kafkaWriter
	}

	worker := decoder.NewDecoderWorker(
		cfg.ID, cfg.RTSPURL, reader, pipeline, serializer, streamOutput,
	)

	m.mu.Lock()
	m.workers[cfg.ID] = worker
	m.mu.Unlock()

	m.health.Register(cfg.ID)

	ffmpegCfg := decoder.FFmpegConfig{
		RTSPURL:     cfg.RTSPURL,
		CaptureFPS:  cfg.CaptureFPS,
		DecodeScale: cfg.DecodeScale,
	}

	backoff := decoder.NewExponentialBackoff(decoder.ExponentialBackoffConfig{
		Initial: cfg.Restart.BackoffInitial,
		Max:     cfg.Restart.BackoffMax,
		Factor:  cfg.Restart.BackoffFactor,
	})

	slog.Info("starting stream worker", "stream", cfg.ID)
	return worker.Run(ffmpegCfg, backoff)
}

func (m *StreamManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, worker := range m.workers {
		slog.Info("stopping stream worker", "stream", id)
		worker.Stop()
		m.health.Unregister(id)
	}

	if m.kafkaWriter != nil {
		m.kafkaWriter.Close()
	}
}

func (m *StreamManager) buildPipeline(cfg config.StreamConfig) *filter.FilterPipeline {
	var filters []filter.FrameFilter

	for _, fs := range cfg.Filters {
		switch fs.Type {
		case "duplicate":
			threshold := 10
			if t, ok := fs.Params["threshold"]; ok {
				if tInt, ok := t.(int); ok {
					threshold = tInt
				}
			}
			filters = append(filters, filter.NewDuplicateFilter(threshold))
		case "noop":
			filters = append(filters, &filter.NoopFilter{})
		default:
			slog.Warn("unknown filter type, skipping", "type", fs.Type)
		}
	}

	if len(filters) == 0 {
		filters = append(filters, &filter.NoopFilter{})
	}

	return filter.NewFilterPipeline(filters)
}

func parseScale(scale string) (int, int) {
	if scale == "" {
		return 1920, 1080
	}
	var w, h int
	_, err := fmt.Sscanf(scale, "%dx%d", &w, &h)
	if err != nil {
		slog.Warn("invalid decode_scale, using 1920x1080", "scale", scale)
		return 1920, 1080
	}
	return w, h
}
