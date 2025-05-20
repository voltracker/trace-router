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
	"github.com/jackc/pgx/v5/pgxpool"
)

type Hop struct {
	src     string
	dest    string
	latency float64
}

type TraceResult struct {
	initialSrc string
	finalDst   string
	hops       []Hop
}

var conn *pgxpool.Pool

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

func parseOutput(output string) []Hop {
	var lines = strings.Split(output, "\n")
	var hops = []Hop{}
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
					src = hops[len(hops)-1].dest
				}
				hops = append(hops, Hop{
					src:     src,
					dest:    split[1],
					latency: val,
				})
			}
		}
	}
	return hops
}

func runTraceroute(src string, dst string, channel chan TraceResult) {
	out, err := exec.Command("/usr/sbin/traceroute", "-n", "-q 1", dst).Output()
	if err != nil {
		slog.Error("failed to execute command", "error", err)
	}
	hops := parseOutput(string(out))
	slog.Info("finished traceroute for " + dst + "...")
	channel <- TraceResult{initialSrc: src, finalDst: dst, hops: hops}
}

func handlePacket(packet gopacket.Packet, wg *sync.WaitGroup, channel chan TraceResult) {
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
	handle := getHandle()
	pool, err := Connect("connectionString")
	if err != nil {
		slog.Error("Error connecting to database", "error", err)
		os.Exit(69)
	}
	conn = pool
	var wg sync.WaitGroup
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	results := make(chan TraceResult)
	for packet := range packetSource.Packets() {
		handlePacket(packet, &wg, results)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	slog.Info("finished running traceroutes...")
	for res := range results {
		slog.Info("Traceroute for " + res.initialSrc + " -> " + res.finalDst)
		for i, hop := range res.hops {
			slog.Info("Hop "+strconv.Itoa(i)+": ", "src", hop.src, "dest", hop.dest, "latency", hop.latency)
		}
	}
}
