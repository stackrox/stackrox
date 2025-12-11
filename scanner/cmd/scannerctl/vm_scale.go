package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/fixtures/vmindexreport"
)

var (
	vmScaleMatchDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "scannerctl_vm_scale_match_duration_seconds",
		Help:    "Time to perform GetVulnerabilities per VM index report",
		Buckets: prometheus.ExponentialBuckets(0.5, 2, 10), // 0.5s to 256s
	}, []string{"worker_id", "error"})

	vmScaleTotalRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "scannerctl_vm_scale_total_requests",
		Help: "Total number of GetVulnerabilities requests",
	}, []string{"status"})
)

// vmScaleStats tracks stats for VM scale testing
type vmScaleStats struct {
	success atomic.Int64
	failure atomic.Int64
}

func (s *vmScaleStats) String() string {
	return fmt.Sprintf("success: %d, failure: %d", s.success.Load(), s.failure.Load())
}

// vmScaleCmd creates the VM scale test command
func vmScaleCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "vm-scale",
		Short: "Perform scale tests by sending VM index reports directly to Scanner V4 Matcher",
		Long: `Send GetVulnerabilities requests with synthetic VM index reports to Scanner V4 Matcher.

This bypasses Central and Sensor, allowing isolated testing of Scanner V4 performance.

Example:
  # Send 100 requests with 15 workers, 2000 packages per report
  scannerctl vm-scale --requests 100 --workers 15 --packages 2000

  # Sustain 3 requests/second for 60 seconds
  scannerctl vm-scale --rate 3 --duration 60s --packages 2000
`,
	}

	flags := cmd.PersistentFlags()
	numRequests := flags.Int("requests", 100, "Total number of requests to send (ignored if --duration is set)")
	numWorkers := flags.Int("workers", 15, "Number of parallel workers")
	numPackages := flags.Int("packages", 2000, "Number of packages per VM index report")
	numRepos := flags.Int("repos", 10, "Number of repositories per VM index report")
	rateLimit := flags.Float64("rate", 0, "Target requests per second (0 = unlimited)")
	duration := flags.Duration("duration", 0, "Run for this duration (0 = run until --requests completed)")
	verbose := flags.Bool("verbose", false, "Print each request result")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		log.Printf("VM Scale Test Configuration:")
		log.Printf("  Workers: %d", *numWorkers)
		log.Printf("  Packages per report: %d", *numPackages)
		log.Printf("  Repositories: %d", *numRepos)
		if *duration > 0 {
			log.Printf("  Duration: %v", *duration)
			log.Printf("  Rate limit: %.2f req/s", *rateLimit)
		} else {
			log.Printf("  Total requests: %d", *numRequests)
		}

		// Create scanner client
		scanner, err := factory.Create(ctx)
		if err != nil {
			return fmt.Errorf("creating scanner client: %w", err)
		}

		// Generate the index report template (reused for all requests)
		// Uses shared pkg/fixtures/vmindexreport package
		gen := vmindexreport.NewGenerator(*numPackages, *numRepos)
		indexReport := gen.GenerateV4IndexReport()
		log.Printf("Generated index report with %d packages, %d repos",
			gen.NumPackages(), gen.NumRepositories())

		digest, err := name.NewDigest(vmindexreport.MockDigestWithRegistry)
		if err != nil {
			return fmt.Errorf("parsing digest: %w", err)
		}

		var stats vmScaleStats
		var wg sync.WaitGroup

		// Channel for work distribution
		workC := make(chan int, *numWorkers*2)

		// Rate limiter (if enabled)
		var rateLimiter <-chan time.Time
		if *rateLimit > 0 {
			interval := time.Duration(float64(time.Second) / *rateLimit)
			rateLimiter = time.Tick(interval)
		}

		// Start workers
		for i := 0; i < *numWorkers; i++ {
			wg.Add(1)
			workerID := i
			go func() {
				defer wg.Done()
				for reqID := range workC {
					if rateLimiter != nil {
						<-rateLimiter
					}

					start := time.Now()
					_, err := scanner.GetVulnerabilities(ctx, digest, indexReport.GetContents())
					elapsed := time.Since(start)

					errStr := "false"
					if err != nil {
						errStr = "true"
						stats.failure.Add(1)
						vmScaleTotalRequests.WithLabelValues("error").Inc()
						if *verbose {
							log.Printf("[worker-%d] req=%d FAILED (%.2fs): %v", workerID, reqID, elapsed.Seconds(), err)
						}
					} else {
						stats.success.Add(1)
						vmScaleTotalRequests.WithLabelValues("success").Inc()
						if *verbose {
							log.Printf("[worker-%d] req=%d OK (%.2fs)", workerID, reqID, elapsed.Seconds())
						}
					}

					vmScaleMatchDuration.WithLabelValues(
						fmt.Sprintf("%d", workerID),
						errStr,
					).Observe(elapsed.Seconds())
				}
			}()
		}

		// Send work
		startTime := time.Now()
		if *duration > 0 {
			// Duration-based mode
			deadline := time.After(*duration)
			reqID := 0
		durationLoop:
			for {
				select {
				case <-deadline:
					break durationLoop
				case workC <- reqID:
					reqID++
				}
			}
		} else {
			// Request-count mode
			for i := 0; i < *numRequests; i++ {
				workC <- i
			}
		}
		close(workC)

		// Wait for all workers to finish
		wg.Wait()
		totalTime := time.Since(startTime)

		totalReqs := stats.success.Load() + stats.failure.Load()
		log.Printf("\n=== VM Scale Test Results ===")
		log.Printf("Total time: %v", totalTime)
		log.Printf("Total requests: %d", totalReqs)
		log.Printf("Success: %d, Failure: %d", stats.success.Load(), stats.failure.Load())
		log.Printf("Throughput: %.2f req/s", float64(totalReqs)/totalTime.Seconds())

		return nil
	}

	return &cmd
}
