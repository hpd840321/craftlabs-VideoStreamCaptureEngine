package store

import (
	"context"
	"time"
)

type Event struct {
	ID           int       `json:"id"`
	StreamID     string    `json:"stream_id"`
	Level        string    `json:"level"`
	Message      string    `json:"message"`
	Acknowledged bool      `json:"acknowledged"`
	CreatedAt    time.Time `json:"created_at"`
}

type EventFilter struct {
	Level    string
	StreamID string
	Page     int
	Size     int
}

func (db *DB) InsertEvent(ctx context.Context, streamID, level, message string) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO events (stream_id, level, message) VALUES ($1, $2, $3)",
		streamID, level, message,
	)
	return err
}

func (db *DB) ListEvents(ctx context.Context, f EventFilter) ([]Event, int, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.Size <= 0 {
		f.Size = 20
	}

	where := "WHERE 1=1"
	args := make([]interface{}, 0)
	argIdx := 1

	if f.Level != "" {
		where += " AND level = $" + itoa(argIdx)
		args = append(args, f.Level)
		argIdx++
	}
	if f.StreamID != "" {
		where += " AND stream_id = $" + itoa(argIdx)
		args = append(args, f.StreamID)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM events " + where
	if err := db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (f.Page - 1) * f.Size
	query := "SELECT id, stream_id, level, message, acknowledged, created_at FROM events " + where +
		" ORDER BY created_at DESC LIMIT $" + itoa(argIdx) + " OFFSET $" + itoa(argIdx+1)
	args = append(args, f.Size, offset)

	rows, err := db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.StreamID, &e.Level, &e.Message, &e.Acknowledged, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	return events, total, nil
}

func (db *DB) AckEvents(ctx context.Context, ids []int) error {
	for _, id := range ids {
		if _, err := db.Pool.Exec(ctx, "UPDATE events SET acknowledged=true WHERE id=$1", id); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) AckAllEvents(ctx context.Context) error {
	_, err := db.Pool.Exec(ctx, "UPDATE events SET acknowledged=true WHERE acknowledged=false")
	return err
}

func itoa(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
