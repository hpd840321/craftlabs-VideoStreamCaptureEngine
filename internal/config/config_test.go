package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig_MinimalYAML(t *testing.T) {
	yamlContent := `
engine:
  max_concurrent_starts: 3
  shutdown_timeout: 15s
output:
  backend: kafka
  kafka:
    brokers:
      - "localhost:9092"
    topic_prefix: "test-frames"
  serializer:
    format: jpeg
    quality: 80
database:
  host: "pg.example.com"
  user: "admin"
  password: "secret"
  dbname: "testdb"
streams:
  - id: "cam-test"
    rtsp_url: "rtsp://10.0.0.1/stream"
    capture_fps: 25
    decode_scale: "1920x1080"
    output_topic: "cam-test-out"
    filters:
      - type: "noop"
`
	tmpFile := writeTempYAML(t, yamlContent)
	defer os.Remove(tmpFile)

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Engine.MaxConcurrentStarts != 3 {
		t.Errorf("MaxConcurrentStarts = %d, want 3", cfg.Engine.MaxConcurrentStarts)
	}
	if cfg.Engine.ShutdownTimeout != 15*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 15s", cfg.Engine.ShutdownTimeout)
	}
	if cfg.Output.Backend != "kafka" {
		t.Errorf("Backend = %s, want kafka", cfg.Output.Backend)
	}
	if len(cfg.Streams) != 1 {
		t.Fatalf("len(Streams) = %d, want 1", len(cfg.Streams))
	}
	if cfg.Database.Host != "pg.example.com" {
		t.Errorf("Database.Host = %s, want pg.example.com", cfg.Database.Host)
	}
	if cfg.Database.DSN() == "" {
		t.Error("DSN() should not be empty")
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	yamlContent := `
output:
  backend: kafka
  kafka:
    brokers:
      - "localhost:9092"
database:
  host: "pg.example.com"
  user: "admin"
  password: "secret"
  dbname: "testdb"
streams: []
`
	tmpFile := writeTempYAML(t, yamlContent)
	defer os.Remove(tmpFile)

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Engine.MaxConcurrentStarts != 5 {
		t.Errorf("default MaxConcurrentStarts = %d, want 5", cfg.Engine.MaxConcurrentStarts)
	}
	if cfg.Output.Serializer.Format != "jpeg" {
		t.Errorf("default Format = %s, want jpeg", cfg.Output.Serializer.Format)
	}
}

func TestLoadConfig_Validation_MissingDatabase(t *testing.T) {
	yamlContent := `
output:
  backend: kafka
  kafka:
    brokers:
      - "localhost:9092"
streams: []
`
	tmpFile := writeTempYAML(t, yamlContent)
	defer os.Remove(tmpFile)

	_, err := Load(tmpFile)
	if err == nil {
		t.Fatal("expected error for missing database config")
	}
}

func TestLoadConfig_Validation_MissingStreamID(t *testing.T) {
	yamlContent := `
output:
  backend: kafka
  kafka:
    brokers:
      - "localhost:9092"
database:
  host: "pg.example.com"
  user: "admin"
  password: "secret"
  dbname: "testdb"
streams:
  - rtsp_url: "rtsp://x"
`
	tmpFile := writeTempYAML(t, yamlContent)
	defer os.Remove(tmpFile)

	_, err := Load(tmpFile)
	if err == nil {
		t.Fatal("expected error for missing stream id")
	}
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpFile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	return tmpFile.Name()
}
