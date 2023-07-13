package updater

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/quay/claircore/alpine"
	"github.com/quay/claircore/aws"
	"github.com/quay/claircore/debian"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/jsonblob"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/claircore/oracle"
	"github.com/quay/claircore/photon"
	"github.com/quay/claircore/rhel"
	"github.com/quay/claircore/suse"
	"github.com/quay/claircore/ubuntu"
	"github.com/quay/claircore/updater/osv"
	"github.com/quay/zlog"
	"github.com/stackrox/stackrox/scanner/v4/updater/manual"
	"golang.org/x/time/rate"
)

// Export is responsible for triggering the updaters to download Common Vulnerabilities and Exposures (CVEs) data
// and then outputting the result as a zstd-compressed file with .ztd extension
func Export(ctx context.Context, outputDir string) error {
	var updaters []driver.Updater

	// Append updater sets directly to the updaters
	appendUpdaterSet := func(updaterSet driver.UpdaterSet, err error) {
		if err != nil {
			zlog.Error(ctx).Err(err).Send()
			return
		}
		updaters = append(updaters, updaterSet.Updaters()...)
	}

	appendUpdaterSet(aws.UpdaterSet(ctx))
	appendUpdaterSet(manual.UpdaterSet(ctx, nil))
	appendUpdaterSet(oracle.UpdaterSet(ctx))
	appendUpdaterSet(photon.UpdaterSet(ctx))
	appendUpdaterSet(suse.UpdaterSet(ctx))

	alpineFac, err := alpine.NewFactory(ctx)
	if err != nil {
		return err
	}

	debianFac, err := debian.NewFactory(ctx)
	if err != nil {
		return err
	}

	rhelFac, err := rhel.NewFactory(ctx, rhel.DefaultManifest)
	if err != nil {
		return err
	}

	ubuntuFac, err := ubuntu.NewFactory(ctx)
	if err != nil {
		return err
	}

	osvFac := osv.Factory

	cfgs := map[string]driver.ConfigUnmarshaler{
		"debian": func(v interface{}) error {
			v.(*debian.FactoryConfig).MirrorURL = `https://deb.debian.org/`
			return nil
		},
		"alpine": func(v interface{}) error {
			v.(*alpine.FactoryConfig).URL = "https://secdb.alpinelinux.org/"
			return nil
		},
		"rhel": func(v interface{}) error {
			return nil
		},
		"ubuntu": func(v interface{}) error {
			v.(*ubuntu.FactoryConfig).Name = "ubuntu"
			return nil
		},
		"osv": func(v interface{}) error {
			cfg := v.(*osv.FactoryConfig)
			cfg.URL = osv.DefaultURL
			return nil
		},
	}

	facs := map[string]driver.UpdaterSetFactory{
		"debian": debianFac,
		"alpine": alpineFac,
		"rhel":   rhelFac,
		"ubuntu": ubuntuFac,
		"osv":    osvFac,
	}

	// create temp folder
	err = os.Mkdir(outputDir, 0700)
	if err != nil {
		return err
	}

	// create temp file
	outputFile, err := os.CreateTemp(outputDir, "output*.json")
	if err != nil {
		return err
	}
	defer os.Remove(outputFile.Name())

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
		updates.WithOutOfTree(updaters),
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
		updates.WithConfigs(cfgs),
		updates.WithFactories(facs),
	)
	if err := configMgr.Run(ctx); err != nil {
		return err
	}

	err = configStore.Store(zstdWriter)
	if err != nil {
		return err
	}

	// use .ztd
	// Generate the final output file path with the changed extension
	finalOutputPath := outputFile.Name() + ".ztd"

	// Rename the temporary file
	err = os.Rename(outputFile.Name(), finalOutputPath)
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
