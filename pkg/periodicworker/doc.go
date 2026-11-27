// Package periodicworker provides a standardized library for timer-periodic background workers.
//
// This package wraps pkg/concurrency.Stopper and provides graceful shutdown, standard Prometheus
// metrics, and panic recovery for periodic tasks that run on a timer.
//
// # Key Features
//
//   - Graceful shutdown with blocking Stop() that waits for in-flight tasks
//   - Standard Prometheus metric envelope for all workers (runs, duration, failures, etc.)
//   - Automatic panic recovery with logging and stack traces
//   - Configurable intervals with optional jitter to avoid thundering herd
//   - Support for immediate runs on start and on-demand kicks
//   - Per-run timeouts with context cancellation
//
// # Basic Usage
//
//	worker := periodicworker.MustNew(periodicworker.Config{
//	    Name:     "example_worker",
//	    Interval: 5 * time.Minute,
//	    Task: func(ctx context.Context) error {
//	        // Do work here, respecting ctx cancellation
//	        return nil
//	    },
//	})
//	worker.Start()
//	defer worker.Stop()  // Blocks until current task completes
//
// # Metrics
//
// All workers automatically expose these Prometheus metrics labeled by worker name:
//
//   - rox_central_worker_runs_total{worker,outcome} - run count by outcome
//   - rox_central_worker_run_duration_seconds{worker} - per-run duration
//   - rox_central_worker_last_run_timestamp_seconds{worker} - last run timestamp
//   - rox_central_worker_last_success_timestamp_seconds{worker} - last success timestamp
//   - rox_central_worker_consecutive_failures{worker} - current failure streak
//   - rox_central_worker_in_progress{worker} - 1 during task execution
//   - rox_central_worker_running{worker} - 1 while worker goroutine is alive
package periodicworker
