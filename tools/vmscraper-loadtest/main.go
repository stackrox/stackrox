// Command vmscraper-loadtest is a PoC stress-test harness for the VSOCK
// pull-mode VMScraper. It drives a real vmscraper.VMScraper against a
// synthetic farm of in-process fake roxagent instances (real
// vsockserver.Handler/ReportCache code, connected over net.Pipe), with no
// KubeVirt, Central, or real VM involved.
//
// See docs/superpowers/specs/2026-07-03-vsock-pull-stress-test-design.md
// (Part A) for the design this implements.
//
// This is a PoC: it does not yet implement uds/tcp transports, churn,
// failure injection, the backlog gauge, or ramp mode from the design doc --
// those are follow-up iterations once this validates the basic approach.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vmscraper"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vmscraper/loadtest"
	"github.com/stackrox/rox/sensor/common/virtualmachine/vsockclient"
)

var log = logging.LoggerForModule()

func main() {
	numVMs := flag.Int("num-vms", 100, "number of synthetic VMs to simulate")
	numPackages := flag.Int("num-packages", 524, "packages per simulated VM report (524 matches the real-world RHEL fixture)")
	duration := flag.Duration("duration", time.Minute, "how long to run the load test before exiting")
	pollInterval := flag.Duration("poll-interval", 15*time.Second, "VMScraper poll interval (ROX_VIRTUAL_MACHINES_SCRAPER_POLL_INTERVAL)")
	concurrency := flag.Int("concurrency", 20, "VMScraper concurrency (ROX_VIRTUAL_MACHINES_SCRAPER_CONCURRENCY)")
	perVMTimeout := flag.Duration("per-vm-timeout", 30*time.Second, "per-VM scrape timeout (ROX_VIRTUAL_MACHINES_SCRAPER_PER_VM_TIMEOUT)")
	dialLatency := flag.Duration("dial-latency", 0, "artificial per-dial latency to inject, simulating KubeVirt/VSOCK round-trip overhead")
	rescanInterval := flag.Duration("rescan-interval", 2*time.Minute, "how often a random synthetic VM bumps its report generation (0 disables rescanning); ignored if --always-changed is set")
	alwaysChanged := flag.Bool("always-changed", false, "bump every VM's report generation on every poll, so VMScraper always takes the full-report path instead of unchanged (recommended for measuring worst-case throughput/latency)")
	metricsAddr := flag.String("metrics-addr", ":9091", "address to serve /metrics on")
	flag.Parse()

	if err := run(runConfig{
		numVMs:         *numVMs,
		numPackages:    *numPackages,
		duration:       *duration,
		pollInterval:   *pollInterval,
		concurrency:    *concurrency,
		perVMTimeout:   *perVMTimeout,
		dialLatency:    *dialLatency,
		rescanInterval: *rescanInterval,
		alwaysChanged:  *alwaysChanged,
		metricsAddr:    *metricsAddr,
	}); err != nil {
		log.Fatalf("vmscraper-loadtest: %v", err)
	}
}

type runConfig struct {
	numVMs         int
	numPackages    int
	duration       time.Duration
	pollInterval   time.Duration
	concurrency    int
	perVMTimeout   time.Duration
	dialLatency    time.Duration
	rescanInterval time.Duration
	alwaysChanged  bool
	metricsAddr    string
}

func run(cfg runConfig) error {
	// VMScraper reads its tuning from env vars at construction time (see
	// pkg/env/virtualmachine.go), so we set them here rather than plumbing
	// separate config through vmscraper.New.
	setEnvOrPanic("ROX_VIRTUAL_MACHINES_SCRAPER_POLL_INTERVAL", cfg.pollInterval.String())
	setEnvOrPanic("ROX_VIRTUAL_MACHINES_SCRAPER_CONCURRENCY", fmt.Sprintf("%d", cfg.concurrency))
	setEnvOrPanic("ROX_VIRTUAL_MACHINES_SCRAPER_PER_VM_TIMEOUT", cfg.perVMTimeout.String())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	farm := loadtest.NewFarm(cfg.numVMs, cfg.numPackages, cfg.rescanInterval, cfg.alwaysChanged)
	farm.Start(ctx)

	dialer := loadtest.NewFarmDialer(farm, cfg.dialLatency)
	sender := loadtest.NullSender{}
	client := vsockclient.NewClient([]string{vsockclient.CapabilityReportV1}, 16*1024*1024)
	scraper := vmscraper.New(farm, sender, dialer, client)

	server := startMetricsServer(cfg.metricsAddr)
	defer func() { _ = server.Close() }()

	log.Infof("vmscraper-loadtest: starting: %d VMs, poll interval %s, concurrency %d, per-VM timeout %s, dial latency %s, always_changed=%t, duration %s",
		cfg.numVMs, cfg.pollInterval, cfg.concurrency, cfg.perVMTimeout, cfg.dialLatency, cfg.alwaysChanged, cfg.duration)
	if err := scraper.Start(); err != nil {
		return fmt.Errorf("starting scraper: %w", err)
	}

	select {
	case <-time.After(cfg.duration):
		log.Infof("vmscraper-loadtest: duration elapsed, stopping")
	case <-ctx.Done():
		log.Infof("vmscraper-loadtest: interrupted, stopping")
	}
	scraper.Stop()
	printSummary(cfg)
	log.Infof("vmscraper-loadtest: full metrics still available at http://%s/metrics until this process exits", cfg.metricsAddr)
	return nil
}

func startMetricsServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	server := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("vmscraper-loadtest: metrics server: %v", err)
		}
	}()
	log.Infof("vmscraper-loadtest: serving metrics on http://%s/metrics", addr)
	return server
}

func setEnvOrPanic(key, value string) {
	if err := os.Setenv(key, value); err != nil {
		panic(fmt.Sprintf("setting %s: %v", key, err))
	}
}
