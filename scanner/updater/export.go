package updater

import (
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

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

// Export is responsible for triggering the updaters to download Common Vulnerabilities and Exposures (CVEs) data
// and then outputting the result as a gzip file
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
	appendUpdaterSet(manual.Factory.UpdaterSet(ctx))
	appendUpdaterSet(oracle.UpdaterSet(ctx))
	appendUpdaterSet(osv.Factory.UpdaterSet(ctx))
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

	tempDir, err := os.MkdirTemp("", "output")
	if err != nil {
		return err
	}
	//defer os.RemoveAll(tempDir)
	outputFile, err := os.Create(filepath.Join(tempDir, "output.json.gz"))
	fmt.Println("Temporary directory path:", tempDir)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	gzipWriter := gzip.NewWriter(outputFile)
	defer gzipWriter.Close()

	httpClient := http.DefaultClient
	err = retryWithBackoff(3, 5*time.Second, func() error {
		updaterSetMgr, err := updates.NewManager(ctx, updaterStore, updates.NewLocalLockSource(), httpClient,
			updates.WithOutOfTree(updaters),
		)
		if err != nil {
			return err
		}
		if err := updaterSetMgr.Run(ctx); err != nil {
			return err
		}
		err = updaterStore.Store(gzipWriter)
		return err
	})

	if err != nil {
		return err
	}

	configStore, err := jsonblob.New()
	err = retryWithBackoff(3, 5*time.Second, func() error {
		configMgr, err := updates.NewManager(ctx, configStore, updates.NewLocalLockSource(), httpClient,
			updates.WithConfigs(cfgs),
			updates.WithFactories(facs),
		)
		if err != nil {
			return err
		}
		if err := configMgr.Run(ctx); err != nil {
			return err
		}
		err = configStore.Store(gzipWriter)
		return err
	})

	if err != nil {
		return err
	}

	return nil
}

func retryWithBackoff(maxAttempts int, backoff time.Duration, fn func() error) error {
	var err error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(backoff)
	}
	return err
}
