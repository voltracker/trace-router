package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
)

type NodeList map[string]NodeInfo

type Location struct {
	Latitude  float32 `json:"lat"`
	Longitude float32 `json:"lon"`
	Query     string  `json:"query"`
}

type NodeInfo struct {
	Ip        string  `json:"ip"`
	Count     int     `json:"count"`
	Latitude  float32 `json:"lat"`
	Longitude float32 `json:"lon"`
}

func GetUniqueNodes(aggs []HopsAgg) NodeList {
	uniques := make(NodeList)
	for _, agg := range aggs {
		incrementNodeList(uniques, agg.Source_ip)
		incrementNodeList(uniques, agg.Dest_ip)
	}
	complete, err := addLocations(uniques)
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

func addLocations(nodes NodeList) (NodeList, error) {
	ips := make([]string, len(nodes))
	i := 0
	for ip := range nodes {
		ips[i] = ip
		i++
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
				info := nodes[loc.Query]
				slog.Info("got location", "ip", loc.Query, "latitude", loc.Latitude, "longitude", loc.Longitude)
				tmpNodes[loc.Query] = NodeInfo{Ip: loc.Query, Count: info.Count, Longitude: loc.Longitude, Latitude: loc.Latitude}
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
			info := nodes[loc.Query]
			slog.Info("got location", "ip", loc.Query, "latitude", loc.Latitude, "longitude", loc.Longitude)
			tmpNodes[loc.Query] = NodeInfo{Ip: loc.Query, Count: info.Count, Longitude: loc.Longitude, Latitude: loc.Latitude}
		}
	}
	return tmpNodes, nil
}

func getLocations(ips []string) ([]Location, error) {
	if len(ips) > 100 {
		return nil, errors.New("can't request more than 100 ips at a time")
	}
	requestStr := "http://ip-api.com/batch?fields=8384"
	requestBody, err := json.Marshal(ips)
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
	return locations, nil
}
