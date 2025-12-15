package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

func main() {
	cfg := parseConfig()

	ctx, cancel := context.WithCancel(context.Background())
	if cfg.duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, cfg.duration)
	}
	defer cancel()

	setupSignalHandler(cancel)

	generator, err := createReportGenerator(cfg)
	if err != nil {
		log.Errorf("creating report generator: %v", err)
		os.Exit(1)
	}

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

	log.Infof("Node %s (index %d/%d) assigned CID range [%d-%d] for %d VMs (total cluster: %d VMs)",
		nodeName, cidInfo.NodeIndex, cidInfo.TotalNodes, cidInfo.StartCID, cidInfo.EndCID, cidInfo.VMsThisNode, cfg.vmCount)

	payloads, err := newPayloadProvider(generator, cidInfo.VMsThisNode, cidInfo.StartCID)
	if err != nil {
		log.Errorf("creating payload provider: %v", err)
		os.Exit(1)
	}

	stats := newStatsCollector()
	metrics := newMetricsRegistry()

	if cfg.metricsPort > 0 {
		go serveMetrics(ctx, cfg.metricsPort)
	}

	var wg sync.WaitGroup
	for i := 0; i < cidInfo.VMsThisNode; i++ {
		cid := cidInfo.StartCID + uint32(i)
		wg.Add(1)
		go func(vmCID uint32) {
			defer wg.Done()
			vmSimulator(ctx, vmCID, cfg, payloads, stats, metrics)
		}(cid)
	}

	log.Infof("vsock-loadgen starting: vms=%d report-interval=%s duration=%s packages=%d cid-range=[%d-%d] port=%d",
		cidInfo.VMsThisNode, cfg.reportInterval, cfg.duration, cfg.numPackages, cidInfo.StartCID, cidInfo.EndCID, cfg.port)

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
