package collector_test

import (
	"fmt"
	"net"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/rumblefrog/go-a2s"

	"github.com/armsnyder/a2s-exporter/internal/collector"
	"github.com/armsnyder/a2s-exporter/internal/testserver"
)

// TestCollector_Describe_PrintTable tests the Describe function.
// It also prints a Markdown-formatted table of all registered metrics, which can be copied to the README.
func TestCollector_Describe_PrintTable(t *testing.T) {
	c := collector.New("", "")
	descs := testDescribe(c)
	if len(descs) == 0 {
		t.Error("expected Descs but got none")
	}

	// HACK: The only exported method on Desc is String().
	pattern := regexp.MustCompile(`fqName: "([a-z_]+)", help: "(.+)", constLabels: .+, variableLabels: \[([^]]*)]`)
	for _, desc := range descs {
		match := pattern.FindStringSubmatch(desc.String())
		if match == nil {
			t.Errorf("failed pattern match for Desc %s", desc)
			continue
		}
		fmt.Println(strings.Join(match[1:], " | "))
	}
}

// testDescribe returns all Descs from the provided Collector, sorted.
func testDescribe(c prometheus.Collector) (descs []*prometheus.Desc) {
	ch := make(chan *prometheus.Desc)
	done := make(chan bool)

	go func() {
		for desc := range ch {
			descs = append(descs, desc)
		}
		sort.Slice(descs, func(i, j int) bool { return descs[i].String() < descs[j].String() })
		close(done)
	}()

	c.Describe(ch)
	close(ch)
	<-done

	return descs
}

func TestCollector(t *testing.T) {
	// Run a test A2S server.
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		t.Fatal(err)
	}
	srv := &testserver.TestServer{
		ServerInfo: &a2s.ServerInfo{
			Name:       "foo",
			Players:    3,
			MaxPlayers: 6,
		},
		PlayerInfo: &a2s.PlayerInfo{
			Count: 3,
			Players: []*a2s.Player{
				{
					Index:    0,
					Name:     "jon",
					Duration: 32,
				},
				{
					Index:    0,
					Name:     "alice",
					Duration: 64,
				},
				// Duplicate players should be de-duplicated to avoid causing registry errors.
				{
					Index:    0,
					Name:     "alice",
					Duration: 99,
				},
			},
		},
	}
	go func() {
		t.Error(srv.Serve(conn))
	}()

	// Set up the registry and gather metrics from the test A2S server.
	registry := prometheus.NewPedanticRegistry()
	registry.MustRegister(collector.New("", conn.LocalAddr().String()))
	metrics, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}

	// Spot check the gathered metrics.
	testAssertGauge(t, metrics, "server_players",
		expectGauge{value: 3, labels: map[string]string{"server_name": "foo"}},
	)
	testAssertGauge(t, metrics, "server_max_players",
		expectGauge{value: 6, labels: map[string]string{"server_name": "foo"}},
	)
	testAssertGauge(t, metrics, "player_count",
		expectGauge{value: 3, labels: map[string]string{"server_name": "foo"}},
	)
	testAssertGauge(t, metrics, "player_duration",
		expectGauge{value: 32, labels: map[string]string{"server_name": "foo", "player_index": "0", "player_name": "jon"}},
		expectGauge{value: 64, labels: map[string]string{"server_name": "foo", "player_index": "0", "player_name": "alice"}},
	)
}

type expectGauge struct {
	value  float64
	labels map[string]string
}

func testAssertGauge(t *testing.T, metricFamilies []*io_prometheus_client.MetricFamily, name string, expectGauges ...expectGauge) {
	t.Helper()

	for _, family := range metricFamilies {
		if family.GetName() != name {
			continue
		}

		metrics := family.GetMetric()
		if len(metrics) != len(expectGauges) {
			t.Errorf("metric %s count mismatch: wanted %d, got %d", name, len(expectGauges), len(metrics))
			return
		}

	nextExpectedGauge:
		for _, expectedGauge := range expectGauges {
			for _, metric := range metrics {
				if testMatchGauge(expectedGauge, metric) {
					continue nextExpectedGauge
				}
			}
			t.Errorf("metric %s did not contain an expected gauge %v", name, expectedGauge)
		}

		return
	}

	t.Errorf("exected metric %s not found", name)
}

func testMatchGauge(expectedGauge expectGauge, metric *io_prometheus_client.Metric) bool {
	gotGauge := metric.GetGauge()
	if gotGauge == nil {
		return false
	}

	if expectedGauge.value != metric.GetGauge().GetValue() {
		return false
	}

nextExpectedLabel:
	for k, v := range expectedGauge.labels {
		for _, label := range metric.GetLabel() {
			if label.GetName() == k && label.GetValue() == v {
				continue nextExpectedLabel
			}
		}
		return false
	}

	return true
}
