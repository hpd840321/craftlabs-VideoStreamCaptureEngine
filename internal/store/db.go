package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

func NewDB(ctx context.Context, dsn string) (*DB, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	config.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}

func (db *DB) Migrate(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(64) UNIQUE NOT NULL,
			password_hash VARCHAR(256) NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			id SERIAL PRIMARY KEY,
			stream_id VARCHAR(64) NOT NULL DEFAULT '',
			level VARCHAR(16) NOT NULL DEFAULT 'info',
			message TEXT NOT NULL DEFAULT '',
			acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_stream ON events(stream_id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_level ON events(level)`,
		`CREATE INDEX IF NOT EXISTS idx_events_created ON events(created_at DESC)`,
	}
	for _, q := range queries {
		if _, err := db.Pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("migration: %w", err)
		}
	}
	return nil
}
