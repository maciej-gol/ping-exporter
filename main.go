package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PingCollector struct {
	packetsSent *prometheus.CounterVec
	packetsRecv *prometheus.CounterVec
	packetsRtt  *prometheus.HistogramVec
}

func (pc *PingCollector) Collect(ch chan<- prometheus.Metric) {
	pc.packetsSent.Collect(ch)
	pc.packetsRecv.Collect(ch)
	pc.packetsRtt.Collect(ch)
}

func (pc *PingCollector) Describe(ch chan<- *prometheus.Desc) {
	pc.packetsSent.Describe(ch)
	pc.packetsRecv.Describe(ch)
	pc.packetsRtt.Describe(ch)
}

func (pc *PingCollector) RegisterPinger(pinger *probing.Pinger) {
	pinger.OnSetup = func() {
		pc.packetsSent.With(prometheus.Labels{"target": pinger.Addr()})
		pc.packetsRecv.With(prometheus.Labels{"target": pinger.Addr()})
		pc.packetsRtt.With(prometheus.Labels{"target": pinger.Addr()})
	}
	pinger.OnSend = func(packet *probing.Packet) {
		pc.packetsSent.With(prometheus.Labels{"target": pinger.Addr()}).Inc()
	}
	pinger.OnRecv = func(packet *probing.Packet) {
		pc.packetsRecv.With(prometheus.Labels{"target": pinger.Addr()}).Inc()
		pc.packetsRtt.With(prometheus.Labels{"target": pinger.Addr()}).Observe(packet.Rtt.Seconds())
	}
}

func NewPingCollector() *PingCollector {
	return &PingCollector{
		packetsSent: prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "ping_packets_sent", Name: "total", Help: "Total number of packets sent"}, []string{"target"}),
		packetsRecv: prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: "ping_packets_recv", Name: "total", Help: "Total number of packets received"}, []string{"target"}),
		packetsRtt:  prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace: "ping_packets_rtt", Name: "seconds", Help: "Round trip time in seconds"}, []string{"target"}),
	}
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "my string representation"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

var targets arrayFlags
var port string

func init() {
	flag.Var(&targets, "target", "Target address")
	flag.StringVar(&port, "port", "2112", "Port number")
}

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	collector := NewPingCollector()
	prometheus.MustRegister(collector)

	for _, target := range targets {
		go func(target string) {
			pinger, err := probing.NewPinger(target)
			if err != nil {
				panic(err)
			}
			collector.RegisterPinger(pinger)
			fmt.Println("Pinging", target)
			pinger.RunWithContext(ctx)
		}(target)
	}

	http.Handle("/metrics", promhttp.Handler())
	fmt.Println("Listening on", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
