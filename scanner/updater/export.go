package updater

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/jsonblob"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/scanner/updater/manual"
	"golang.org/x/time/rate"

	// default updaters
	_ "github.com/quay/claircore/updater/defaults"
)

// Export is responsible for triggering the updaters to download Common Vulnerabilities and Exposures (CVEs) data
// and then outputting the result as a zstd-compressed file with .ztd extension
func Export(ctx context.Context, outputDir string) error {
	err := os.MkdirAll(outputDir, 0700)
	if err != nil {
		return err
	}
	// create output json file
	outputFile, err := os.Create(filepath.Join(outputDir, "output.json.zst"))
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

	manualSet, err := manual.UpdaterSet(ctx, nil)
	if err != nil {
		return err
	}

	outOfTree := make([]driver.Updater, 0)
	outOfTree = append(outOfTree, manualSet.Updaters()...)

	for i, uSet := range [][]string{
		{"oracle"}, {"photon"},
		{"suse"}, {"aws"},
		{"alpine"}, {"rhel"},
		{"debian"}, {"rhcc"},
		{"ubuntu"}, {"osv"},
	} {
		jsonStore, err := jsonblob.New()
		if err != nil {
			return err
		}
		var mgr *updates.Manager
		if i == 0 {
			mgr, err = updates.NewManager(ctx, jsonStore, updates.NewLocalLockSource(), httpClient,
				updates.WithEnabled(uSet),
				updates.WithOutOfTree(outOfTree),
			)
		} else {
			mgr, err = updates.NewManager(ctx, jsonStore, updates.NewLocalLockSource(), httpClient,
				updates.WithEnabled(uSet),
			)
		}
		if err != nil {
			return err
		}
		if err = mgr.Run(ctx); err != nil {
			return err
		}

		if err = jsonStore.Store(zstdWriter); err != nil {
			return err
		}
		if err = zstdWriter.Flush(); err != nil {
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
