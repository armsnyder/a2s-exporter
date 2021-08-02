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

	"github.com/armsnyder/a2s-exporter/internal/collector"
)

func main() {
	// Flags.
	address := flag.String("address", envOrDefault("A2S_EXPORTER_QUERY_ADDRESS", ""), "Address of the A2S query server as host:port (This is a separate port from the main server port).")
	port := flag.Int("port", envOrDefaultInt("A2S_EXPORTER_PORT", 9856), "Port for the metrics exporter.")
	path := flag.String("path", envOrDefault("A2S_EXPORTER_PATH", "/metrics"), "Path for the metrics exporter.")
	namespace := flag.String("namespace", envOrDefault("A2S_EXPORTER_NAMESPACE", "a2s"), "Namespace prefix for all exported a2s metrics.")
	a2sOnlyMetrics := flag.Bool("a2s-only-metrics", envOrDefaultBool("A2S_EXPORTER_A2S_ONLY_METRICS", false), "If true, skips exporting Go runtime metrics.")
	help := flag.Bool("h", false, "Show help.")

	flag.Parse()

	defer os.Exit(1)

	// Show help.
	if *help || flag.NArg() > 0 {
		flag.Usage()
		return
	}

	// Check required arguments.
	if *address == "" {
		fmt.Println("address argument is required")
		flag.Usage()
		return
	}

	// Set up prometheus metrics registry.
	var registry *prometheus.Registry
	if *a2sOnlyMetrics {
		registry = prometheus.NewRegistry()
	} else {
		registry = prometheus.DefaultRegisterer.(*prometheus.Registry)
	}

	// Register A2S metrics.
	registry.MustRegister(collector.New(*namespace, *address))

	// Set up http handler.
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	if !*a2sOnlyMetrics {
		handler = promhttp.InstrumentMetricHandler(registry, handler)
	}

	http.Handle(*path, handler)

	// Run http server.
	fmt.Printf("Serving metrics at http://127.0.0.1:%d%s\n", *port, *path)
	fmt.Println(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
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
