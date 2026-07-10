package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Engine   EngineConfig   `yaml:"engine"`
	Output   OutputConfig   `yaml:"output"`
	Database DatabaseConfig `yaml:"database"`
	Streams  []StreamConfig `yaml:"streams"`
}

type EngineConfig struct {
	MaxConcurrentStarts int           `yaml:"max_concurrent_starts"`
	ShutdownTimeout     time.Duration `yaml:"shutdown_timeout"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	HealthCheckTimeout  time.Duration `yaml:"health_check_timeout"`
}

type OutputConfig struct {
	Backend    string           `yaml:"backend"`
	Kafka      KafkaConfig      `yaml:"kafka"`
	Serializer SerializerConfig `yaml:"serializer"`
}

type KafkaConfig struct {
	Brokers         []string `yaml:"brokers"`
	TopicPrefix     string   `yaml:"topic_prefix"`
	MaxMessageBytes int      `yaml:"max_message_bytes"`
	RequiredAcks    int16    `yaml:"required_acks"`
	Compression     string   `yaml:"compression"`
}

type SerializerConfig struct {
	Format  string `yaml:"format"`
	Quality int    `yaml:"quality"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

type StreamConfig struct {
	ID          string        `yaml:"id"`
	RTSPURL     string        `yaml:"rtsp_url"`
	OutputTopic string        `yaml:"output_topic"`
	CaptureFPS  int           `yaml:"capture_fps"`
	DecodeScale string        `yaml:"decode_scale"`
	Group       string        `yaml:"group"`
	Restart     RestartPolicy `yaml:"restart"`
	Filters     []FilterSpec  `yaml:"filters"`

	// Phase 2 reserved (ignored in Phase 1)
	PipeFormat  string `yaml:"pipe_format"`
	JPEGQuality int    `yaml:"jpeg_quality"`
	HWAccel     string `yaml:"hwaccel"`
}

type RestartPolicy struct {
	MaxRetries     int           `yaml:"max_retries"`
	BackoffInitial time.Duration `yaml:"backoff_initial"`
	BackoffMax     time.Duration `yaml:"backoff_max"`
	BackoffFactor  float64       `yaml:"backoff_factor"`
}

type FilterSpec struct {
	Type   string                 `yaml:"type"`
	Params map[string]interface{} `yaml:"params"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := &Config{
		Engine: EngineConfig{
			MaxConcurrentStarts: 5,
			ShutdownTimeout:     30 * time.Second,
			HealthCheckInterval: 10 * time.Second,
			HealthCheckTimeout:  30 * time.Second,
		},
		Output: OutputConfig{
			Serializer: SerializerConfig{
				Format:  "jpeg",
				Quality: 85,
			},
		},
		Database: DatabaseConfig{
			Port:    5432,
			SSLMode: "disable",
		},
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Output.Backend == "" {
		return fmt.Errorf("output.backend is required")
	}
	if len(c.Output.Kafka.Brokers) == 0 {
		return fmt.Errorf("output.kafka.brokers is required")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("database.host is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database.user is required")
	}
	if c.Database.DBName == "" {
		return fmt.Errorf("database.dbname is required")
	}
	for i, s := range c.Streams {
		if s.ID == "" {
			return fmt.Errorf("streams[%d].id is required", i)
		}
		if s.RTSPURL == "" {
			return fmt.Errorf("streams[%d].rtsp_url is required", i)
		}
	}
	return nil
}
