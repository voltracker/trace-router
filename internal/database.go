package internal

import (
	"context"
	"errors"
	"log/slog"

	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/pgtype"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HopsAgg struct {
	Source_ip   string
	Dest_ip     string
	Count       int
	Avg_latency float32
}

type tmpHopsAgg struct {
	Source_ip   pgtype.Inet
	Dest_ip     pgtype.Inet
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

type tmpGeoip struct {
	Id  uuid.UUID   `db:"id"`
	Ip  pgtype.Inet `db:"ip_addr"`
	Lat float32     `db:"lat"`
	Lon float32     `db:"lon"`
	Isp string      `db:"isp"`
	Org string      `db:"org"`
	Asn string      `db:"asn"`
}

type DBErrors string

const (
	HopInsertionError DBErrors = "failed to insert hop"
	IpInsertionError  DBErrors = "failed to insert geoip"
	ReadAggsError     DBErrors = "failed to read aggregated statistics"
	IpNotPresent      DBErrors = "ip not present in db"
	IpLookupError     DBErrors = "ip lookup error"
	IpUpdateError     DBErrors = "failed to update ip"
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

	aggs, err := pgx.CollectRows(rows, pgx.RowToStructByName[tmpHopsAgg])
	if err != nil {
		slog.Error("failed to collect rows", "error", err)
		return nil, ReadAggsError
	}

	res := make([]HopsAgg, len(aggs))
	for i, agg := range aggs {
		res[i] = HopsAgg{
			Source_ip:   agg.Source_ip.IPNet.IP.String(),
			Dest_ip:     agg.Dest_ip.IPNet.IP.String(),
			Count:       agg.Count,
			Avg_latency: agg.Avg_latency,
		}
	}

	return res, nil
}

func getIp(ip string, conn *pgxpool.Pool) (Geoip, error) {
	rows, err := conn.Query(context.Background(), "select * from geoip where ip_addr = $1", ip)
	if err != nil {
		slog.Error("failed to collect ip row", "error", err)
		// todo: return appropriate error
		return Geoip{}, err
	}
	res, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[tmpGeoip])
	if err != nil {
		if errors.Is(pgx.ErrNoRows, err) {
			slog.Info("ip address ")
			return Geoip{}, IpNotPresent
		} else {
			return Geoip{}, IpLookupError
		}
	}

	geoip := Geoip{
		Id:  res.Id,
		Ip:  res.Ip.IPNet.IP.String(),
		Lat: res.Lat,
		Lon: res.Lon,
		Isp: res.Isp,
		Org: res.Org,
		Asn: res.Asn,
	}

	return geoip, nil
}

func AddIp(ip Geoip, conn *pgxpool.Pool) error {
	_, err := conn.Exec(context.Background(), "insert into geoip values ($1, $2, $3, $4, $5, $6, $7)", ip.Id, ip.Ip, ip.Lat, ip.Lon, ip.Isp, ip.Org, ip.Asn)
	if err != nil {
		if strings.Contains(err.Error(), "23505") {
			slog.Info("ip already in DB")
			return nil
		}
		slog.Error("failed to insert ip", "error", err)
		return IpInsertionError
	}
	return nil
}

func UpdateIp(ip Geoip, conn *pgxpool.Pool) error {
	_, err := conn.Exec(context.Background(), "update geoip set lat = $1, lon = $2, isp = $3, org = $4, asn = $5 where id = $6", ip.Lat, ip.Lon, ip.Isp, ip.Org, ip.Asn, ip.Id)
	if err != nil {
		slog.Error("failed to update ip", "error", err)
		return IpUpdateError
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
