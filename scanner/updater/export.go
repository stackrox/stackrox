package updater

import (
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"

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
	"github.com/stackrox/scanner/v4/updater/manual"
)

// ExportAction is responsible for triggering the updaters to download Common Vulnerabilities and Exposures (CVEs) data
// and then outputting the result as a gzip file
func ExportAction() error {
	ctx := context.Background()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	updaterStore, err := jsonblob.New()
	if err != nil {
		return err
	}

	updaterList := []driver.Updater{}

	// Append updater sets directly to the updaterList
	appendUpdaterSet := func(updaterSet driver.UpdaterSet, err error) {
		if err != nil {
			zlog.Error(ctx).Msg(err.Error())
			return
		}
		updaterList = append(updaterList, updaterSet.Updaters()...)
	}

	appendUpdaterSet(aws.UpdaterSet(ctx))
	appendUpdaterSet(oracle.UpdaterSet(ctx))
	appendUpdaterSet(photon.UpdaterSet(ctx))
	appendUpdaterSet(suse.UpdaterSet(ctx))
	appendUpdaterSet(osv.Factory.UpdaterSet(ctx))
	appendUpdaterSet(manual.Factory.UpdaterSet(ctx))

	debianFac, err := debian.NewFactory(ctx)
	if err != nil {
		return err
	}

	alpineFac, err := alpine.NewFactory(ctx)
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
	}

	facs := map[string]driver.UpdaterSetFactory{
		"debian": debianFac,
		"alpine": alpineFac,
		"rhel":   rhelFac,
		"ubuntu": ubuntuFac,
	}

	os.Mkdir("tmp", 0700)
	outputFile, err := os.Create("tmp/output.json.gz")
	if err != nil {
		return err
	}
	defer outputFile.Close()
	gzipWriter := gzip.NewWriter(outputFile)
	defer gzipWriter.Close()

	updaterSetMgr, err := updates.NewManager(ctx, updaterStore, updates.NewLocalLockSource(), http.DefaultClient,
		updates.WithOutOfTree(updaterList),
	)
	if err != nil {
		return err
	}
	if err := updaterSetMgr.Run(ctx); err != nil {
		return err
	}
	err = updaterStore.Store(gzipWriter)
	if err != nil {
		return err
	}

	configStore, err := jsonblob.New()
	configMgr, err := updates.NewManager(ctx, configStore, updates.NewLocalLockSource(), srv.Client(),
		updates.WithConfigs(cfgs),
		updates.WithFactories(facs),
	)
	if err := configMgr.Run(ctx); err != nil {
		return err
	}

	err = configStore.Store(gzipWriter)
	if err != nil {
		return err
	}

	err = gzipWriter.Flush()
	if err != nil {
		return err
	}

	return nil
}
