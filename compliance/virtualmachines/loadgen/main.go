package main

import (
	"context"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/stackrox/rox/pkg/sync"
)

func main() {
	cfg := parseConfig()

	ctx, cancel := context.WithCancel(context.Background())
	if cfg.duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, cfg.duration)
	}
	defer cancel()

	setupSignalHandler(cancel)

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Error("NODE_NAME environment variable not set")
		os.Exit(1)
	}

	cidInfo, err := calculateCIDRange(ctx, nodeName, cfg.vmCount)
	if err != nil {
		log.Errorf("calculating CID range: %v", err)
		os.Exit(1)
	}

	// Parse seed from environment variable, fallback to current time
	// Note: parseConfig() already parsed flags, so we only check env var here
	var seed int64
	if seedEnv := os.Getenv("LOADGEN_SEED"); seedEnv != "" {
		seed, err = strconv.ParseInt(seedEnv, 10, 64)
		if err != nil {
			log.Errorf("parsing LOADGEN_SEED environment variable: %v", err)
			os.Exit(1)
		}
		log.Infof("Using seed from LOADGEN_SEED: %d", seed)
	} else {
		seed = time.Now().UnixNano()
		log.Infof("No seed provided, using current time: %d", seed)
	}

	log.Infof("Node %s (index %d/%d) assigned CID range [%d-%d] for %d VMs (total cluster: %d VMs)",
		nodeName, cidInfo.NodeIndex, cidInfo.TotalNodes, cidInfo.StartCID, cidInfo.EndCID, cidInfo.VMsThisNode, cfg.vmCount)

	// Assign VM configurations using distribution sampling
	vmConfigs := assignVMConfigs(cidInfo.VMsThisNode, cidInfo.StartCID, cfg.packageDist, cfg.intervalDist, seed)

	// Precompute payloads for all VMs
	payloads, err := newPayloadProvider(vmConfigs, cfg.specificPackage)
	if err != nil {
		log.Errorf("creating payload provider: %v", err)
		os.Exit(1)
	}

	stats := newStatsCollector()
	metrics := newMetricsRegistry()

	// Compute and log distribution stats, set Prometheus gauges
	computeDistributionStats(vmConfigs, metrics)

	if cfg.metricsPort > 0 {
		go serveMetrics(ctx, cfg.metricsPort)
	}

	var wg sync.WaitGroup
	for _, vmCfg := range vmConfigs {
		wg.Add(1)
		go func(vmCfg vmConfig) {
			defer wg.Done()
			simulateVM(ctx, vmCfg, cfg, payloads, stats, metrics)
		}(vmCfg)
	}

	log.Infof("vsock-loadgen starting: vms=%d duration=%s cid-range=[%d-%d] port=%d seed=%d",
		cidInfo.VMsThisNode, cfg.duration, cidInfo.StartCID, cidInfo.EndCID, cfg.port, seed)

	start := time.Now()
	ticker := time.NewTicker(cfg.statsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			logSnapshot("final", stats.snapshot(time.Since(start)))
			return
		case <-ticker.C:
			logSnapshot("progress", stats.snapshot(time.Since(start)))
		}
	}
}

func setupSignalHandler(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info("received shutdown signal, stopping...")
		cancel()
	}()
}
