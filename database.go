package main

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
)

func connect(connString string) (*pgxpool.Pool, error) {
	slog.Info("Starting DB connection pool")
	pool, err := pgxpool.New(context.Background(), connString)
	return pool, err
}
