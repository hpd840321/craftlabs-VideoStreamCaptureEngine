package api

import (
	"context"

	"github.com/craftlabs/video-stream-capture-engine/internal/store"
)

type Store interface {
	GetUserByUsername(ctx context.Context, username string) (*store.User, error)
	CreateUser(ctx context.Context, username, passwordHash string) error
	UpdatePassword(ctx context.Context, username, newHash string) error
	InsertEvent(ctx context.Context, streamID, level, message string) error
	ListEvents(ctx context.Context, f store.EventFilter) ([]store.Event, int, error)
	AckEvents(ctx context.Context, ids []int) error
	AckAllEvents(ctx context.Context) error
	Migrate(ctx context.Context) error
	Close()
}

var _ Store = (*storeDBWrapper)(nil)

type storeDBWrapper struct {
	*store.DB
}

func newStoreDBWrapper(db *store.DB) Store {
	return &storeDBWrapper{DB: db}
}
