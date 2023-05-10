package updater

import (
	"bytes"
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

	// Initialize out with a buffer.
	buf := new(bytes.Buffer)
	out = gzip.NewWriter(buf)

	// Close the gzip writer at the end of the function.
	defer func() {
		if gw, ok := out.(*gzip.Writer); ok {
			if err := gw.Close(); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
	}()
	os.Mkdir("tmp", 0700)
	// Write the gzip data to a file.
	f, err := os.Create("tmp/updates.json.gz")
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, buf)
	if err != nil {
		return err
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer srv.Close()

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
