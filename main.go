package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rumblefrog/go-a2s"

	"github.com/armsnyder/a2s-exporter/internal/collector"
)

// buildVersion variable is set at build time.
var buildVersion = "development"

func main() {
	// Flags.
	address := flag.String("address", envOrDefault("A2S_EXPORTER_QUERY_ADDRESS", ""), "Address of the A2S query server as host:port (This is a separate port from the main server port).")
	port := flag.Int("port", envOrDefaultInt("A2S_EXPORTER_PORT", 9841), "Port for the metrics exporter.")
	path := flag.String("path", envOrDefault("A2S_EXPORTER_PATH", "/metrics"), "Path for the metrics exporter.")
	namespace := flag.String("namespace", envOrDefault("A2S_EXPORTER_NAMESPACE", "a2s"), "Namespace prefix for all exported a2s metrics.")
	a2sOnlyMetrics := flag.Bool("a2s-only-metrics", envOrDefaultBool("A2S_EXPORTER_A2S_ONLY_METRICS", false), "If true, skips exporting Go runtime metrics.")
	maxPacketSize := flag.Int("max-packet-size", envOrDefaultInt("A2S_EXPORTER_MAX_PACKET_SIZE", 1400), "Advanced option to set a non-standard max packet size of the A2S query server.")
	help := flag.Bool("h", false, "Show help.")
	version := flag.Bool("version", false, "Show build version.")

	flag.Parse()

	// Show version.
	if *version || flag.Arg(0) == "version" {
		fmt.Println(buildVersion)
		os.Exit(0)
	}

	// Show help.
	if *help || flag.NArg() > 0 {
		flag.Usage()
		os.Exit(1)
	}

	// Check required arguments.
	if *address == "" {
		fmt.Println("address argument is required")
		flag.Usage()
		os.Exit(1)
	}

	// Set up prometheus metrics registry.
	var registry *prometheus.Registry
	if *a2sOnlyMetrics {
		registry = prometheus.NewRegistry()
	} else {
		registry = prometheus.DefaultRegisterer.(*prometheus.Registry)
	}

	// Register A2S metrics.
	clientOptions := []func(*a2s.Client) error{
		a2s.SetMaxPacketSize(uint32(*maxPacketSize)),
	}
	registry.MustRegister(collector.New(*namespace, *address, clientOptions...))

	// Set up http handler.
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	if !*a2sOnlyMetrics {
		handler = promhttp.InstrumentMetricHandler(registry, handler)
	}

	http.Handle(*path, handler)

	// Run http server.
	fmt.Printf("Serving metrics at http://127.0.0.1:%d%s\n", *port, *path)
	fmt.Println(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))

	os.Exit(1)
}

func envOrDefault(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func envOrDefaultInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok {
		v2, _ := strconv.Atoi(v)
		return v2
	}
	return def
}

func envOrDefaultBool(key string, def bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		return !strings.EqualFold(v, "false") && !strings.EqualFold(v, "0")
	}
	return def
}
