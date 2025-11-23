package internal

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HopsAgg struct {
	Source_ip   string
	Dest_ip     string
	Count       int
	Avg_latency float32
}

type Hop struct {
	Id      uuid.UUID
	Src     string
	Dest    string
	Latency float64
}

type TraceResult struct {
	InitialSrc string
	FinalDst   string
	Hops       []Hop
}

type Geoip struct {
	Id  uuid.UUID
	Ip  string
	Lat float32
	Lon float32
	Isp string
	Org string
	Asn string
}

type DBErrors string

const (
	HopInsertionError DBErrors = "failed to insert hop"
	IpInsertionError  DBErrors = "failed to insert geoip"
	ReadAggsError     DBErrors = "failed to read aggregated statistics"
	IpNotPresent      DBErrors = "ip not present in db"
	IpLookupError     DBErrors = "ip lookup error"
)

func (e DBErrors) Error() string {
	return string(e)
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
		return nil, ReadAggsError
	}

	aggs, err := pgx.CollectRows(rows, pgx.RowToStructByName[HopsAgg])
	if err != nil {
		slog.Error("failed to collect rows", "error", err)
		return nil, ReadAggsError
	}
	return aggs, nil
}

func getIp(ip string, conn *pgxpool.Pool) (Geoip, error) {
	rows, err := conn.Query(context.Background(), "select * from geoip where ip_addr = $1", ip)
	if err != nil {
		slog.Error("failed to collect ip row", "error", err)
		// todo: return appropriate error
		return Geoip{}, err
	}
	geoip, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Geoip])
	if err != nil {
		if errors.Is(pgx.ErrNoRows, err) {
			slog.Info("ip address ")
			return Geoip{}, IpNotPresent
		} else {
			return Geoip{}, IpLookupError
		}
	}
	return geoip, nil
}

func AddIp(ip Geoip, conn *pgxpool.Pool) error {
	_, err := conn.Exec(context.Background(), "insert into geoip values ($1, $2, $3, $4, $5, $6, $7)", ip.Id, ip.Ip, ip.Lat, ip.Lon, ip.Isp, ip.Org, ip.Asn)
	if err != nil {
		slog.Error("failed to insert ip", "error", err)
		return IpInsertionError
	}
	return nil
}

func AddHop(hop Hop, conn *pgxpool.Pool) error {
	_, err := conn.Exec(context.Background(), "insert into hops values ($1, $2, $3, $4)", hop.Id, hop.Src, hop.Dest, hop.Latency)
	if err != nil {
		slog.Error("failed to insert new hop", "error", err)
		return HopInsertionError
	}
	return nil
}
