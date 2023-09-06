package enrichment

import (
	"context"
	"net/http"
	"time"

	"io"
	"testing"

	"github.com/pkg/errors"
	"github.com/quay/claircore/enricher/cvss"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
)

var (
	pkgClient = &http.Client{}

	fp driver.Fingerprint
)

// TestCVSS verifies the accessibility and correct parsing of CVSS data stored in Google Storage.
func TestCVSS(t *testing.T) {
	ctx := zlog.Test(context.Background(), t)
	e := &cvss.Enricher{}

	// Create a custom feed root URL
	customFeedRoot := "https://storage.googleapis.com/scanner-v4-test/nvddata/"

	// Create a custom config unmarshaler
	configFunc := func(cfg interface{}) error {
		c, ok := cfg.(*cvss.Config) // Type assertion for safety
		if !ok {
			return errors.New("invalid config type")
		}
		c.FeedRoot = &customFeedRoot
		return nil
	}

	err := e.Configure(ctx, configFunc, pkgClient)
	if err != nil {
		t.Errorf("Failed to configure enricher: %v", err) // Print the error and fail the test
		return
	}
	runEnricher(ctx, t, e)
}

func runEnricher(ctx context.Context, t *testing.T, u driver.EnrichmentUpdater) {
	var rc io.ReadCloser
	var nfp driver.Fingerprint
	var err error
	// Debounce any network hiccups.
	for i := 0; i < 5; i++ {
		rc, nfp, err = u.FetchEnrichment(ctx, fp)
		if err == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After((2 << i) * time.Second):
		}
	}
	if err != nil {
		t.Fatal(err)
	}
	t.Log(nfp)
	defer func() {
		if err := rc.Close(); err != nil {
			t.Log(err)
		}
	}()

	ers, err := u.ParseEnrichment(ctx, rc)
	if err != nil {
		t.Error(err)
	}
	t.Logf("reported %d records", len(ers))
}
