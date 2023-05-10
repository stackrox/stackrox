package updater

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/libvuln/jsonblob"
	"github.com/quay/claircore/libvuln/updates"
	"github.com/quay/claircore/updater/osv"
)

// ExportAction is responsible for triggering the updaters to download Common Vulnerabilities and Exposures (CVEs) data
// and then outputting the result as a gzip file
func ExportAction() error {
	ctx := context.Background()
	var out io.Writer

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

	enc := gzip.NewWriter(out)
	defer func() {
		if err := enc.Close(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	out = enc

	cfgs := make(map[string]driver.ConfigUnmarshaler, 1)
	cfgs["osv"] = func(v interface{}) error {
		cfg := v.(*osv.Config)
		cfg.URL = osv.DefaultURL
		return nil
	}
	facs := make(map[string]driver.UpdaterSetFactory, 1)
	facs["osv"] = osv.Factory

	store, err := jsonblob.New()
	if err != nil {
		return err
	}
	defer func() {
		if err := store.Store(out); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()
	mgr, err := updates.NewManager(ctx, store, updates.NewLocalLockSource(), srv.Client(),
		updates.WithConfigs(cfgs),
		updates.WithFactories(facs),
	)
	if err != nil {
		return err
	}

	if err := mgr.Run(ctx); err != nil {
		return err
	}
	return nil
}
