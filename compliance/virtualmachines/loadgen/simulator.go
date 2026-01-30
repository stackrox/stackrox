package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// simulateVM simulates a single VM sending index reports periodically.
// It staggers the initial delay and adds jitter to report intervals.
func simulateVM(ctx context.Context, vmCfg vmConfig, globalCfg config, provider *payloadProvider, stats *statsCollector, metrics *metricsRegistry) {
	payload, err := provider.get(vmCfg.cid)
	if err != nil {
		log.Errorf("VM[%d]: failed to get payload: %v", vmCfg.cid, err)
		return
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(vmCfg.cid)))

	// Stagger VM starts with random initial delay [0, reportInterval)
	initialDelay := time.Duration(rng.Int63n(int64(vmCfg.reportInterval)))
	select {
	case <-ctx.Done():
		return
	case <-time.After(initialDelay):
	}

	lastSend := time.Now()
	sendVMReport(vmCfg.cid, payload, vmCfg.numPackages, 0, globalCfg.port, globalCfg.requestTimeout, stats, metrics)
	lastSend = time.Now()

	for {
		// Add ±5% jitter to report interval, clamped to >= 500ms
		jitterPercent := (rng.Float64()*0.1 - 0.05) // ±5%
		jitter := time.Duration(float64(vmCfg.reportInterval) * jitterPercent)
		nextInterval := vmCfg.reportInterval + jitter
		if nextInterval < 500*time.Millisecond {
			nextInterval = 500 * time.Millisecond
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(nextInterval):
			sendStart := time.Now()
			intervalSinceLast := sendStart.Sub(lastSend)
			sendVMReport(vmCfg.cid, payload, vmCfg.numPackages, intervalSinceLast, globalCfg.port, globalCfg.requestTimeout, stats, metrics)
			lastSend = time.Now()
		}
	}
}

func sendVMReport(cid uint32, payload []byte, packages int, observedInterval time.Duration, port uint, timeout time.Duration, stats *statsCollector, metrics *metricsRegistry) {
	start := time.Now()
	err := sendReport(payload, port, timeout)
	latency := time.Since(start)

	if err != nil {
		logging.GetRateLimitedLogger().ErrorL(
			"vsock-send",
			"VM[%d]: send error: %v",
			cid, err,
		)
		metrics.observeFailure(errorLabel(err))
		stats.recordFailure()
		return
	}

	metrics.observeSuccess(latency, len(payload))
	metrics.observeReport(packages, observedInterval)
	stats.recordSuccess(latency, len(payload))
}

func sendReport(payload []byte, port uint, timeout time.Duration) error {
	conn, err := vsock.Dial(vsock.Local, uint32(port), &vsock.Config{})
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
