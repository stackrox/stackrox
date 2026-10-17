package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"slices"
	"sync/atomic"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/spf13/cobra"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/fixtures/vmindexreport"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	"github.com/stackrox/rox/pkg/sync"
)

// vmScaleStats tracks success and failure counts for one vm-scale run.
type vmScaleStats struct {
	success atomic.Int64
	failure atomic.Int64
}

type vmScaleConfig struct {
	matcherAddr    string
	numRequests    int
	numWorkers     int
	numPackages    int
	rateLimit      float64
	duration       time.Duration
	verbose        bool
	directPodIPs   bool
	vulnDataFile   string
	targetVulns    int
	zeroVulnsOnly  bool
	vulnerableOnly bool
}

// validateVMScaleConfig rejects flag combinations that would hang, panic, or silently ignore a selection mode.
func validateVMScaleConfig(cfg vmScaleConfig) error {
	if cfg.numWorkers <= 0 {
		return fmt.Errorf("--workers must be positive, got %d", cfg.numWorkers)
	}
	if cfg.numPackages < 0 {
		return fmt.Errorf("--packages must be non-negative, got %d", cfg.numPackages)
	}
	if cfg.numRequests < 0 {
		return fmt.Errorf("--requests must be non-negative, got %d", cfg.numRequests)
	}
	if cfg.rateLimit < 0 {
		return fmt.Errorf("--rate must be non-negative, got %v", cfg.rateLimit)
	}
	if cfg.duration < 0 {
		return fmt.Errorf("--duration must be non-negative, got %s", cfg.duration)
	}
	if cfg.rateLimit > 0 {
		interval := time.Duration(float64(time.Second) / cfg.rateLimit)
		if interval <= 0 {
			return fmt.Errorf("--rate %v is too high to schedule", cfg.rateLimit)
		}
	}
	modes := 0
	if cfg.zeroVulnsOnly {
		modes++
	}
	if cfg.vulnerableOnly {
		modes++
	}
	if cfg.targetVulns > 0 {
		modes++
	}
	if modes > 1 {
		return errors.New("--zero-vulns, --vulnerable-only, and --target-vulns are mutually exclusive")
	}
	if modes > 0 && cfg.vulnDataFile == "" {
		return errors.New("--zero-vulns, --vulnerable-only, and --target-vulns require --vuln-data")
	}
	if cfg.directPodIPs && cfg.matcherAddr == "" {
		return errors.New("--direct-pod-ips requires --matcher-address")
	}
	return nil
}

// resolvePodIPs resolves a service address to host:port targets.
// A headless Service returns one address per pod; a ClusterIP returns a single address.
func resolvePodIPs(address string) ([]string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid address %q: %w", address, err)
	}

	ips, err := net.LookupHost(host)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup failed for %q: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("DNS lookup for %q returned no addresses", host)
	}

	addresses := make([]string, len(ips))
	for i, ip := range ips {
		addresses[i] = net.JoinHostPort(ip, port)
	}
	return addresses, nil
}

// selectPackagesFromLearnedData picks fixture indices for one index report.
// It returns an error when the learned file cannot fill numPackages, so the run does not silently under-size the report.
func selectPackagesFromLearnedData(learned *LearnedVulnData, numPackages int, targetVulns int, zeroVulnsOnly bool, vulnerableOnly bool) ([]int, int, error) {
	var withVulns, withoutVulns []PackageVulnData
	for _, pkg := range learned.Packages {
		if pkg.Vulns < 0 {
			continue
		}
		if pkg.Vulns > 0 {
			withVulns = append(withVulns, pkg)
		} else {
			withoutVulns = append(withoutVulns, pkg)
		}
	}

	slices.SortFunc(withVulns, func(a, b PackageVulnData) int {
		return cmp.Compare(b.Vulns, a.Vulns)
	})

	var selected []int
	expectedVulns := 0

	switch {
	case zeroVulnsOnly:
		for i := 0; i < numPackages && i < len(withoutVulns); i++ {
			selected = append(selected, withoutVulns[i].Index)
		}

	case vulnerableOnly:
		for i := 0; i < numPackages && i < len(withVulns); i++ {
			selected = append(selected, withVulns[i].Index)
			expectedVulns += withVulns[i].Vulns
		}

	case targetVulns > 0:
		for _, pkg := range withVulns {
			if expectedVulns >= targetVulns {
				break
			}
			selected = append(selected, pkg.Index)
			expectedVulns += pkg.Vulns
		}
		for i := 0; len(selected) < numPackages && i < len(withoutVulns); i++ {
			selected = append(selected, withoutVulns[i].Index)
		}

	default:
		for _, pkg := range learned.Packages {
			if len(selected) >= numPackages {
				break
			}
			if pkg.Vulns < 0 {
				continue
			}
			selected = append(selected, pkg.Index)
			expectedVulns += pkg.Vulns
		}
	}

	if len(selected) < numPackages {
		return nil, 0, fmt.Errorf("only %d packages match the selection, fewer than --packages %d", len(selected), numPackages)
	}
	return selected, expectedVulns, nil
}

// dispatchVMScaleWork queues request ids until n have been sent, duration elapses, or ctx is canceled.
// A nil rate sends as fast as workers accept work. The first request is not rate-limited.
func dispatchVMScaleWork(ctx context.Context, workC chan<- int, n int, duration time.Duration, rate <-chan time.Time) {
	var deadline <-chan time.Time
	if duration > 0 {
		timer := time.NewTimer(duration)
		defer timer.Stop()
		deadline = timer.C
	}
	sent := 0
	for {
		if duration == 0 && sent >= n {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-deadline:
			return
		case workC <- sent:
			sent++
		}
		if rate == nil || (duration == 0 && sent >= n) {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-deadline:
			return
		case <-rate:
		}
	}
}

// dialMatcher opens a matcher client. podAddr replaces only the matcher address so the root command's TLS options still apply.
func dialMatcher(ctx context.Context, podAddr string) (client.Scanner, error) {
	if podAddr == "" {
		return factory.Create(ctx)
	}
	opts := append(slices.Clone(factory), client.WithMatcherAddress(podAddr))
	return client.NewGRPCScanner(ctx, opts...)
}

type scaleWorker struct {
	scanner client.Scanner
	podAddr string
}

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
	vulnDataFile := flags.String("vuln-data", "", "Path to learned vulnerability data (from vm-learn command)")
	targetVulns := flags.Int("target-vulns", 0, "Target number of vulnerabilities (requires --vuln-data)")
	zeroVulnsOnly := flags.Bool("zero-vulns", false, "Use only packages with 0 vulnerabilities (requires --vuln-data)")
	vulnerableOnly := flags.Bool("vulnerable-only", false, "Use only packages with vulnerabilities (requires --vuln-data)")

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		matcherAddr, err := cmd.Flags().GetString("matcher-address")
		if err != nil {
			return fmt.Errorf("getting matcher-address: %w", err)
		}
		cfg := vmScaleConfig{
			matcherAddr:    matcherAddr,
			numRequests:    *numRequests,
			numWorkers:     *numWorkers,
			numPackages:    *numPackages,
			rateLimit:      *rateLimit,
			duration:       *duration,
			verbose:        *verbose,
			directPodIPs:   *directPodIPs,
			vulnDataFile:   *vulnDataFile,
			targetVulns:    *targetVulns,
			zeroVulnsOnly:  *zeroVulnsOnly,
			vulnerableOnly: *vulnerableOnly,
		}
		if err := validateVMScaleConfig(cfg); err != nil {
			return err
		}

		var podAddresses []string
		if cfg.directPodIPs {
			podAddresses, err = resolvePodIPs(cfg.matcherAddr)
			if err != nil {
				return fmt.Errorf("resolving pod IPs: %w", err)
			}
			log.Printf("Resolved %d pod IPs: %v", len(podAddresses), podAddresses)
			if len(podAddresses) == 1 && cfg.numWorkers > 1 {
				log.Printf("DNS returned a single address; load will not spread across matcher pods. The Service must be headless.")
			}
		}

		log.Printf("VM Scale Test Configuration:")
		log.Printf("  Workers: %d", cfg.numWorkers)
		log.Printf("  Packages per report: %d", cfg.numPackages)
		log.Printf("  Direct pod IPs: %v", cfg.directPodIPs)
		if cfg.vulnDataFile != "" {
			log.Printf("  Vuln data file: %s", cfg.vulnDataFile)
			switch {
			case cfg.zeroVulnsOnly:
				log.Printf("  Mode: zero-vulns only")
			case cfg.vulnerableOnly:
				log.Printf("  Mode: vulnerable-only")
			case cfg.targetVulns > 0:
				log.Printf("  Target vulns: %d", cfg.targetVulns)
			}
		}
		if cfg.duration > 0 {
			log.Printf("  Duration: %v", cfg.duration)
			log.Printf("  Rate limit: %.2f req/s", cfg.rateLimit)
		} else {
			log.Printf("  Total requests: %d", cfg.numRequests)
		}

		indexReport, err := buildVMScaleIndexReport(cfg)
		if err != nil {
			return err
		}
		digest, err := name.NewDigest(vmindexreport.MockDigestWithRegistry)
		if err != nil {
			return fmt.Errorf("parsing digest: %w", err)
		}

		workers, err := startScaleWorkers(ctx, cfg.numWorkers, podAddresses)
		if err != nil {
			return err
		}
		defer closeScaleWorkers(workers)

		var rateLimiter <-chan time.Time
		if cfg.rateLimit > 0 {
			interval := time.Duration(float64(time.Second) / cfg.rateLimit)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			rateLimiter = ticker.C
		}

		var stats vmScaleStats
		var wg sync.WaitGroup
		workC := make(chan int, cfg.numWorkers*2)
		for i := range workers {
			worker := workers[i]
			workerID := i
			wg.Go(func() {
				for reqID := range workC {
					runVMScaleRequest(ctx, cfg.verbose, worker, digest, indexReport, workerID, reqID, &stats)
				}
			})
		}

		startTime := time.Now()
		dispatchVMScaleWork(ctx, workC, cfg.numRequests, cfg.duration, rateLimiter)
		close(workC)
		wg.Wait()
		totalTime := time.Since(startTime)

		totalReqs := stats.success.Load() + stats.failure.Load()
		log.Printf("\n=== VM Scale Test Results ===")
		log.Printf("Total time: %v", totalTime)
		log.Printf("Total requests: %d", totalReqs)
		log.Printf("Success: %d, Failure: %d", stats.success.Load(), stats.failure.Load())
		if totalTime > 0 {
			log.Printf("Throughput: %.2f req/s", float64(totalReqs)/totalTime.Seconds())
		}
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("scale test interrupted: %w", err)
		}
		if totalReqs == 0 {
			return errors.New("scale test sent no requests")
		}
		if stats.success.Load() == 0 {
			return errors.New("scale test completed with no successful requests")
		}
		return nil
	}

	return &cmd
}

func buildVMScaleIndexReport(cfg vmScaleConfig) (*v4.IndexReport, error) {
	if cfg.vulnDataFile == "" {
		gen, err := vmindexreport.NewGeneratorWithSeed(cfg.numPackages, 42)
		if err != nil {
			return nil, fmt.Errorf("creating VM index report generator: %w", err)
		}
		log.Printf("Generated index report with %d packages, %d repos", gen.NumPackages(), gen.NumRepositories())
		return gen.GenerateV4IndexReport(), nil
	}

	learned, err := LoadLearnedData(cfg.vulnDataFile)
	if err != nil {
		return nil, fmt.Errorf("loading vuln data: %w", err)
	}
	log.Printf("Loaded vulnerability data for %d packages (learned at %s)",
		len(learned.Packages), learned.LearnedAt.Format(time.RFC3339))

	indices, expectedVulns, err := selectPackagesFromLearnedData(learned, cfg.numPackages, cfg.targetVulns, cfg.zeroVulnsOnly, cfg.vulnerableOnly)
	if err != nil {
		return nil, err
	}
	gen, err := vmindexreport.NewGeneratorWithPackageIndices(indices)
	if err != nil {
		return nil, fmt.Errorf("creating VM index report generator: %w", err)
	}
	log.Printf("Generated index report with %d packages, %d repos (expected ~%d vulns)",
		gen.NumPackages(), gen.NumRepositories(), expectedVulns)
	return gen.GenerateV4IndexReport(), nil
}

// startScaleWorkers dials every worker before any request is sent, so a connection failure returns instead of leaving the sender blocked.
func startScaleWorkers(ctx context.Context, numWorkers int, podAddresses []string) ([]scaleWorker, error) {
	workers := make([]scaleWorker, 0, numWorkers)
	for i := range numWorkers {
		var podAddr string
		if len(podAddresses) > 0 {
			podAddr = podAddresses[i%len(podAddresses)]
		}
		if podAddr != "" {
			log.Printf("[worker-%d] connecting to pod %s", i, podAddr)
		}
		scanner, err := dialMatcher(ctx, podAddr)
		if err != nil {
			closeScaleWorkers(workers)
			return nil, fmt.Errorf("worker %d: creating scanner client: %w", i, err)
		}
		workers = append(workers, scaleWorker{scanner: scanner, podAddr: podAddr})
	}
	return workers, nil
}

func closeScaleWorkers(workers []scaleWorker) {
	for _, worker := range workers {
		if worker.scanner == nil {
			continue
		}
		if err := worker.scanner.Close(); err != nil {
			log.Printf("closing scanner client: %v", err)
		}
	}
}

func runVMScaleRequest(ctx context.Context, verbose bool, worker scaleWorker, digest name.Digest, indexReport *v4.IndexReport, workerID, reqID int, stats *vmScaleStats) {
	start := time.Now()
	vulnReport, err := worker.scanner.GetVulnerabilities(ctx, digest, indexReport.GetContents())
	elapsed := time.Since(start)

	podInfo := ""
	if worker.podAddr != "" {
		podInfo = fmt.Sprintf(" [pod=%s]", worker.podAddr)
	}
	if err != nil {
		stats.failure.Add(1)
		if verbose {
			log.Printf("[worker-%d]%s req=%d FAILED (%.2fs): %v", workerID, podInfo, reqID, elapsed.Seconds(), err)
		}
		return
	}
	stats.success.Add(1)
	if verbose {
		vulnCount := 0
		if vulnReport != nil {
			vulnCount = len(vulnReport.GetVulnerabilities())
		}
		log.Printf("[worker-%d]%s req=%d OK (%.2fs) vulns=%d", workerID, podInfo, reqID, elapsed.Seconds(), vulnCount)
	}
}
