package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/mdlayher/vsock"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// simulateVM simulates a single VM sending index reports periodically.
// It staggers the initial delay and adds jitter to report intervals.
func simulateVM(ctx context.Context, cid uint32, cfg config, provider *payloadProvider, stats *statsCollector, metrics *metricsRegistry) {
	payload, err := provider.get(cid)
	if err != nil {
		log.Errorf("VM[%d]: failed to get payload: %v", cid, err)
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

	sendVMReport(cid, payload, cfg.requestTimeout, stats, metrics)

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
	stats.recordSuccess(latency, len(payload))
}

func sendReport(payload []byte, timeout time.Duration) error {
	port := env.VirtualMachinesVsockPort.IntegerSetting()
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
