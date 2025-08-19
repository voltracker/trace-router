package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
)

type HopsAgg struct {
	Source_ip   string
	Dest_ip     string
	Count       int
	Avg_latency float32
}

type DBErrors string

const (
	HopInsertionError DBErrors = "failed to insert hop"
	ReadAggsError     DBErrors = "failed to read aggregated statistics"
)

func Connect(connString string) (*pgxpool.Pool, error) {
	slog.Info("Starting DB connection pool")
	pool, err := pgxpool.New(context.Background(), connString)
	return pool, err
}

func GetAggs(conn *pgxpool.Pool) ([]HopsAgg, error) {
	rows, err := conn.Query(context.Background(), "select * from hops_agg;")
	if err != nil {
		slog.Error("failed to read aggregated statistics from DB", "error", err)
		return nil, errors.New(string(ReadAggsError))
	}

	aggs, err := pgx.CollectRows(rows, pgx.RowToStructByName[HopsAgg])
	if err != nil {
		slog.Error("failed to collect rows", "error", err)
		return nil, errors.New(string(ReadAggsError))
	}
	return aggs, nil
}

func AddHop(hop Hop, conn *pgxpool.Pool) error {
	_, err := conn.Exec(context.Background(), "insert into hops values ($1, $2, $3, $4)", hop.id, hop.src, hop.dest, hop.latency)
	if err != nil {
		slog.Error("failed to insert new hop", "error", err)
		return errors.New(string(HopInsertionError))
	}
	return nil
}
