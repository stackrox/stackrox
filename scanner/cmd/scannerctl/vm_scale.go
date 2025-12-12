package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/fixtures/vmindexreport"
	"github.com/stackrox/rox/pkg/scannerv4/client"
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

// resolvePodIPs resolves a service address to individual pod IPs
func resolvePodIPs(address string) ([]string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", address, err)
	}

	ips, err := net.LookupHost(host)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup failed for %q: %w", host, err)
	}

	// Append port to each IP
	addresses := make([]string, len(ips))
	for i, ip := range ips {
		addresses[i] = net.JoinHostPort(ip, port)
	}
	return addresses, nil
}

// selectPackagesFromLearnedData selects package indices based on learned vulnerability data.
// Returns the selected indices and expected vulnerability count.
func selectPackagesFromLearnedData(learned *LearnedVulnData, numPackages int, targetVulns int, zeroVulnsOnly bool, vulnerableOnly bool) ([]int, int) {
	// Separate packages by vulnerability status
	var withVulns, withoutVulns []PackageVulnData
	for _, pkg := range learned.Packages {
		if pkg.Vulns < 0 {
			continue // Skip errored packages
		}
		if pkg.Vulns > 0 {
			withVulns = append(withVulns, pkg)
		} else {
			withoutVulns = append(withoutVulns, pkg)
		}
	}

	// Sort vulnerable packages by vuln count (descending) for easier selection
	sort.Slice(withVulns, func(i, j int) bool {
		return withVulns[i].Vulns > withVulns[j].Vulns
	})

	var selected []int
	expectedVulns := 0

	switch {
	case zeroVulnsOnly:
		// Only use packages with 0 vulnerabilities
		for i := 0; i < numPackages && i < len(withoutVulns); i++ {
			selected = append(selected, withoutVulns[i].Index)
		}

	case vulnerableOnly:
		// Only use packages with vulnerabilities
		for i := 0; i < numPackages && i < len(withVulns); i++ {
			selected = append(selected, withVulns[i].Index)
			expectedVulns += withVulns[i].Vulns
		}

	case targetVulns > 0:
		// Select packages to reach target vulnerability count
		// Strategy: add vulnerable packages until we reach target, fill rest with non-vulnerable
		for _, pkg := range withVulns {
			if expectedVulns >= targetVulns {
				break
			}
			selected = append(selected, pkg.Index)
			expectedVulns += pkg.Vulns
		}
		// Fill remaining slots with non-vulnerable packages
		for i := 0; len(selected) < numPackages && i < len(withoutVulns); i++ {
			selected = append(selected, withoutVulns[i].Index)
		}

	default:
		// Default: use first N packages (original behavior)
		for i := 0; i < numPackages && i < len(learned.Packages); i++ {
			if learned.Packages[i].Vulns >= 0 {
				selected = append(selected, learned.Packages[i].Index)
				expectedVulns += learned.Packages[i].Vulns
			}
		}
	}

	return selected, expectedVulns
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

  # Use learned vulnerability data to control vulnerability count
  scannerctl vm-learn --output vulns.json
  scannerctl vm-scale --vuln-data vulns.json --target-vulns 100 --packages 50
  scannerctl vm-scale --vuln-data vulns.json --zero-vulns --packages 500
`,
	}

	flags := cmd.PersistentFlags()
	numRequests := flags.Int("requests", 100, "Total number of requests to send (ignored if --duration is set)")
	numWorkers := flags.Int("workers", 15, "Number of parallel workers")
	numPackages := flags.Int("packages", 2000, "Number of packages per VM index report")
	rateLimit := flags.Float64("rate", 0, "Target requests per second (0 = unlimited)")
	duration := flags.Duration("duration", 0, "Run for this duration (0 = run until --requests completed)")
	verbose := flags.Bool("verbose", false, "Print each request result")
	directPodIPs := flags.Bool("direct-pod-ips", false, "Resolve service DNS and connect directly to pod IPs (distributes load)")

	// Learned data options
	vulnDataFile := flags.String("vuln-data", "", "Path to learned vulnerability data (from vm-learn command)")
	targetVulns := flags.Int("target-vulns", 0, "Target number of vulnerabilities (requires --vuln-data)")
	zeroVulnsOnly := flags.Bool("zero-vulns", false, "Use only packages with 0 vulnerabilities (requires --vuln-data)")
	vulnerableOnly := flags.Bool("vulnerable-only", false, "Use only packages with vulnerabilities (requires --vuln-data)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		matcherAddr, _ := cmd.Flags().GetString("matcher-address")

		// Resolve pod IPs if direct mode is enabled
		var podAddresses []string
		if *directPodIPs && matcherAddr != "" {
			var err error
			podAddresses, err = resolvePodIPs(matcherAddr)
			if err != nil {
				return fmt.Errorf("resolving pod IPs: %w", err)
			}
			log.Printf("Resolved %d pod IPs: %v", len(podAddresses), podAddresses)
		}

		log.Printf("VM Scale Test Configuration:")
		log.Printf("  Workers: %d", *numWorkers)
		log.Printf("  Packages per report: %d", *numPackages)
		log.Printf("  Direct pod IPs: %v", *directPodIPs)
		if *vulnDataFile != "" {
			log.Printf("  Vuln data file: %s", *vulnDataFile)
			if *zeroVulnsOnly {
				log.Printf("  Mode: zero-vulns only")
			} else if *vulnerableOnly {
				log.Printf("  Mode: vulnerable-only")
			} else if *targetVulns > 0 {
				log.Printf("  Target vulns: %d", *targetVulns)
			}
		}
		if *duration > 0 {
			log.Printf("  Duration: %v", *duration)
			log.Printf("  Rate limit: %.2f req/s", *rateLimit)
		} else {
			log.Printf("  Total requests: %d", *numRequests)
		}

		// Generate the index report template (reused for all requests)
		var indexReport *v4.IndexReport
		var expectedVulns int

		if *vulnDataFile != "" {
			// Load learned data and select packages
			learned, err := LoadLearnedData(*vulnDataFile)
			if err != nil {
				return fmt.Errorf("loading vuln data: %w", err)
			}
			log.Printf("Loaded vulnerability data for %d packages (learned at %s)",
				len(learned.Packages), learned.LearnedAt.Format(time.RFC3339))

			indices, vulns := selectPackagesFromLearnedData(learned, *numPackages, *targetVulns, *zeroVulnsOnly, *vulnerableOnly)
			expectedVulns = vulns

			gen := vmindexreport.NewGeneratorWithPackageIndices(indices)
			indexReport = gen.GenerateV4IndexReport()
			log.Printf("Generated index report with %d packages, %d repos (expected ~%d vulns)",
				gen.NumPackages(), gen.NumRepositories(), expectedVulns)
		} else {
			// Default behavior: use standard generator
			gen := vmindexreport.NewGeneratorWithSeed(*numPackages, 42)
			indexReport = gen.GenerateV4IndexReport()
			log.Printf("Generated index report with %d packages, %d repos",
				gen.NumPackages(), gen.NumRepositories())
		}

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

		// Start workers - each with its OWN scanner client for load balancing
		for i := 0; i < *numWorkers; i++ {
			wg.Add(1)
			workerID := i

			// If direct pod IPs mode, assign each worker to a specific pod (round-robin)
			var workerPodAddr string
			if len(podAddresses) > 0 {
				workerPodAddr = podAddresses[workerID%len(podAddresses)]
			}

			go func() {
				defer wg.Done()

				var scanner client.Scanner
				var err error

				if workerPodAddr != "" {
					// Connect directly to assigned pod IP
					log.Printf("[worker-%d] connecting to pod %s", workerID, workerPodAddr)
					scanner, err = client.NewGRPCScanner(ctx,
						client.WithMatcherAddress(workerPodAddr),
						client.SkipTLSVerification,
					)
				} else {
					// Use factory (connects to service address)
					scanner, err = factory.Create(ctx)
				}
				if err != nil {
					log.Printf("[worker-%d] failed to create scanner client: %v", workerID, err)
					return
				}

				for reqID := range workC {
					if rateLimiter != nil {
						<-rateLimiter
					}

					start := time.Now()
					vulnReport, err := scanner.GetVulnerabilities(ctx, digest, indexReport.GetContents())
					elapsed := time.Since(start)

					errStr := "false"
					podInfo := ""
					if workerPodAddr != "" {
						podInfo = fmt.Sprintf(" [pod=%s]", workerPodAddr)
					}
					if err != nil {
						errStr = "true"
						stats.failure.Add(1)
						vmScaleTotalRequests.WithLabelValues("error").Inc()
						if *verbose {
							log.Printf("[worker-%d]%s req=%d FAILED (%.2fs): %v", workerID, podInfo, reqID, elapsed.Seconds(), err)
						}
					} else {
						stats.success.Add(1)
						vmScaleTotalRequests.WithLabelValues("success").Inc()
						vulnCount := 0
						if vulnReport != nil {
							vulnCount = len(vulnReport.GetVulnerabilities())
						}
						if *verbose {
							log.Printf("[worker-%d]%s req=%d OK (%.2fs) vulns=%d", workerID, podInfo, reqID, elapsed.Seconds(), vulnCount)
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
