package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NodeList map[string]NodeInfo

type Location struct {
	Latitude  float32 `json:"lat"`
	Longitude float32 `json:"lon"`
	Query     string  `json:"query"`
	Isp       string  `json:"isp"`
	Org       string  `json:"org"`
	Asn       string  `json:"as"`
}

type NodeInfo struct {
	Ip        string  `json:"ip"`
	Count     int     `json:"count"`
	Latitude  float32 `json:"lat"`
	Longitude float32 `json:"lon"`
	Isp       string  `json:"isp"`
	Org       string  `json:"org"`
	Asn       string  `json:"as"`
}

func GetUniqueNodes(aggs []HopsAgg, conn *pgxpool.Pool) NodeList {
	uniques := make(NodeList)
	for _, agg := range aggs {
		incrementNodeList(uniques, agg.Source_ip)
		incrementNodeList(uniques, agg.Dest_ip)
	}
	complete, err := addLocations(uniques, conn)
	slog.Info("finished adding locations")
	if err != nil {
		slog.Error("error getting locations for ips", "error", err)
	}
	return complete
}

func incrementNodeList(nodeList NodeList, ip string) {
	if info, ok := nodeList[ip]; ok {
		nodeList[ip] = NodeInfo{Ip: info.Ip, Count: info.Count + 1, Latitude: info.Latitude, Longitude: info.Longitude}
	} else {
		nodeList[ip] = NodeInfo{Ip: ip, Count: 1, Latitude: 0, Longitude: 0}
	}
}

func (l Location) toGeoip() Geoip {
	return Geoip{uuid.New(), l.Query, l.Latitude, l.Longitude, l.Isp, l.Org, l.Asn}
}

func addLocations(nodes NodeList, conn *pgxpool.Pool) (NodeList, error) {
	ips := []Geoip{}
	ipsInDb := []Geoip{}
	for ip := range nodes {
		geoip, err := getIp(ip, conn)
		if errors.Is(IpNotPresent, err) {
			panic(1)
		} else {
			if geoip.Org == "" {
				ips = append(ips, geoip)
			} else {
				ipsInDb = append(ipsInDb, geoip)
			}
		}
	}
	tmpNodes := make(NodeList)
	base := 0
	for base+99 < len(ips) {
		slog.Info("grabbing locations", "from", base, "to", base+99, "count", len(ips[base:base+99]))
		locs, err := getLocations(ips[base : base+99])
		if err != nil {
			slog.Error("failed to retrieve locations for ips", "error", err)
			return nil, err
		} else {
			for _, loc := range locs {
				UpdateIp(loc, conn)
				info := nodes[loc.Ip]
				slog.Info("got location", "ip", loc.Ip, "latitude", loc.Lat, "longitude", loc.Lon)
				tmpNodes[loc.Ip] = NodeInfo{Ip: loc.Ip, Count: info.Count, Longitude: loc.Lon, Latitude: loc.Lat}
			}
			base += 100
		}
	}
	slog.Info("grabbing locations", "from", base, "to", len(ips), "count", len(ips[base:len(ips)-1]))
	locs, err := getLocations(ips[base : len(ips)-1])
	if err != nil {
		slog.Error("failed to retrieve locations for ips", "error", err)
		return nil, err
	} else {
		for _, loc := range locs {
			UpdateIp(loc, conn)
			info := nodes[loc.Ip]
			slog.Info("got location", "ip", loc.Ip, "latitude", loc.Lat, "longitude", loc.Lon)
			tmpNodes[loc.Ip] = NodeInfo{Ip: loc.Ip, Count: info.Count, Longitude: loc.Lon, Latitude: loc.Lat}
		}
	}

	for _, ip := range ipsInDb {
		info := nodes[ip.Ip]
		tmpNodes[ip.Ip] = NodeInfo{Ip: ip.Ip, Count: info.Count, Longitude: ip.Lon, Latitude: ip.Lat}
	}

	return tmpNodes, nil
}

func getLocations(ips []Geoip) ([]Geoip, error) {
	if len(ips) > 100 {
		return nil, errors.New("can't request more than 100 ips at a time")
	}
	rawips := make([]string, len(ips))
	for i := range ips {
		rawips[i] = ips[i].Ip
	}
	requestStr := "http://ip-api.com/batch?fields=11968"
	requestBody, err := json.Marshal(rawips)
	slog.Info("body", "content", requestBody)
	if err != nil {
		slog.Error("failed to marshal ip array to json", "error", err)
		return nil, err
	}
	reader := bytes.NewBuffer(requestBody)
	resp, err := http.Post(requestStr, "application/json", reader)
	if err != nil {
		slog.Error("failed to fetch location", "error", err)
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("failed to read body of location response", "error", err)
		return nil, err
	}
	locations := make([]Location, len(ips))
	err = json.Unmarshal(body, &locations)
	if err != nil {
		slog.Error("Failed to parse json", "input", body, "error", err)
		return nil, err
	}

	for _, loc := range locations {
		for i, _ := range ips {
			if loc.Query == ips[i].Ip {
				ips[i].Lat = loc.Latitude
				ips[i].Lon = loc.Longitude
				ips[i].Isp = loc.Isp
				ips[i].Org = loc.Org
				ips[i].Asn = loc.Asn
			}
		}
	}
	return ips, nil
}
