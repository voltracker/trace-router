package main

import (
	"encoding/json"
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
	out, err := json.Marshal(hops)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error retrieving hops")
		return
	}
	w.Write(out)

}

func HttpGetNodes(w http.ResponseWriter, req *http.Request) {
	hops, err := GetAggs(conn)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error retrieving hops")
		return
	}
	nodeList := GetUniqueNodes(hops)
	out, err := json.Marshal(nodeList)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error retrieving hops")
		return
	}
	w.Write(out)
}
