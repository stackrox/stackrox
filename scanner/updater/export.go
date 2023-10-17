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
	outputFile, err := os.Create(filepath.Join(outputDir, "output.json.ztd"))
	if err != nil {
		return err
	}

	limiter := rate.NewLimiter(rate.Every(time.Second), 5)
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
			zlog.Error(ctx).Err(closeErr).Msg("Failed to close zstd writer")
		}
	}()

	updaterSet, err := manual.UpdaterSet(ctx, nil)
	if err != nil {
		return err
	}
	outOfTree := append(make([][]driver.Updater, 1), updaterSet.Updaters())

	for i, uSet := range [][]string{
		{"oracle", "aws", "rhcc"},
		{"alpine", "rhel", "debian"},
		{"ubuntu", "suse", "photon"},
		{"osv"},
	} {
		jsonStore, err := jsonblob.New()
		if err != nil {
			return err
		}

		options := []updates.ManagerOption{
			updates.WithEnabled(uSet),
		}
		if i < len(outOfTree) {
			options = append(options, updates.WithOutOfTree(outOfTree[i]))
		}
		updateMgr, err := updates.NewManager(ctx, jsonStore, updates.NewLocalLockSource(), httpClient, options...)
		if err != nil {
			return err
		}

		if err := updateMgr.Run(ctx); err != nil {
			return err
		}

		err = jsonStore.Store(zstdWriter)
		if err != nil {
			return err
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
