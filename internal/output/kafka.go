package output

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/IBM/sarama"
)

type KafkaWriter struct {
	producer sarama.SyncProducer
	topic    string
	backend  string
}

func NewKafkaWriter(brokers []string, topic string, compression string, requiredAcks int16) (*KafkaWriter, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.RequiredAcks(requiredAcks)
	config.Producer.Return.Successes = true

	switch compression {
	case "snappy":
		config.Producer.Compression = sarama.CompressionSnappy
	case "gzip":
		config.Producer.Compression = sarama.CompressionGZIP
	case "lz4":
		config.Producer.Compression = sarama.CompressionLZ4
	default:
		config.Producer.Compression = sarama.CompressionNone
	}

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("create kafka producer: %w", err)
	}

	return &KafkaWriter{
		producer: producer,
		topic:    topic,
		backend:  "kafka",
	}, nil
}

func (w *KafkaWriter) Write(frame *OutputFrame) error {
	data, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal frame: %w", err)
	}

	_, _, err = w.producer.SendMessage(&sarama.ProducerMessage{
		Topic: w.topic,
		Key:   sarama.StringEncoder(frame.StreamID),
		Value: sarama.ByteEncoder(data),
	})
	if err != nil {
		slog.Error("kafka write failed", "stream", frame.StreamID, "error", err)
		return fmt.Errorf("kafka send: %w", err)
	}

	return nil
}

func (w *KafkaWriter) Close() error {
	return w.producer.Close()
}

func (w *KafkaWriter) Backend() string {
	return w.backend
}

func (w *KafkaWriter) SetTopic(topic string) {
	w.topic = topic
}

func ResolveTopic(prefix, streamID, outputTopic string) string {
	if outputTopic != "" {
		return outputTopic
	}
	if prefix != "" {
		return prefix + "." + streamID
	}
	return streamID
}
