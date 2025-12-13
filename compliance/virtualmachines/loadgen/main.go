package main

import (
	"context"
	"log"
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
		log.Fatalf("creating report generator: %v", err)
	}

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Fatalf("NODE_NAME environment variable not set")
	}

	cidInfo, err := calculateCIDRange(ctx, nodeName, cfg.vmCount)
	if err != nil {
		log.Fatalf("calculating CID range: %v", err)
	}

	log.Printf("Node %s (index %d/%d) assigned CID range [%d-%d] for %d VMs (total cluster: %d VMs)",
		nodeName, cidInfo.NodeIndex, cidInfo.TotalNodes, cidInfo.StartCID, cidInfo.EndCID, cidInfo.VMsThisNode, cfg.vmCount)

	payloads, err := newPayloadProvider(generator, cidInfo.VMsThisNode, cidInfo.StartCID)
	if err != nil {
		log.Fatalf("creating payload provider: %v", err)
	}

	stats := newStatsCollector()
	metrics := newMetricsRegistry()
	errorLimiter := newErrorLogLimiter(cfg.statsInterval / 10)

	if cfg.metricsPort > 0 {
		go serveMetrics(ctx, cfg.metricsPort)
	}

	var wg sync.WaitGroup
	for i := 0; i < cidInfo.VMsThisNode; i++ {
		cid := cidInfo.StartCID + uint32(i)
		wg.Add(1)
		go func(vmCID uint32) {
			defer wg.Done()
			vmSimulator(ctx, vmCID, cfg, payloads, stats, metrics, errorLimiter)
		}(cid)
	}

	log.Printf("vsock-loadgen starting: vms=%d report-interval=%s duration=%s packages=%d repos=%d cid-range=[%d-%d] port=%d",
		cidInfo.VMsThisNode, cfg.reportInterval, cfg.duration, cfg.numPackages, cfg.numRepositories, cidInfo.StartCID, cidInfo.EndCID, cfg.port)

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
		log.Printf("received shutdown signal, stopping...")
		cancel()
	}()
}
