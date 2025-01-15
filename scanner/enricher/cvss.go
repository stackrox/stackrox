package enricher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
)

var (
	_ driver.Enricher          = (*Enricher)(nil)
	_ driver.EnrichmentUpdater = (*Enricher)(nil)

	defaultFeed  *url.URL
	cvssFilePath string
)

const (
	// Type is the type of data returned from the Enricher's Enrich method.
	Type = `message/vnd.clair.map.vulnerability; enricher=rox.cvss schema=https://csrc.nist.gov/schema/nvd/feed/1.1/cvss-v3.x.json`
	// DefaultFeeds is the default place to look for CVE feeds.
	//
	// The enricher expects the structure to mirror that found here: files
	// organized by year, prefixed with `nvdcve-1.1-` and with `.meta` and
	// `.json.gz` extensions.
	//
	//doc:url updater
	DefaultFeeds = `https://storage.googleapis.com/scanner-v4-test/nvd-bundle/nvd-data.tar.gz`

	// This appears above and must be the same.
	name = `rox.cvss`

	baseDir        = "v4enrichmentdata"
	cvssFileSuffix = "data.tar.gz"
)

func init() {
	var err error
	defaultFeed, err = url.Parse(DefaultFeeds)
	if err != nil {
		panic(err)
	}
}

// Enricher provides CVSS data as enrichments to a VulnerabilityReport.
//
// Configure must be called before any other methods.
type Enricher struct {
	driver.NoopUpdater
	c    *http.Client
	feed *url.URL
}

// Config is the configuration for Enricher.
type Config struct {
	FeedRoot *string `json:"feed_root" yaml:"feed_root"`
}

// Configure implements driver.Configurable.
func (e *Enricher) Configure(ctx context.Context, f driver.ConfigUnmarshaler, c *http.Client) error {
	var cfg Config
	e.c = c
	if err := f(&cfg); err != nil {
		return err
	}
	if cfg.FeedRoot != nil {
		if !strings.HasSuffix(*cfg.FeedRoot, "/") {
			return fmt.Errorf("URL missing trailing slash: %q", *cfg.FeedRoot)
		}
		u, err := url.Parse(*cfg.FeedRoot)
		if err != nil {
			return err
		}
		e.feed = u
	} else {
		var err error
		e.feed, err = defaultFeed.Parse(".")
		if err != nil {
			panic("programmer error: " + err.Error())
		}
	}
	return nil
}

func (e Enricher) Enrich(ctx context.Context, getter driver.EnrichmentGetter, report *claircore.VulnerabilityReport) (string, []json.RawMessage, error) {
	//TODO implement me
	panic("implement me")
}

func (e Enricher) Name() string {
	return name
}

func (e Enricher) FetchEnrichment(ctx context.Context, hint driver.Fingerprint) (io.ReadCloser, driver.Fingerprint, error) {
	u := e.feed
	zlog.Debug(ctx).
		Stringer("url", u).
		Msg("fetching CVSS data bundle")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, hint, err
	}
	res, err := e.c.Do(req)
	if err != nil {
		return nil, hint, err
	}
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", baseDir)
	if err != nil {
		return nil, hint, err
	}

	cvssFilePath = filepath.Join(tmpDir, cvssFileSuffix)
	tmpFile, err := os.Create(cvssFilePath)
	if err != nil {
		return nil, hint, err
	}

	// Copy the data from the HTTP response to the file
	_, err = io.Copy(tmpFile, res.Body)
	if err != nil {
		return nil, hint, err
	}

	// Close the response body and the file
	if err := res.Body.Close(); err != nil {
		return nil, hint, err
	}
	if err := tmpFile.Close(); err != nil {
		return nil, hint, err
	}
	return nil, hint, nil
}

func (e Enricher) ParseEnrichment(ctx context.Context, closer io.ReadCloser) ([]driver.EnrichmentRecord, error) {
	//TODO implement me
	panic("implement me")
}
