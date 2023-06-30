package updater

import (
	"context"
	"net/http"
	"os"

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
)

// Export is responsible for triggering the updaters to download Common Vulnerabilities and Exposures (CVEs) data
// and then outputting the result as a zstd-compressed file with .ztd extension
func Export(ctx context.Context) error {
	updaterStore, err := jsonblob.New()
	if err != nil {
		return err
	}

	var updaters []driver.Updater

	// Append updater sets directly to the updaters
	appendUpdaterSet := func(updaterSet driver.UpdaterSet, err error) {
		if err != nil {
			zlog.Error(ctx).Msg(err.Error())
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

	os.Mkdir("tmp", 0700)
	outputFile, err := os.Create("tmp/output.json")
	if err != nil {
		return err
	}
	defer outputFile.Close()

	httpClient := http.DefaultClient

	updaterSetMgr, err := updates.NewManager(ctx, updaterStore, updates.NewLocalLockSource(), httpClient,
		updates.WithOutOfTree(updaters),
	)
	if err != nil {
		return err
	}
	if err := updaterSetMgr.Run(ctx); err != nil {
		return err
	}

	zstdWriter := zstd.NewWriter(outputFile)
	defer zstdWriter.Close()

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

	// Rename the output file to have the .ztd extension
	err = os.Rename("tmp/output.json", "tmp/output.json.ztd")
	if err != nil {
		return err
	}

	return nil
}
