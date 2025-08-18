package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"sort"
)

func HttpGetAggs(w http.ResponseWriter, req *http.Request) {
	hops, err := GetAggs(conn)
	if req.Method != http.MethodGet {
		slog.Info("tried to use method other than GET on getAggs")
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error retrieving hops")
		return
	}
	sort.Slice(hops[:], func(i, j int) bool {
		return hops[i].Count > hops[j].Count
	})
	for _, agg := range hops {
		if agg.Source_ip != "localhost" {
			fmt.Fprintf(w, "source = %s, destination = %s, latency = %f, count = %d \n", agg.Source_ip, agg.Dest_ip, agg.Avg_latency, agg.Count)
		}
	}

}
