package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/quay/claircore/enricher/epss"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/jsonblob"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/claircore/rhel/vex"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/enricher/csaf"
	"github.com/stackrox/rox/scanner/enricher/nvd"
	"github.com/stackrox/rox/scanner/updater/manual"
	"golang.org/x/time/rate"

	// Default updaters. This is required to ensure updater factories are set properly.
	_ "github.com/quay/claircore/updater/defaults"
)

// UpdaterStatus represents the result of a single updater execution.
type UpdaterStatus struct {
	Name                 string    `json:"name"`
	Status               string    `json:"status"` // "success" or "failed"
	Error                string    `json:"error,omitempty"`
	LastAttempt          time.Time `json:"last_attempt"`
	LastSuccessfulUpdate *string   `json:"last_successful_update,omitempty"` // ISO8601 timestamp, enriched by Python
}

// ExportStatus contains the results of all updater executions.
type ExportStatus struct {
	Updaters []UpdaterStatus `json:"updaters"`
}

// HasFailures returns true if any updater failed.
func (s *ExportStatus) HasFailures() bool {
	for _, u := range s.Updaters {
		if u.Status == StatusFailed {
			return true
		}
	}
	return false
}

// SuccessCount returns the number of successful updaters.
func (s *ExportStatus) SuccessCount() int {
	count := 0
	for _, u := range s.Updaters {
		if u.Status == StatusSuccess {
			count++
		}
	}
	return count
}

// FailureCount returns the number of failed updaters.
func (s *ExportStatus) FailureCount() int {
	count := 0
	for _, u := range s.Updaters {
		if u.Status == StatusFailed {
			count++
		}
	}
	return count
}

const (
	rhelVexUpdaterName = "rhel-vex"

	// Status values for UpdaterStatus
	StatusSuccess = "success"
	StatusFailed  = "failed"

	// Per-updater timeout prevents one slow updater from starving the rest.
	updaterTimeout = 30 * time.Minute
)

var (
	// ccUpdaterSets represents Claircore updater sets to initialize.
	ccUpdaterSets = []string{
		"alpine",
		"aws",
		"debian",
		"oracle",
		"osv",
		"photon",
		rhelVexUpdaterName,
		"suse",
		"ubuntu",
	}
)

// BundleExporter defines the interface for exporting vulnerability bundle data.
type BundleExporter interface {
	ExportBundle(ctx context.Context, w io.Writer, opts []updates.ManagerOption) error
}

// clairCoreBundleExporter is the production implementation using ClairCore.
type clairCoreBundleExporter struct {
	httpClient *http.Client
}

// ExportBundle implements BundleExporter using the ClairCore bundle function.
func (e *clairCoreBundleExporter) ExportBundle(ctx context.Context, w io.Writer, opts []updates.ManagerOption) error {
	return bundle(ctx, e.httpClient, w, opts)
}

// NewBundleExporter creates a new production bundle exporter with the given HTTP client.
func NewBundleExporter(httpClient *http.Client) BundleExporter {
	return &clairCoreBundleExporter{httpClient: httpClient}
}

// NewDefaultBundleExporter creates a new production bundle exporter with rate-limited HTTP client.
// Rate limit is ~16 requests/second by default, configurable via STACKROX_SCANNER_V4_UPDATER_INTERVAL.
func NewDefaultBundleExporter() BundleExporter {
	// Rate limit to ~16 requests/second by default.
	interval := 62 * time.Millisecond
	configuredInterval := os.Getenv("STACKROX_SCANNER_V4_UPDATER_INTERVAL")
	if configuredInterval != "" {
		parsedInterval, err := time.ParseDuration(configuredInterval)
		switch {
		case err != nil:
			log.Printf("invalid interval, using default (%v): %v", interval, err)
		case parsedInterval < interval:
			log.Printf("interval is too small (%v): using default (%v)", parsedInterval, interval)
		default:
			interval = parsedInterval
		}
	}

	httpClient := &http.Client{
		Transport: &rateLimitedTransport{
			limiter:   rate.NewLimiter(rate.Every(interval), 1),
			transport: http.DefaultTransport,
		},
	}

	return NewBundleExporter(httpClient)
}

type ExportOptions struct {
	ManualVulnURL string
}

// Export is responsible for triggering the updaters to download Common Vulnerabilities and Exposures (CVEs) data.
// Depending on the export option, this will output either a single zstd file called vulns.json.zst
// or several zstd files all written to the given outputDir.
//
// Export supports partial failure tolerance: if some updaters fail, the successful ones
// are still written. A status.json file is written to outputDir alongside the bundle files,
// recording the success or failure status of each updater. An error is only returned if ALL
// updaters fail.
func Export(ctx context.Context, outputDir string, opts *ExportOptions, exporter BundleExporter) (*ExportStatus, error) {
	err := os.MkdirAll(outputDir, 0700)
	if err != nil {
		return nil, fmt.Errorf("creating output dir: %w", err)
	}

	// Map of vulnerability bundles to their updater options.
	bundles := make(map[string][]updates.ManagerOption)

	// Our own updaters.
	bundles["manual"], err = manualOpts(ctx, opts.ManualVulnURL)
	if err != nil {
		return nil, fmt.Errorf("initializing: manual: %w", err)
	}
	bundles["nvd"] = nvdOpts()
	bundles["epss"] = epssOpts()
	bundles["stackrox-rhel-csaf"] = redhatCSAFOpts()

	// Claircore Updaters.
	for _, uSet := range ccUpdaterSets {
		managerOpts := []updates.ManagerOption{updates.WithEnabled([]string{uSet})}
		if uSet == rhelVexUpdaterName {
			managerOpts = rhelVexOpts()
		}
		bundles[uSet] = managerOpts
	}

	// Export to bundle(s) with partial failure tolerance.
	var status ExportStatus
	for name, o := range bundles {
		now := time.Now()
		bundleCtx, cancel := context.WithTimeout(ctx, updaterTimeout)
		bundleCtx = zlog.ContextWithValues(bundleCtx, "bundle", name)
		bundlePath := filepath.Join(outputDir, fmt.Sprintf("%s.json.zst", name))

		w, err := zstdWriter(bundlePath)
		if err != nil {
			cancel()
			zlog.Error(bundleCtx).Err(err).Msg("failed to create bundle output file")
			status.Updaters = append(status.Updaters, UpdaterStatus{
				Name:        name,
				Status:      StatusFailed,
				Error:       fmt.Sprintf("create output file: %v", err),
				LastAttempt: now,
			})
			continue
		}

		err = exporter.ExportBundle(bundleCtx, w, o)
		closeErr := w.Close()
		cancel()

		// Prefer bundle error, but capture close error if bundle succeeded
		if err == nil && closeErr != nil {
			err = fmt.Errorf("close output file: %w", closeErr)
		}

		if err != nil {
			zlog.Error(bundleCtx).Err(err).Msg("bundle export failed")
			status.Updaters = append(status.Updaters, UpdaterStatus{
				Name:        name,
				Status:      StatusFailed,
				Error:       err.Error(),
				LastAttempt: now,
			})
			if removeErr := os.Remove(bundlePath); removeErr != nil && !os.IsNotExist(removeErr) {
				zlog.Warn(bundleCtx).Err(removeErr).Msg("failed to remove bundle file")
			}
			continue
		}

		// Success
		zlog.Info(bundleCtx).Msg("bundle export completed successfully")
		status.Updaters = append(status.Updaters, UpdaterStatus{
			Name:        name,
			Status:      StatusSuccess,
			LastAttempt: now,
		})
	}

	// Write status.json with all results
	if err := writeStatusFile(outputDir, &status); err != nil {
		zlog.Warn(ctx).Err(err).Msg("failed to write status.json")
	}

	// Only return error if ALL bundles failed
	if status.SuccessCount() == 0 {
		return &status, fmt.Errorf("all %d updaters failed", len(bundles))
	}

	return &status, nil
}

// writeStatusFile writes the export status to a JSON file in the output directory.
func writeStatusFile(outputDir string, status *ExportStatus) error {
	statusPath := filepath.Join(outputDir, "status.json")
	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}
	if err := os.WriteFile(statusPath, data, 0644); err != nil {
		return fmt.Errorf("write status file: %w", err)
	}
	return nil
}

func manualOpts(ctx context.Context, uri string) ([]updates.ManagerOption, error) {
	manualSet, err := manual.UpdaterSet(ctx, uri)
	if err != nil {
		return nil, err
	}
	return []updates.ManagerOption{
		// This is required to prevent default updaters from running.
		updates.WithEnabled([]string{}),
		updates.WithOutOfTree(manualSet.Updaters()),
	}, nil

}

func nvdOpts() []updates.ManagerOption {
	return []updates.ManagerOption{
		// This is required to prevent default updaters from running.
		updates.WithEnabled([]string{}),
		updates.WithFactories(map[string]driver.UpdaterSetFactory{
			"nvd": nvd.NewFactory(),
		}),
		updates.WithConfigs(map[string]driver.ConfigUnmarshaler{
			"nvd": func(i interface{}) error {
				cfg, ok := i.(*nvd.Config)
				if !ok {
					return errors.New("internal error: config assertion failed")
				}
				path := os.Getenv("STACKROX_NVD_ZIP_PATH")
				if path != "" {
					cfg.FeedPath = &path
				}
				ci := os.Getenv("STACKROX_NVD_API_CALL_INTERVAL")
				if ci != "" {
					cfg.CallInterval = &ci
				}
				key := os.Getenv("STACKROX_NVD_API_KEY")
				if key != "" {
					cfg.APIKey = &key
				}
				return nil
			},
		}),
	}
}

func epssOpts() []updates.ManagerOption {
	return []updates.ManagerOption{
		// This is required to prevent default updaters from running.
		updates.WithEnabled([]string{}),
		updates.WithFactories(map[string]driver.UpdaterSetFactory{
			"clair.epss": epss.NewFactory(),
		}),
	}
}

func rhelVexOpts() []updates.ManagerOption {
	return []updates.ManagerOption{
		updates.WithEnabled([]string{rhelVexUpdaterName}),
		updates.WithConfigs(map[string]driver.ConfigUnmarshaler{
			rhelVexUpdaterName: func(i any) error {
				ctx := zlog.ContextWithValues(context.Background(), "updater", rhelVexUpdaterName)

				// This function gets called for both the Factory and the Updater.
				// We only need to configure the Factory (which has the CompressedFileTimeout field).
				switch cfg := i.(type) {
				case *vex.FactoryConfig:
					// Configure the factory with custom timeout.
					timeout := os.Getenv("STACKROX_RHEL_VEX_COMPRESSED_FILE_TIMEOUT")
					if timeout != "" {
						parsedTimeout, err := time.ParseDuration(timeout)
						if err != nil {
							zlog.Warn(ctx).
								Err(err).
								Msg("using default STACKROX_RHEL_VEX_COMPRESSED_FILE_TIMEOUT due to invalid duration")
						} else {
							cfg.CompressedFileTimeout = claircore.Duration(parsedTimeout)
							zlog.Info(ctx).
								Str("timeout", parsedTimeout.String()).
								Msg("using compressed file timeout")
						}
					}
				case *vex.UpdaterConfig:
					// Updater config - nothing to configure here.
				default:
					return fmt.Errorf("rhel-vex: unexpected config type: %T", i)
				}
				return nil
			},
		}),
	}
}

// TODO(ROX-26672): remove this.
func redhatCSAFOpts() []updates.ManagerOption {
	return []updates.ManagerOption{
		// This is required to prevent default updaters from running.
		updates.WithEnabled([]string{}),
		updates.WithFactories(map[string]driver.UpdaterSetFactory{
			"stackrox.rhel-csaf": csaf.NewFactory(),
		}),
	}
}

func zstdWriter(filename string) (io.WriteCloser, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	w, err := zstd.NewWriter(f)
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return w, nil
}

func bundle(ctx context.Context, client *http.Client, w io.Writer, opts []updates.ManagerOption) error {
	jsonStore, err := jsonblob.New()
	if err != nil {
		return err
	}
	mgr, err := updates.NewManager(ctx, jsonStore, updates.NewLocalLockSource(), client, opts...)
	if err != nil {
		return fmt.Errorf("new manager: %w", err)
	}
	err = mgr.Run(ctx)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}
	err = jsonStore.Store(w)
	if err != nil {
		return fmt.Errorf("json store: %w", err)
	}
	return nil
}

type rateLimitedTransport struct {
	limiter   *rate.Limiter
	transport http.RoundTripper
}

func (t *rateLimitedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.limiter.Wait(req.Context()); err != nil {
		return nil, err
	}
	return t.transport.RoundTrip(req)
}
