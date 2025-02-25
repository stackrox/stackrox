package updater

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/pkg/errors"
	"github.com/quay/claircore/enricher/epss"
	"github.com/quay/claircore/enricher/kev"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/jsonblob"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/enricher/csaf"
	"github.com/stackrox/rox/scanner/enricher/nvd"
	"github.com/stackrox/rox/scanner/updater/manual"
	"golang.org/x/time/rate"

	// Default updaters. This is required to ensure updater factories are set properly.
	_ "github.com/quay/claircore/updater/defaults"
)

type ExportOptions struct {
	SplitBundles  bool
	ManualVulnURL string
}

// Export is responsible for triggering the updaters to download Common Vulnerabilities and Exposures (CVEs) data.
// Depending on the export option, this will output either a single zstd file called vulns.json.zst
// or several zstd files all written to the given outputDir.
func Export(ctx context.Context, outputDir string, opts *ExportOptions) error {
	err := os.MkdirAll(outputDir, 0700)
	if err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	// Map of vulnerability bundles to their updater options.
	bundles := make(map[string][]updates.ManagerOption)

	// Our own updaters.
	bundles["manual"], err = manualOpts(ctx, opts.ManualVulnURL)
	if err != nil {
		return fmt.Errorf("initializing: manual: %w", err)
	}
	bundles["nvd"] = nvdOpts()
	bundles["epss"] = epssOpts()
	bundles["stackrox-rhel-csaf"] = redhatCSAFOpts()
	bundles["cisa-kev"] = kevOpts()

	// ClairCore updaters.
	for _, uSet := range []string{
		"alpine",
		"aws",
		"chainguard",
		"debian",
		"oracle",
		"osv",
		"photon",
		"rhel-vex",
		"suse",
		"ubuntu",
	} {
		bundles[uSet] = []updates.ManagerOption{updates.WithEnabled([]string{uSet})}
	}

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

	// The http client for pulling data from security sources.
	httpClient := &http.Client{
		Transport: &rateLimitedTransport{
			limiter:   rate.NewLimiter(rate.Every(interval), 1),
			transport: http.DefaultTransport,
		},
	}

	// Export to bundle(s).
	if opts.SplitBundles {
		for name, o := range bundles {
			ctx = zlog.ContextWithValues(ctx, "bundle", name)
			w, err := zstdWriter(filepath.Join(outputDir, fmt.Sprintf("%s.json.zst", name)))
			if err != nil {
				return err
			}
			err = bundle(ctx, httpClient, w, o)
			if err != nil {
				_ = w.Close()
				return err
			}
			if err := w.Close(); err != nil {
				// Fail to close here means the data might not have been written fully, so we
				// fail.
				return fmt.Errorf("failed to close bundle output file: %w", err)
			}
		}
	} else {
		w, err := zstdWriter(filepath.Join(outputDir, "vulns.json.zst"))
		if err != nil {
			return err
		}
		for name, o := range bundles {
			ctx = zlog.ContextWithValues(ctx, "bundle", name)
			err := bundle(ctx, httpClient, w, o)
			if err != nil {
				_ = w.Close()
				return err
			}
		}
		// Fail to close here means the data might not have been written fully, so we
		// fail.
		if err := w.Close(); err != nil {
			return fmt.Errorf("failed to close bundle output file: %w", err)
		}
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

func kevOpts() []updates.ManagerOption {
	return []updates.ManagerOption{
		// This is required to prevent default updaters from running.
		updates.WithEnabled([]string{}),
		updates.WithFactories(map[string]driver.UpdaterSetFactory{
			"clair.kev": kev.NewFactory(),
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
