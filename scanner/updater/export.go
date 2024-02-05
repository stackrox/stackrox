package updater

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/pkg/errors"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/jsonblob"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/enricher/nvd"
	"github.com/stackrox/rox/scanner/updater/manual"
	"github.com/stackrox/rox/scanner/updater/rhel"
	"golang.org/x/time/rate"

	// Default updaters. This is required to ensure updater factories are set properly.
	_ "github.com/quay/claircore/updater/defaults"
)

// Export is responsible for triggering the updaters to download Common Vulnerabilities and Exposures (CVEs) data
// and then outputting the result as a zstd-compressed file named vulns.json.zst.
func Export(ctx context.Context, outputDir string) error {
	err := os.MkdirAll(outputDir, 0700)
	if err != nil {
		return err
	}
	// create output json file
	outputFile, err := os.Create(filepath.Join(outputDir, "vulns.json.zst"))
	if err != nil {
		return err
	}
	defer func() {
		if err := outputFile.Close(); err != nil {
			zlog.Error(ctx).Err(err).Msg("Failed to close output file")
		}
	}()

	limiter := rate.NewLimiter(rate.Every(time.Second), 15)
	httpClient := &http.Client{
		Transport: &rateLimitedTransport{
			limiter:   limiter,
			transport: http.DefaultTransport,
		},
	}

	zstdWriter, err := zstd.NewWriter(outputFile)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := zstdWriter.Close()
		if closeErr != nil {
			zlog.Error(ctx).Err(closeErr).Msg("Failed to closing zstdWriter")
		}
	}()

	var opts [][]updates.ManagerOption

	// Manual CVEs.
	manualSet, err := manual.UpdaterSet(ctx, nil)
	if err != nil {
		return err
	}
	opts = append(opts, []updates.ManagerOption{
		// This is required to prevent default updaters from running.
		updates.WithEnabled([]string{}),
		updates.WithOutOfTree(manualSet.Updaters()),
	})

	// RHEL custom: Forked from ClairCore.
	fac, err := rhel.NewFactory(ctx, rhel.DefaultManifest)
	if err != nil {
		return err
	}
	opts = append(opts, []updates.ManagerOption{
		// This is required to prevent default updaters from running.
		updates.WithEnabled([]string{}),
		updates.WithFactories(map[string]driver.UpdaterSetFactory{
			"rhel-custom": fac,
		})})

	// NVD enricher.
	opts = append(opts, []updates.ManagerOption{
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
	})

	// ClairCore updaters.
	for _, uSet := range [][]string{
		{"oracle"},
		{"photon"},
		{"suse"},
		{"aws"},
		{"alpine"},
		{"debian"},
		{"rhcc"},
		{"ubuntu"},
		{"osv"},
	} {
		opts = append(opts, []updates.ManagerOption{updates.WithEnabled(uSet)})
	}

	for _, o := range opts {
		jsonStore, err := jsonblob.New()
		if err != nil {
			return err
		}
		mgr, err := updates.NewManager(ctx, jsonStore, updates.NewLocalLockSource(), httpClient, o...)
		if err != nil {
			return err
		}
		if err := mgr.Run(ctx); err != nil {
			return err
		}
		if err := jsonStore.Store(zstdWriter); err != nil {
			return err
		}
		if err := zstdWriter.Flush(); err != nil {
			zlog.Error(ctx).Err(err).Msg("Failed to flush zstd writer")
		}
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
