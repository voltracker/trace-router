package server

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voltracker/trace-router/internal"
)

var conn *pgxpool.Pool

//go:embed stub_response.json
var stubbed_nodes []byte

//go:embed stubbed_aggs.json
var stubbed_aggs []byte

type ServerConfig struct {
	ApiPrefix    string
	DBConnection *pgxpool.Pool
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func stubbedNodes(w http.ResponseWriter, req *http.Request) {
	slog.Info("req", "request", req)
	w.Write(stubbed_nodes)
}

func stubbedAggs(w http.ResponseWriter, req *http.Request) {
	slog.Info("req", "request", req)
	w.Write(stubbed_aggs)
}

func Start(config *ServerConfig) {
	mux := http.NewServeMux()

	mux.HandleFunc(config.ApiPrefix+"aggs/", stubbedAggs)
	mux.HandleFunc(config.ApiPrefix+"nodes/", stubbedNodes)
	mux.HandleFunc(config.ApiPrefix+"aggs", stubbedAggs)
	mux.HandleFunc(config.ApiPrefix+"nodes", stubbedNodes)

	handler := corsMiddleware(mux)

	if err := http.ListenAndServe(":8080", handler); err != nil {
		slog.Error("HTTP server error", "error", err)
		os.Exit(1)
	}
}

func HttpGetAggs(w http.ResponseWriter, req *http.Request) {
	hops, err := internal.GetAggs(conn)
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
	slog.Info("retrieving nodes")
	hops, err := internal.GetAggs(conn)
	slog.Info("retrieved aggs")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error retrieving hops")
		return
	}
	nodeList := internal.GetUniqueNodes(hops)
	slog.Info("retrieving unique nodes")
	arr := []internal.NodeInfo{}
	for _, v := range nodeList {
		arr = append(arr, v)
	}
	out, err := json.Marshal(arr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "error retrieving hops")
		return
	}
	w.Write(out)
}
