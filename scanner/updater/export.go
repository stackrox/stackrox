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
	_ "github.com/quay/claircore/updater/defaults"
	"github.com/quay/zlog"
	"github.com/stackrox/stackrox/scanner/v4/updater/manual"
	"golang.org/x/time/rate"
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

	// create temp folder
	err := os.Mkdir(outputDir, 0700)
	if err != nil {
		return err
	}

	// create temp file
	outputFile, err := os.Create(filepath.Join(outputDir, "output.json.ztd"))
	if err != nil {
		return err
	}
	defer outputFile.Close()

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

	updaterStore, err := jsonblob.New()
	updaterSetMgr, err := updates.NewManager(ctx, updaterStore, updates.NewLocalLockSource(), httpClient,
		updates.WithEnabled([]string{"oracle", "photon", "suse", "aws", "rhcc"}),
		updates.WithOutOfTree(outOfTree),
	)
	if err != nil {
		return err
	}
	if err := updaterSetMgr.Run(ctx); err != nil {
		return err
	}

	err = updaterStore.Store(zstdWriter)
	if err != nil {
		return err
	}

	configStore, err := jsonblob.New()
	configMgr, err := updates.NewManager(ctx, configStore, updates.NewLocalLockSource(), http.DefaultClient,
		updates.WithEnabled([]string{"debian", "alpine", "rhel", "ubuntu", "osv"}),
	)
	if err := configMgr.Run(ctx); err != nil {
		return err
	}

	err = configStore.Store(zstdWriter)
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
