package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
)

type HopsAgg struct {
	source      string
	destination string
	count       int
	avg_latency float32
}

func Connect(connString string) (*pgxpool.Pool, error) {
	slog.Info("Starting DB connection pool")
	pool, err := pgxpool.New(context.Background(), connString)
	return pool, err
}

func GetAggs(conn *pgxpool.Pool) ([]HopsAgg, error) {
	rows, err := conn.Query(context.Background(), "select * from hops_agg;")
	if err != nil {
		slog.Error("failed to read aggregated statistics from DB", "error", err)
		return nil, errors.New("couldn't read from database")
	}

	aggs, err := pgx.CollectRows(rows, pgx.RowToStructByName[HopsAgg])
	if err != nil {
		slog.Error("failed to collect rows", "error", err)
		return nil, errors.New("couldn't read rows")
	}
	return aggs, nil
}

func AddHop(hop Hop, conn *pgxpool.Pool) error {
	_, err := conn.Exec(context.Background(), "insert into hop values ($1, $2, $3, $4)", hop.src, hop.dest, hop.latency)
	if err != nil {
		slog.Error("failed to insert new hop")
	}
	return nil
}
