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
	"sort"
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
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type config struct {
	vmCount  int
	duration time.Duration

	payloadSize string
	payloadFile string
	numPackages int
	numRepos    int

	port           uint
	metricsPort    int
	statsInterval  time.Duration
	requestTimeout time.Duration
	reportInterval time.Duration
}

// yamlConfig represents the structure of the YAML config file
type yamlConfig struct {
	Loadgen struct {
		VmCount        int    `yaml:"vmCount"`
		PayloadSize    string `yaml:"payloadSize"`
		StatsInterval  string `yaml:"statsInterval"`
		Port           uint   `yaml:"port"`
		MetricsPort    int    `yaml:"metricsPort"`
		RequestTimeout string `yaml:"requestTimeout,omitempty"`
		ReportInterval string `yaml:"reportInterval,omitempty"`
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

	// Calculate CID range based on node index to avoid overlap between DaemonSet pods
	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Fatalf("NODE_NAME environment variable not set")
	}

	startCID, endCID, nodeIndex, totalNodes, vmsThisNode, err := calculateCIDRange(ctx, nodeName, cfg.vmCount)
	if err != nil {
		log.Fatalf("calculating CID range: %v", err)
	}

	log.Printf("Node %s (index %d/%d) assigned CID range [%d-%d] for %d VMs (total cluster: %d VMs)",
		nodeName, nodeIndex, totalNodes, startCID, endCID, vmsThisNode, cfg.vmCount)

	payloads, err := newPayloadProviderWithRange(baseReport, vmsThisNode, startCID)
	if err != nil {
		log.Fatalf("creating payload provider: %v", err)
	}
	stats := newStatsCollector()
	metrics := newMetricsRegistry()
	start := time.Now()

	if cfg.metricsPort > 0 {
		go serveMetrics(ctx, cfg.metricsPort)
	}

	// Spawn one goroutine per VM, each with a unique CID from our assigned range
	var vms sync.WaitGroup
	for i := 0; i < vmsThisNode; i++ {
		cid := startCID + uint32(i)
		vms.Add(1)
		go func(vmCID uint32) {
			defer vms.Done()
			vmSimulator(ctx, vmCID, cfg, payloads, stats, metrics)
		}(cid)
	}

	log.Printf("vsock-loadgen starting: vms=%d report-interval=%s duration=%s payload=%s cid-range=[%d-%d] port=%d",
		vmsThisNode, cfg.reportInterval, cfg.duration, describePayload(cfg), startCID, endCID, cfg.port)

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

	vms.Wait()
	logSnapshot("final", stats.snapshot(time.Since(start)))
}

func parseConfig() config {
	var configFile string
	var cfg config

	// Define flags
	flag.StringVar(&configFile, "config", "", "Path to YAML config file")
	flag.IntVar(&cfg.vmCount, "vm-count", 50, "Number of VMs to simulate")
	flag.DurationVar(&cfg.duration, "duration", 0, "Stop after this duration (0 = unbounded)")

	flag.StringVar(&cfg.payloadSize, "payload-size", "small", "Embedded payload size to use (small|avg|large)")
	flag.StringVar(&cfg.payloadFile, "payload-file", "", "Path to .pb payload fixture (overrides payload-size)")
	flag.IntVar(&cfg.numPackages, "num-packages", 0, "Generate payload with this many packages (overrides payload-size)")
	flag.IntVar(&cfg.numRepos, "num-repos", 0, "Generate payload with this many repositories (default derived from packages)")

	flag.UintVar(&cfg.port, "port", 818, "Vsock port for the relay")
	flag.IntVar(&cfg.metricsPort, "metrics-port", 9090, "Expose Prometheus metrics on this port (0 = disabled)")
	flag.DurationVar(&cfg.statsInterval, "stats-interval", 10*time.Second, "Console stats interval")
	flag.DurationVar(&cfg.requestTimeout, "request-timeout", 10*time.Second, "Per-request vsock deadline")
	flag.DurationVar(&cfg.reportInterval, "report-interval", 30*time.Second, "Interval at which each VM sends reports")
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
	if cfg.vmCount <= 0 {
		log.Fatalf("vm-count must be > 0")
	}
	if cfg.vmCount > 100000 {
		log.Fatalf("vm-count must be <= 100000")
	}
	if cfg.reportInterval <= 0 {
		log.Fatalf("report-interval must be > 0")
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

	if !setFlags["vm-count"] && yamlCfg.Loadgen.VmCount > 0 {
		cfg.vmCount = yamlCfg.Loadgen.VmCount
	}
	if !setFlags["payload-size"] && yamlCfg.Loadgen.PayloadSize != "" {
		cfg.payloadSize = yamlCfg.Loadgen.PayloadSize
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
	if !setFlags["report-interval"] && yamlCfg.Loadgen.ReportInterval != "" {
		if d, err := time.ParseDuration(yamlCfg.Loadgen.ReportInterval); err == nil {
			cfg.reportInterval = d
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
			VsockCID:        3, // Will be customized per-VM in newPayloadProvider
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

// calculateCIDRange determines the CID range for this pod based on its node's index
// in the sorted list of worker nodes. This ensures no overlap between DaemonSet pods.
// vmCountTotal is the total number of VMs across ALL nodes in the cluster.
// Returns: startCID, endCID, nodeIndex, totalNodes, vmsThisNode, error
func calculateCIDRange(ctx context.Context, nodeName string, vmCountTotal int) (startCID, endCID uint32, nodeIndex, totalNodes, vmsThisNode int, err error) {
	const (
		firstValidCID  = 3     // First valid vsock CID (0=hypervisor, 1=loopback, 2=host)
		vmsPerNodeSlot = 10000 // Max VMs per node partition (for spacing)
	)

	// Get in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("get cluster config: %w", err)
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("create clientset: %w", err)
	}

	// List all worker nodes
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{
		LabelSelector: "node-role.kubernetes.io/worker",
	})
	if err != nil {
		return 0, 0, 0, 0, 0, fmt.Errorf("list nodes: %w", err)
	}

	if len(nodes.Items) == 0 {
		return 0, 0, 0, 0, 0, fmt.Errorf("no worker nodes found")
	}

	// Sort nodes by name for deterministic ordering
	nodeNames := make([]string, 0, len(nodes.Items))
	for _, node := range nodes.Items {
		nodeNames = append(nodeNames, node.Name)
	}
	sort.Strings(nodeNames)
	totalNodes = len(nodeNames)

	// Find current node's index
	nodeIndex = -1
	for i, name := range nodeNames {
		if name == nodeName {
			nodeIndex = i
			break
		}
	}

	if nodeIndex == -1 {
		return 0, 0, 0, 0, 0, fmt.Errorf("node %s not found in worker node list", nodeName)
	}

	// Divide total VMs evenly across all nodes
	vmsPerNode := vmCountTotal / totalNodes
	remainder := vmCountTotal % totalNodes

	// Distribute remainder VMs to first nodes (0, 1, 2, ...)
	// Each pod independently evaluates this condition based on its own nodeIndex,
	// so only the first 'remainder' nodes (by sorted name) get an extra VM.
	// Example: 1000 VMs / 3 nodes = 333 per node, 1 remainder
	//   Node 0: 0 < 1 = true  → 333 + 1 = 334 VMs
	//   Node 1: 1 < 1 = false → 333 VMs
	//   Node 2: 2 < 1 = false → 333 VMs
	vmsThisNode = vmsPerNode
	if nodeIndex < remainder {
		vmsThisNode++
	}

	// Validate per-node capacity
	if vmsThisNode > vmsPerNodeSlot {
		return 0, 0, 0, 0, 0, fmt.Errorf("too many VMs per node: %d VMs/node exceeds capacity of %d (reduce vmCount or add more nodes)", vmsThisNode, vmsPerNodeSlot)
	}

	// Calculate CID range based on node index
	// Each node gets a partition of vmsPerNodeSlot to ensure no overlap
	startCID = uint32(firstValidCID) + uint32(nodeIndex*vmsPerNodeSlot)
	endCID = startCID + uint32(vmsThisNode) - 1

	// Validate CID overflow
	const maxCID = uint32(4294967295) // uint32 max
	if endCID > maxCID {
		return 0, 0, 0, 0, 0, fmt.Errorf("CID overflow: endCID %d exceeds maximum %d (too many nodes or VMs)", endCID, maxCID)
	}

	return startCID, endCID, nodeIndex, totalNodes, vmsThisNode, nil
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

type payloadProvider struct {
	payloads map[uint32][]byte // CID -> pre-marshaled payload
}

func newPayloadProviderWithRange(base *v1.IndexReport, vmCount int, startCID uint32) (*payloadProvider, error) {
	endCID := startCID + uint32(vmCount) - 1

	log.Printf("pre-generating %d unique reports for CID range [%d-%d]...", vmCount, startCID, endCID)
	start := time.Now()

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	payloads := make(map[uint32][]byte)

	for i := 0; i < vmCount; i++ {
		cid := startCID + uint32(i)

		// Clone base report and customize for this CID
		report := proto.Clone(base).(*v1.IndexReport)
		report.VsockCid = fmt.Sprintf("%d", cid)

		// Apply variability to make each report unique
		applyVariability(report, i, rng)
		baseHash := report.GetIndexV4().GetHashId()
		report.IndexV4.HashId = fmt.Sprintf("%s-cid%d", baseHash, cid)

		data, err := proto.Marshal(report)
		if err != nil {
			return nil, fmt.Errorf("marshal report for CID %d: %w", cid, err)
		}
		payloads[cid] = data
	}

	log.Printf("pre-generated %d unique reports in %s", len(payloads), time.Since(start))
	return &payloadProvider{payloads: payloads}, nil
}

func applyVariability(report *v1.IndexReport, templateIdx int, rng *rand.Rand) {
	baseHash := report.GetIndexV4().GetHashId()
	report.IndexV4.HashId = fmt.Sprintf("%s-tpl%d-%d", baseHash, templateIdx, rng.Int63())

	packages := report.GetIndexV4().GetContents().GetPackages()
	// Modify a subset of packages to create variability
	modified := 0
	for _, pkg := range packages {
		pkg.Version = fmt.Sprintf("%s-var%d", pkg.GetVersion(), rng.Intn(10000))
		modified++
		if modified >= 5 {
			break
		}
	}
}

func (p *payloadProvider) Get(cid uint32) ([]byte, error) {
	payload, ok := p.payloads[cid]
	if !ok {
		return nil, fmt.Errorf("CID %d not in pre-generated range", cid)
	}
	return payload, nil
}

// vmSimulator simulates a single VM with a specific CID sending index reports periodically.
// Realistic timing behavior:
// - Random initial delay (0 to reportInterval) to stagger VM starts
// - ±5% jitter on report intervals to simulate real-world variance
func vmSimulator(ctx context.Context, cid uint32, cfg config, provider *payloadProvider, stats *statsCollector, metrics *metricsRegistry) {
	payload, err := provider.Get(cid)
	if err != nil {
		log.Printf("VM[%d]: failed to get payload: %v", cid, err)
		return
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(cid)))

	// Stagger VM starts with random initial delay [0, reportInterval)
	initialDelay := time.Duration(rng.Int63n(int64(cfg.reportInterval)))
	select {
	case <-ctx.Done():
		return
	case <-time.After(initialDelay):
	}

	// Send first report after initial delay
	sendVMReport(cid, payload, cfg.requestTimeout, stats, metrics)

	// Continue sending reports with jittered intervals
	for {
		// Add ±5% jitter to report interval
		jitter := time.Duration(float64(cfg.reportInterval) * (rng.Float64()*0.1 - 0.05))
		nextInterval := cfg.reportInterval + jitter

		select {
		case <-ctx.Done():
			return
		case <-time.After(nextInterval):
			sendVMReport(cid, payload, cfg.requestTimeout, stats, metrics)
		}
	}
}

func sendVMReport(cid uint32, payload []byte, timeout time.Duration, stats *statsCollector, metrics *metricsRegistry) {
	start := time.Now()
	err := sendReport(payload, timeout)
	latency := time.Since(start)

	if err != nil {
		log.Printf("VM[%d]: send error: %v", cid, err)
		metrics.observeFailure(errorLabel(err))
		stats.recordFailure()
		return
	}

	metrics.observeSuccess(latency, len(payload))
	stats.recordSuccess(latency, len(payload))
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
