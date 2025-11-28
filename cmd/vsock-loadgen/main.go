package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stackrox/rox/compliance/virtualmachines/relay/vsock"
	"github.com/stackrox/rox/compliance/virtualmachines/testdata"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

type config struct {
	concurrency   int
	totalRequests uint64
	duration      time.Duration
	rateLimit     float64

	payloadSize string
	payloadFile string
	numPackages int
	numRepos    int
	randomize   bool

	startCID       uint
	port           uint
	metricsPort    int
	statsInterval  time.Duration
	requestTimeout time.Duration
}

// yamlConfig represents the structure of the YAML config file
type yamlConfig struct {
	Loadgen struct {
		Concurrency    int    `yaml:"concurrency"`
		TotalRequests  uint64 `yaml:"totalRequests"`
		RateLimit      int    `yaml:"rateLimit"`
		PayloadSize    string `yaml:"payloadSize"`
		StartCID       uint   `yaml:"startCID"`
		StatsInterval  string `yaml:"statsInterval"`
		Port           uint   `yaml:"port"`
		MetricsPort    int    `yaml:"metricsPort"`
		RequestTimeout string `yaml:"requestTimeout,omitempty"`
	} `yaml:"loadgen"`
}

func main() {
	cfg := parseConfig()
	ctx, cancel := context.WithCancel(context.Background())
	if cfg.duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, cfg.duration)
	}
	defer cancel()

	setupSignalHandler(cancel)

	baseReport, err := loadBaseReport(cfg)
	if err != nil {
		log.Fatalf("loading payload: %v", err)
	}

	payloads := newPayloadProvider(baseReport, cfg.randomize)
	stats := newStatsCollector()
	metrics := newMetricsRegistry()
	start := time.Now()

	if cfg.metricsPort > 0 {
		go serveMetrics(ctx, cfg.metricsPort)
	}

	requests := make(chan uint64)
	var workers sync.WaitGroup
	for i := 0; i < cfg.concurrency; i++ {
		workers.Add(1)
		go func(id int) {
			defer workers.Done()
			worker(ctx, id, cfg, payloads, requests, stats, metrics)
		}(i)
	}

	go func() {
		defer close(requests)
		produceRequests(ctx, cfg, requests)
	}()

	log.Printf("vsock-loadgen starting: concurrency=%d rate-limit=%.2f total-requests=%d duration=%s payload=%s randomize=%t port=%d start-cid=%d",
		cfg.concurrency, cfg.rateLimit, cfg.totalRequests, cfg.duration, describePayload(cfg), cfg.randomize, cfg.port, cfg.startCID)

	ticker := time.NewTicker(cfg.statsInterval)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-ticker.C:
			logSnapshot("progress", stats.snapshot(time.Since(start)))
		}
	}

	workers.Wait()
	logSnapshot("final", stats.snapshot(time.Since(start)))
}

func parseConfig() config {
	var configFile string
	var cfg config

	// Define flags
	flag.StringVar(&configFile, "config", "", "Path to YAML config file")
	flag.IntVar(&cfg.concurrency, "concurrency", 50, "Number of concurrent workers")
	flag.Uint64Var(&cfg.totalRequests, "total-requests", 0, "Total requests to send (0 = unbounded)")
	flag.DurationVar(&cfg.duration, "duration", 0, "Stop after this duration (0 = until completion)")
	flag.Float64Var(&cfg.rateLimit, "rate-limit", 0, "Requests per second (0 = unlimited)")

	flag.StringVar(&cfg.payloadSize, "payload-size", "small", "Embedded payload size to use (small|avg|large)")
	flag.StringVar(&cfg.payloadFile, "payload-file", "", "Path to .pb payload fixture (overrides payload-size)")
	flag.IntVar(&cfg.numPackages, "num-packages", 0, "Generate payload with this many packages (overrides payload-size)")
	flag.IntVar(&cfg.numRepos, "num-repos", 0, "Generate payload with this many repositories (default derived from packages)")
	flag.BoolVar(&cfg.randomize, "randomize", false, "Randomize hash/package versions per request")

	flag.UintVar(&cfg.startCID, "start-cid", 100, "Starting vsock CID to embed in payloads")
	flag.UintVar(&cfg.port, "port", 818, "Vsock port for the relay")
	flag.IntVar(&cfg.metricsPort, "metrics-port", 9090, "Expose Prometheus metrics on this port (0 = disabled)")
	flag.DurationVar(&cfg.statsInterval, "stats-interval", 10*time.Second, "Console stats interval")
	flag.DurationVar(&cfg.requestTimeout, "request-timeout", 10*time.Second, "Per-request vsock deadline")
	flag.Parse()

	// Track which flags were explicitly set
	setFlags := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	// Load YAML config if specified and apply values for flags that weren't explicitly set
	if configFile != "" {
		yamlCfg, err := loadYAMLConfig(configFile)
		if err != nil {
			log.Fatalf("loading config file: %v", err)
		}
		applyYAMLConfig(&cfg, yamlCfg, setFlags)
	}

	// Validation
	if cfg.concurrency <= 0 {
		log.Fatalf("concurrency must be > 0")
	}
	if cfg.rateLimit < 0 {
		log.Fatalf("rate-limit must be >= 0")
	}
	if cfg.statsInterval <= 0 {
		cfg.statsInterval = 10 * time.Second
	}
	return cfg
}

func loadYAMLConfig(path string) (*yamlConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	var cfg yamlConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	return &cfg, nil
}

func applyYAMLConfig(cfg *config, yamlCfg *yamlConfig, setFlags map[string]bool) {
	// Priority: CLI flag > config file > default
	// Only apply YAML values for flags that weren't explicitly set via command line

	if !setFlags["concurrency"] && yamlCfg.Loadgen.Concurrency > 0 {
		cfg.concurrency = yamlCfg.Loadgen.Concurrency
	}
	if !setFlags["total-requests"] && yamlCfg.Loadgen.TotalRequests > 0 {
		cfg.totalRequests = yamlCfg.Loadgen.TotalRequests
	}
	if !setFlags["rate-limit"] && yamlCfg.Loadgen.RateLimit > 0 {
		cfg.rateLimit = float64(yamlCfg.Loadgen.RateLimit)
	}
	if !setFlags["payload-size"] && yamlCfg.Loadgen.PayloadSize != "" {
		cfg.payloadSize = yamlCfg.Loadgen.PayloadSize
	}
	if !setFlags["start-cid"] && yamlCfg.Loadgen.StartCID > 0 {
		cfg.startCID = yamlCfg.Loadgen.StartCID
	}
	if !setFlags["port"] && yamlCfg.Loadgen.Port > 0 {
		cfg.port = yamlCfg.Loadgen.Port
	}
	if !setFlags["metrics-port"] {
		cfg.metricsPort = yamlCfg.Loadgen.MetricsPort
	}
	if !setFlags["stats-interval"] && yamlCfg.Loadgen.StatsInterval != "" {
		if d, err := time.ParseDuration(yamlCfg.Loadgen.StatsInterval); err == nil {
			cfg.statsInterval = d
		}
	}
	if !setFlags["request-timeout"] && yamlCfg.Loadgen.RequestTimeout != "" {
		if d, err := time.ParseDuration(yamlCfg.Loadgen.RequestTimeout); err == nil {
			cfg.requestTimeout = d
		}
	}
}

func loadBaseReport(cfg config) (*v1.IndexReport, error) {
	switch {
	case cfg.payloadFile != "":
		report, err := testdata.LoadFixture(cfg.payloadFile)
		if err != nil {
			return nil, fmt.Errorf("load fixture: %w", err)
		}
		return report, nil
	case cfg.numPackages > 0:
		report, err := testdata.GenerateIndexReport(testdata.Options{
			VsockCID:        uint32(cfg.startCID),
			NumPackages:     cfg.numPackages,
			NumRepositories: cfg.numRepos,
		})
		if err != nil {
			return nil, fmt.Errorf("generate report: %w", err)
		}
		return report, nil
	default:
		data, err := testdata.EmbeddedFixture(cfg.payloadSize)
		if err != nil {
			return nil, fmt.Errorf("read embedded fixture: %w", err)
		}
		report, err := testdata.LoadReportFromBytes(data)
		if err != nil {
			return nil, fmt.Errorf("load embedded report: %w", err)
		}
		return report, nil
	}
}

func describePayload(cfg config) string {
	switch {
	case cfg.payloadFile != "":
		return fmt.Sprintf("file:%s", cfg.payloadFile)
	case cfg.numPackages > 0:
		return fmt.Sprintf("generated:%d-packages", cfg.numPackages)
	default:
		return fmt.Sprintf("embedded:%s", cfg.payloadSize)
	}
}

func setupSignalHandler(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Printf("received shutdown signal, stopping...")
		cancel()
	}()
}

func serveMetrics(ctx context.Context, port int) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	log.Printf("metrics server listening on :%d", port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("metrics server error: %v", err)
	}
}

func produceRequests(ctx context.Context, cfg config, requests chan<- uint64) {
	var limiter *rate.Limiter
	if cfg.rateLimit > 0 {
		limiter = rate.NewLimiter(rate.Limit(cfg.rateLimit), 1)
	}

	var sent uint64
	for {
		if cfg.totalRequests > 0 && sent >= cfg.totalRequests {
			return
		}
		if limiter != nil {
			if err := limiter.Wait(ctx); err != nil {
				return
			}
		} else {
			select {
			case <-ctx.Done():
				return
			default:
			}
		}

		sent++
		select {
		case <-ctx.Done():
			return
		case requests <- sent:
		}
	}
}

type payloadProvider struct {
	base      *v1.IndexReport
	randomize bool
	randMu    sync.Mutex
	rng       *rand.Rand
}

func newPayloadProvider(base *v1.IndexReport, randomize bool) *payloadProvider {
	return &payloadProvider{
		base:      proto.Clone(base).(*v1.IndexReport),
		randomize: randomize,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (p *payloadProvider) Next(cid uint32) ([]byte, error) {
	report := proto.Clone(p.base).(*v1.IndexReport)
	report.VsockCid = fmt.Sprintf("%d", cid)

	baseHash := report.GetIndexV4().GetHashId()
	report.IndexV4.HashId = fmt.Sprintf("%s-%d", baseHash, cid)

	if p.randomize {
		p.applyRandomization(report, cid)
	}
	data, err := proto.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("marshal report: %w", err)
	}
	return data, nil
}

func (p *payloadProvider) applyRandomization(report *v1.IndexReport, cid uint32) {
	r := func() *rand.Rand {
		p.randMu.Lock()
		defer p.randMu.Unlock()
		return rand.New(rand.NewSource(p.rng.Int63()))
	}()
	report.IndexV4.HashId = fmt.Sprintf("%s-%d-%d", report.GetIndexV4().GetHashId(), cid, r.Int63())

	packages := report.GetIndexV4().GetContents().GetPackages()
	updated := 0
	for _, pkg := range packages {
		pkg.Version = fmt.Sprintf("%s-%d", pkg.GetVersion(), r.Intn(10000))
		updated++
		if updated >= 5 {
			break
		}
	}
}

func worker(ctx context.Context, id int, cfg config, provider *payloadProvider, requests <-chan uint64, stats *statsCollector, metrics *metricsRegistry) {
	for {
		select {
		case <-ctx.Done():
			return
		case seq, ok := <-requests:
			if !ok {
				return
			}
			cid := uint32(cfg.startCID) + uint32(seq-1)
			payload, err := provider.Next(cid)
			if err != nil {
				log.Printf("worker %d: payload error: %v", id, err)
				metrics.observeFailure("payload")
				stats.recordFailure()
				continue
			}
			start := time.Now()
			err = sendReport(payload, cfg.requestTimeout)
			latency := time.Since(start)
			if err != nil {
				log.Printf("worker %d: send error: %v", id, err)
				metrics.observeFailure(errorLabel(err))
				stats.recordFailure()
				continue
			}
			metrics.observeSuccess(latency, len(payload))
			stats.recordSuccess(latency, len(payload))
		}
	}
}

func sendReport(payload []byte, timeout time.Duration) error {
	// Use DialLocal for loopback connections when running on the same host as the relay
	// This is the typical case for load testing with vsock-loadgen in the collector pod
	conn, err := vsock.DialLocal()
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if timeout > 0 {
		_ = conn.SetDeadline(time.Now().Add(timeout))
	}

	n, err := conn.Write(payload)
	if err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if n != len(payload) {
		return fmt.Errorf("short write: wrote %d of %d bytes", n, len(payload))
	}
	return nil
}

func errorLabel(err error) string {
	if err == nil {
		return "success"
	}
	switch {
	case errors.Is(err, context.Canceled):
		return "context"
	case strings.Contains(err.Error(), "dial"):
		return "dial"
	case strings.Contains(err.Error(), "write"):
		return "write"
	default:
		return "send"
	}
}

type statsCollector struct {
	total   atomic.Uint64
	success atomic.Uint64
	failure atomic.Uint64
	bytes   atomic.Uint64

	mu        sync.Mutex
	histogram *hdrhistogram.Histogram
}

func newStatsCollector() *statsCollector {
	return &statsCollector{
		histogram: hdrhistogram.New(1, int64((5*time.Minute)/time.Microsecond), 3),
	}
}

func (s *statsCollector) recordSuccess(latency time.Duration, bytes int) {
	s.total.Add(1)
	s.success.Add(1)
	if bytes > 0 {
		s.bytes.Add(uint64(bytes))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.histogram.RecordValue(latency.Microseconds())
}

func (s *statsCollector) recordFailure() {
	s.total.Add(1)
	s.failure.Add(1)
}

type statsSnapshot struct {
	Total      uint64
	Success    uint64
	Failure    uint64
	Bytes      uint64
	P50        time.Duration
	P95        time.Duration
	P99        time.Duration
	Elapsed    time.Duration
	Throughput float64
}

func (s *statsCollector) snapshot(elapsed time.Duration) statsSnapshot {
	total := s.total.Load()
	throughput := 0.0
	if elapsed > 0 {
		throughput = float64(total) / elapsed.Seconds()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	p50 := durationFromHist(s.histogram, 50)
	p95 := durationFromHist(s.histogram, 95)
	p99 := durationFromHist(s.histogram, 99)

	return statsSnapshot{
		Total:      total,
		Success:    s.success.Load(),
		Failure:    s.failure.Load(),
		Bytes:      s.bytes.Load(),
		P50:        p50,
		P95:        p95,
		P99:        p99,
		Elapsed:    elapsed,
		Throughput: throughput,
	}
}

func durationFromHist(h *hdrhistogram.Histogram, percentile float64) time.Duration {
	if h.TotalCount() == 0 {
		return 0
	}
	return time.Microsecond * time.Duration(h.ValueAtQuantile(percentile))
}

func logSnapshot(prefix string, snap statsSnapshot) {
	mbSent := float64(snap.Bytes) / (1024.0 * 1024.0)
	log.Printf("[%s] sent=%d success=%d failure=%d throughput=%.2f req/s data=%.2f MiB p50=%s p95=%s p99=%s",
		prefix, snap.Total, snap.Success, snap.Failure, snap.Throughput, mbSent, snap.P50, snap.P95, snap.P99)
}

type metricsRegistry struct {
	requests *prometheus.CounterVec
	bytes    prometheus.Counter
	latency  prometheus.Histogram
}

func newMetricsRegistry() *metricsRegistry {
	m := &metricsRegistry{
		requests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "vsock_loadgen_requests_total",
				Help: "Total vsock load generator requests by result.",
			},
			[]string{"result"},
		),
		bytes: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "vsock_loadgen_bytes_total",
				Help: "Total bytes sent to the relay.",
			},
		),
		latency: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "vsock_loadgen_request_latency_seconds",
				Help:    "Request latency in seconds.",
				Buckets: prometheus.ExponentialBuckets(0.01, 1.4, 15),
			},
		),
	}
	prometheus.MustRegister(m.requests, m.bytes, m.latency)
	return m
}

func (m *metricsRegistry) observeSuccess(latency time.Duration, bytes int) {
	m.requests.WithLabelValues("success").Inc()
	m.bytes.Add(float64(bytes))
	m.latency.Observe(latency.Seconds())
}

func (m *metricsRegistry) observeFailure(reason string) {
	m.requests.WithLabelValues(reason).Inc()
}
