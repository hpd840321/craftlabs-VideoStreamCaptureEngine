package store

import (
	"context"
	"fmt"
)

type User struct {
	ID           int    `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"created_at"`
}

func (db *DB) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var u User
	err := db.Pool.QueryRow(ctx,
		"SELECT id, username, password_hash, created_at FROM users WHERE username=$1",
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return &u, nil
}

func (db *DB) CreateUser(ctx context.Context, username, passwordHash string) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO users (username, password_hash) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		username, passwordHash,
	)
	return err
}

func (db *DB) UpdatePassword(ctx context.Context, username, newHash string) error {
	_, err := db.Pool.Exec(ctx,
		"UPDATE users SET password_hash=$1 WHERE username=$2",
		newHash, username,
	)
	return err
}
