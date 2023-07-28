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

	_ "github.com/quay/claircore/updater/defaults"
)

// Export is responsible for triggering the updaters to download Common Vulnerabilities and Exposures (CVEs) data
// and then outputting the result as a zstd-compressed file with .ztd extension
func Export(ctx context.Context, outputDir string) error {
	var outOfTree []driver.Updater

	// Append updater sets directly to the outOfTree
	appendUpdaterSet := func(updaterSet driver.UpdaterSet, err error) {
		if err != nil {
			zlog.Error(ctx).Err(err).Send()
			return
		}
		outOfTree = append(outOfTree, updaterSet.Updaters()...)
	}

	appendUpdaterSet(manual.UpdaterSet(ctx, nil))
	err := os.MkdirAll(outputDir, 0700)
	if err != nil {
		return err
	}

	// create temp file
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
	defer zstdWriter.Close()

	jsonStore1, err := jsonblob.New()
	updateMgr1, err := updates.NewManager(ctx, jsonStore1, updates.NewLocalLockSource(), httpClient,
		updates.WithEnabled([]string{"oracle", "photon", "suse", "aws", "rhcc"}),
		updates.WithOutOfTree(outOfTree),
	)
	if err != nil {
		return err
	}
	if err := updateMgr1.Run(ctx); err != nil {
		return err
	}
	err = jsonStore1.Store(zstdWriter)
	if err != nil {
		return err
	}

	jsonStore2, err := jsonblob.New()
	updateMgr2, err := updates.NewManager(ctx, jsonStore2, updates.NewLocalLockSource(), httpClient,
		updates.WithEnabled([]string{"alpine", "rhel", "ubuntu", "osv", "debian"}),
	)
	if err != nil {
		return err
	}
	if err := updateMgr2.Run(ctx); err != nil {
		return err
	}
	err = jsonStore2.Store(zstdWriter)
	if err != nil {
		return err
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
