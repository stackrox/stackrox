package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	pkgauthn "github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/scannerv4/client"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/cmd/scannerctl/authn"
	"github.com/stackrox/rox/scanner/cmd/scannerctl/fixtures"
	"github.com/stackrox/rox/scanner/indexer"
)

var (
	scanTimeout = env.ScanTimeout.DurationSetting()

	testRunStart = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scannerctl_scale_test_run_start",
		Help: "Marks the start of a test run",
	}, []string{"total_workers", "total_images", "indexer_state"})

	testRunEnd = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scannerctl_scale_test_run_end",
		Help: "Marks the end of a test run",
	}, []string{"total_workers", "total_images", "indexer_state"})

	testTimeMillis = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "scannerctl_scale_test_time_millis",
		Help:    "Time to execute one test case",
		Buckets: prometheus.ExponentialBuckets(1000, 1.6, 13),
	}, []string{"worker_id", "total_workers", "total_images", "indexer_state", "error"})

	indexDurationMillis = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "scannerctl_scale_index_duration_millis",
		Help:    "Time to perform indexing per image",
		Buckets: prometheus.ExponentialBuckets(1000, 1.6, 13),
	}, []string{"worker_id", "total_workers", "total_images", "indexer_state", "error"})

	matchDurationMillis = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "scannerctl_scale_match_duration_millis",
		Help:    "Time to perform matching per image",
		Buckets: prometheus.ExponentialBuckets(1000, 1.6, 13),
	}, []string{"worker_id", "total_workers", "total_images", "indexer_state", "error"})

	registryLatencyMillis = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "scannerctl_scale_registry_duration_millis",
		Help:    "Time to contacting registries",
		Buckets: prometheus.ExponentialBuckets(1000, 1.6, 13),
	}, []string{"worker_id", "total_workers", "total_images", "indexer_state", "error"})
)

// scaleStats specifies the stats we want to track when performing scale tests.
type scaleStats struct {
	preFailure atomic.Int64

	indexSuccess atomic.Int64
	indexFailure atomic.Int64

	matchSuccess atomic.Int64
	matchFailure atomic.Int64
}

func (s *scaleStats) String() string {
	var ret strings.Builder
	ret.WriteRune('\n')
	ret.WriteString(fmt.Sprintf("pre-scanning failure: %d\n", s.preFailure.Load()))
	ret.WriteString(fmt.Sprintf("index success: %d\n", s.indexSuccess.Load()))
	ret.WriteString(fmt.Sprintf("index failure: %d\n", s.indexFailure.Load()))
	ret.WriteString(fmt.Sprintf("match success: %d\n", s.matchSuccess.Load()))
	ret.WriteString(fmt.Sprintf("match failure: %d\n", s.matchFailure.Load()))
	ret.WriteRune('\n')
	return ret.String()
}

// scaleCmd creates the scale command.
func scaleCmd(ctx context.Context) *cobra.Command {
	cmd := cobra.Command{
		Use:   "scale [OPTIONS]",
		Short: "Perform scale tests via querying for the first N images in the given repository.",
	}

	flags := cmd.PersistentFlags()
	basicAuth := flags.String(
		"auth",
		"",
		fmt.Sprintf("Use the specified basic auth credentials (warning: debug "+
			"only and unsafe, use env var %s).", authn.BasicAuthSetting))
	repository := flags.String(
		"repository",
		"",
		"Specify the repository from which to pull images (ex: quay.io/stackrox-io/scanner-v4)")
	images := flags.Int(
		"images",
		1000,
		"Specify the number of images from the given repository to scan (only used when repository is set)")
	workers := flags.Int(
		"workers",
		15,
		"Specify the number of parallel scans")
	indexOnly := flags.Bool(
		"index-only",
		false,
		"Only index the specified image")

	indexerCacheState := flags.String("indexer-state", "cold",
		"Tag the Indexer cache state (ex: warm if manifests were pre-scanned, or cold if not).")

	pprofDir := flags.String(
		"pprof-dir",
		"",
		"Enable pprof scraping from both the Matcher and Indexer services (accessible via "+
			"port :9443) by specifying a target directory where the scraped pprof profiling data—collected "+
			"from both components—will be stored for analysis or debugging.")

	metricsEnabled := flags.Bool(
		"metrics",
		false,
		"Enable a Prometheus metrics endpoint (available on port :9443) that exposes runtime metrics "+
			"related to client execution—such as request latency and error rates—by instrumenting and wrapping "+
			"API calls made to the Scanner Client.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		switch *indexerCacheState {
		case "cold", "warm":
		default:
			return fmt.Errorf("unknown indexer cache state: %s", *indexerCacheState)
		}
		flags.VisitAll(func(f *pflag.Flag) {
			log.Printf("%s=%s", f.Name, f.Value.String())
		})

		// Extract basic auth username and password.
		auth, err := authn.ParseBasic(*basicAuth)
		if err != nil {
			return err
		}

		if *metricsEnabled {
			// Open the metrics endpoint before anything else, so scrapping will get data.
			go func() {
				http.Handle("/metrics", promhttp.Handler())
				log.Fatal(http.ListenAndServe(":9090", nil))
			}()
		}

		testRunStart.
			WithLabelValues(
				fmt.Sprintf("%d", *workers),
				fmt.Sprintf("%d", *images),
				*indexerCacheState).
			SetToCurrentTime()

		// Create scanner client.
		scanner, err := factory.Create(ctx)
		if err != nil {
			return fmt.Errorf("creating client: %w", err)
		}

		var refs []name.Reference
		if *repository != "" {
			refs, err = references(ctx, auth, *repository, *images)
		} else {
			refs, err = fixtures.References()
			refs = refs[:*images]
		}
		if err != nil {
			return fmt.Errorf("fetching image references: %w", err)
		}

		log.Printf("scale testing with %d images with timeout %v (can be changed with %s)",
			len(refs), scanTimeout, env.ScanTimeout.EnvVar())

		refsC := make(chan name.Reference)
		go func() {
			for _, ref := range refs {
				refsC <- ref
			}
			close(refsC)
		}()

		indexerAddr, err := cmd.Flags().GetString("indexer-address")
		if err != nil {
			log.Fatalf("getting indexer-address: %v", err)
		}

		matcherAddr, err := cmd.Flags().GetString("matcher-address")
		if err != nil {
			log.Fatalf("getting matcher-address: %v", err)
		}

		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}

		if *pprofDir != "" {
			pprofStopC := make(chan any)
			defer close(pprofStopC)

			log.Printf("pprof dir: %s", *pprofDir)
			go profileForever("indexer", indexerAddr, httpClient, *pprofDir, pprofStopC)
			go profileForever("matcher", matcherAddr, httpClient, *pprofDir, pprofStopC)
		}

		var stats scaleStats
		var wg sync.WaitGroup
		for i := 0; i < *workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for ref := range refsC {
					start := time.Now()
					d, err := indexer.GetDigestFromReference(ref, auth)

					registryLatencyMillis.
						WithLabelValues(
							fmt.Sprintf("%d", i),
							fmt.Sprintf("%d", *workers),
							fmt.Sprintf("%d", *images),
							*indexerCacheState,
							strconv.FormatBool(err != nil)).
						Observe(float64(time.Since(start).Milliseconds()))

					if err != nil {
						stats.preFailure.Add(1)
						log.Printf("could not get digest for image %v: %v", ref, err)
						continue
					}

					start = time.Now()
					err = doWithTimeout(ctx, scanTimeout, func(ctx context.Context) error {
						log.Printf("indexing image %v", ref)
						// TODO(ROX-23898): add flag for skipping TLS verification.
						opt := client.ImageRegistryOpt{InsecureSkipTLSVerify: false}
						indexStart := time.Now()
						_, err := scanner.GetOrCreateImageIndex(ctx, d, auth, opt)
						indexDurationMillis.WithLabelValues(
							fmt.Sprintf("%d", i),
							fmt.Sprintf("%d", *workers),
							fmt.Sprintf("%d", *images),
							*indexerCacheState,
							strconv.FormatBool(err != nil),
						).Observe(float64(time.Since(indexStart).Milliseconds()))
						if err != nil {
							stats.indexFailure.Add(1)
							return fmt.Errorf("indexing: %w", err)
						}
						stats.indexSuccess.Add(1)

						if *indexOnly {
							return nil
						}

						log.Printf("matching image %v", ref)
						// Though this method both indexes and matches, we know the indexing has already completed,
						// and this method will just verify the index still exists. We don't account for
						// this verification's potential failures at this time.
						// TODO(ROX-23898): add flag for skipping TLS verification.
						matchStart := time.Now()
						_, err = scanner.IndexAndScanImage(ctx, d, auth, opt)
						matchDurationMillis.WithLabelValues(
							fmt.Sprintf("%d", i),
							fmt.Sprintf("%d", *workers),
							fmt.Sprintf("%d", *images),
							*indexerCacheState,
							strconv.FormatBool(err != nil),
						).Observe(float64(time.Since(matchStart).Milliseconds()))
						if err != nil {
							stats.matchFailure.Add(1)
							return fmt.Errorf("matching: %w", err)
						}
						stats.matchSuccess.Add(1)

						return nil
					})

					testTimeMillis.
						WithLabelValues(fmt.Sprintf("%d", i),
							fmt.Sprintf("%d", *workers),
							fmt.Sprintf("%d", len(refs)),
							*indexerCacheState,
							strconv.FormatBool(err != nil)).
						Observe(float64(time.Since(start).Milliseconds()))

					if err != nil {
						log.Printf("error scanning image %v: %v", ref, err)
					}
				}
			}()
		}

		wg.Wait()
		testRunEnd.
			WithLabelValues(
				fmt.Sprintf("%d", *workers),
				fmt.Sprintf("%d", *images),
				*indexerCacheState).
			SetToCurrentTime()

		log.Printf("scale tests complete: %v", &stats)

		return nil
	}
	return &cmd
}

func references(ctx context.Context, auth pkgauthn.Authenticator, repository string, n int) ([]name.Reference, error) {
	repo, err := name.NewRepository(repository, name.StrictValidation)
	if err != nil {
		return nil, fmt.Errorf("validating repository: %w", err)
	}
	puller, err := remote.NewPuller(remote.WithAuth(auth))
	if err != nil {
		return nil, fmt.Errorf("creating puller: %w", err)
	}
	lister, err := puller.Lister(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("creating lister: %w", err)
	}

	refs := make([]name.Reference, 0, n)
ListTags:
	for lister.HasNext() {
		ts, err := lister.Next(ctx)
		if err != nil {
			return nil, fmt.Errorf("listing tags: %w", err)
		}

		for _, tag := range ts.Tags {
			ref, err := name.ParseReference(repository+":"+tag, name.StrictValidation)
			if err != nil {
				return nil, err
			}

			refs = append(refs, ref)

			if len(refs) == cap(refs) {
				break ListTags
			}
		}
	}

	return refs, nil
}

func doWithTimeout(ctx context.Context, timeout time.Duration, f func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return f(ctx)
}

// profileForever queries the scanner at the given endpoint with the given client
// and saves the contents in the given directory.
// The stopC channel signals the profiler should terminate gracefully.
//
// This function writes to the stopC channel to indicate when it has terminated gracefully.
func profileForever(service, svcAddr string, cli *http.Client, dir string, stopC chan any) {
	// Replace whatever port with the pprof default port.
	parts := strings.SplitN(svcAddr, ":", 1)
	pprofAddr := fmt.Sprintf("%s:%s", parts[0], ":9443")

	log.Printf("profiling %s: %s", service, pprofAddr)
	heapReq, heapErr := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s/debug/heap", pprofAddr), nil)
	cpuReq, cpuErr := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s/debug/pprof/profile", pprofAddr), nil)
	goroutineReq, goroutineErr := http.NewRequest(http.MethodGet, fmt.Sprintf("https://%s/debug/goroutine", pprofAddr), nil)
	utils.CrashOnError(heapErr, cpuErr, goroutineErr)

	// Representation of: Mon Jan 2 15:04:05 -0700 MST 2006
	layout := "2006-01-02-15-04-05"
	for {
		select {
		case <-stopC:
			return
		default:
		}

		heapResp, heapErr := cli.Do(heapReq)
		cpuResp, cpuErr := cli.Do(cpuReq)
		goroutineResp, goroutineErr := cli.Do(goroutineReq)
		if heapErr != nil || cpuErr != nil || goroutineErr != nil {
			log.Fatalf("unable to get profile(s) from %s: heapErr=%v, cpuErr=%v, goroutineErr=%v",
				service, heapErr, cpuErr, goroutineErr)
		}

		now := time.Now()
		heapF, heapErr := os.Create(fmt.Sprintf("%s/%s.heap_%s.tar.gz", dir, service, now.Format(layout)))
		cpuF, cpuErr := os.Create(fmt.Sprintf("%s/%s.cpu_%s.tar.gz", dir, service, now.Format(layout)))
		goroutineF, goroutineErr := os.Create(fmt.Sprintf("%s/%s.goroutine_%s.tar.gz", dir, service, now.Format(layout)))
		utils.CrashOnError(heapErr, cpuErr, goroutineErr)

		_, heapErr = io.Copy(heapF, heapResp.Body)
		_, cpuErr = io.Copy(cpuF, cpuResp.Body)
		_, goroutineErr = io.Copy(goroutineF, goroutineResp.Body)
		utils.CrashOnError(heapErr, cpuErr, goroutineErr)

		utils.IgnoreError(heapF.Close)
		utils.IgnoreError(cpuF.Close)
		utils.IgnoreError(goroutineF.Close)

		time.Sleep(30 * time.Second)
	}
}
