package main

import (
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/voltracker/trace-router/internal"
	"github.com/voltracker/trace-router/internal/server"
)

import _ "embed"

var conn *pgxpool.Pool

const apiPrefix = "/api/v1/"

func getHandle() *pcap.Handle {
	if os.Getenv("TRACE_ROUTER_LIVE") != "1" {
		if handle, err := pcap.OpenOffline(os.Getenv("TRACE_ROUTER_INPUT_FILE")); err != nil {
			panic(err)
		} else {
			return handle
		}
	}
	if handle, err := pcap.OpenLive(os.Getenv("TRACE_ROUTER_INTERFACE"), 1600, true, pcap.BlockForever); err != nil {
		panic(err)
	} else {
		return handle
	}
}

func parseOutput(output string) []internal.Hop {
	var lines = strings.Split(output, "\n")
	var hops = []internal.Hop{}
	for line := range lines {
		if !strings.Contains(lines[line], "*") {
			split := strings.Fields(lines[line])
			if len(split) > 2 {
				val, err := strconv.ParseFloat(split[2], 32)
				if err != nil {
					slog.Error("failed to parse float from traceroute output")
				}
				var src string
				src = "localhost"
				if len(hops) >= 1 {
					src = hops[len(hops)-1].Dest
				}
				hops = append(hops, internal.Hop{
					Id:      uuid.New(),
					Src:     src,
					Dest:    split[1],
					Latency: val,
				})
			}
		}
	}
	return hops
}

func runTraceroute(src string, dst string, channel chan internal.TraceResult) {
	out, err := exec.Command("/usr/sbin/traceroute", "-n", "-q 1", dst).Output()
	if err != nil {
		slog.Error("failed to execute command", "error", err)
	}
	hops := parseOutput(string(out))
	slog.Info("finished traceroute for " + dst + "...")
	channel <- internal.TraceResult{InitialSrc: src, FinalDst: dst, Hops: hops}
}

func handlePacket(packet gopacket.Packet, wg *sync.WaitGroup, channel chan internal.TraceResult) {
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer != nil {
		ipv4, ok := ipLayer.(*layers.IPv4)
		if ok && !ipv4.DstIP.IsPrivate() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				runTraceroute("localhost", ipv4.DstIP.String(), channel)
			}()
			slog.Info("starting traceroute to " + ipv4.DstIP.String() + "...")
		}
	}
}

func main() {

	pool, err := internal.Connect(os.Getenv("CONNECTION_STRING"))
	if err != nil {
		slog.Error("Error connecting to database", "error", err)
		os.Exit(69)
	}

	conf := &server.ServerConfig{
		ApiPrefix:    apiPrefix,
		DBConnection: pool,
	}

	go server.Start(conf)
	handle := getHandle()
	var wg sync.WaitGroup
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	results := make(chan internal.TraceResult)
	for packet := range packetSource.Packets() {
		handlePacket(packet, &wg, results)
	}

	go func() {
		wg.Wait()
		close(results)
	}()
	slog.Info("finished running traceroutes...")
	for res := range results {
		// slog.Info("Traceroute for " + res.initialSrc + " -> " + res.finalDst)
		for _, hop := range res.Hops {
			// slog.Info("Hop "+strconv.Itoa(i)+": ", "src", hop.src, "dest", hop.dest, "latency", hop.latency)
			internal.AddHop(hop, conn)
		}
	}
	select {}
}
