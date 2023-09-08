package enricher

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
)

// Ensure CvssEnricher implements driver.Enricher
var (
	defaultFeed *url.URL
)

const (
	// DefaultFeeds is the default place to look for CVE feeds.
	//
	// The enricher expects the structure to mirror that found here: files
	// organized by year, prefixed with `nvdcve-1.1-` and with `.meta` and
	// `.json.gz` extensions.
	//
	//doc:url updater
	DefaultFeeds = `https://nvd.nist.gov/feeds/json/cve/1.1/`

	// This appears above and must be the same.
	name = `CVSS_Enricher`

	// First year for the yearly CVE feeds: https://nvd.nist.gov/vuln/data-feeds
	firstYear = 2002
)

// Initialize the defaultFeed.
func init() {
	var err error
	defaultFeed, err = url.Parse(DefaultFeeds)
	if err != nil {
		panic(err)
	}
}

type Enricher struct {
	driver.NoopUpdater
	c    *http.Client
	feed *url.URL
}

// Config holds the configuration for the CvssEnricher.
type Config struct {
	FeedRoot *string `json:"feed_root" yaml:"feed_root"`
}

// Configure sets up the CvssEnricher with given configuration.
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

// Name returns the name of the enricher.
func (e *Enricher) Name() string {
	return name
}

// FetchEnrichment fetches the enrichment data for the given fingerprint and file path.
func (e *Enricher) FetchEnrichment(ctx context.Context, hint driver.Fingerprint, fileName string) (string, driver.Fingerprint, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "central/scannerV4Definitions/cvss/FetchEnrichment")

	prev := make(map[int]string)
	if err := json.Unmarshal([]byte(hint), &prev); err != nil && hint != "" {
		return "", driver.Fingerprint(""), err
	}
	cur := make(map[int]string, len(prev))
	yrs := make([]int, 0)

	for y, lim := firstYear, time.Now().Year(); y <= lim; y++ {
		yrs = append(yrs, y)
		u, err := metafileURL(e.feed, y)
		if err != nil {
			return "", hint, err
		}
		zlog.Debug(ctx).
			Int("year", y).
			Stringer("url", u).
			Msg("fetching meta file")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return "", hint, err
		}
		res, err := e.c.Do(req)
		if err != nil {
			return "", hint, err
		}
		var buf bytes.Buffer
		_, err = io.Copy(&buf, res.Body)
		if err != nil {
			return "", hint, err
		}
		err = res.Body.Close() // Don't defer because we're in a loop.
		if err != nil {
			zlog.Error(ctx).Msg(err.Error())
		}
		var mf meta
		if err := mf.parseBufferToMeta(&buf); err != nil {
			return "", hint, err
		}
		zlog.Debug(ctx).
			Int("year", y).
			Stringer("url", u).
			Time("mod", mf.LastModifiedDate).
			Msg("parsed meta file")
		cur[y] = strings.ToUpper(mf.SHA256)
	}

	doFetch := false
	for _, y := range yrs {
		if prev[y] != cur[y] {
			doFetch = true
			break
		}
	}
	if !doFetch {
		return "", hint, driver.Unchanged
	}

	tmpDir, err := os.MkdirTemp("", "prefix")
	if err != nil {
		return "", hint, fmt.Errorf("failed to create temp directory: %w", err)
	}
	tmpFilePath := filepath.Join(tmpDir, fileName)
	out, err := os.Create(tmpFilePath)
	if err != nil {
		return "", hint, fmt.Errorf("failed to create temp file: %w", err)
	}

	for _, y := range yrs {
		u, err := gzURL(e.feed, y)
		if err != nil {
			return "", hint, fmt.Errorf("bad URL: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return "", hint, fmt.Errorf("unable to create request: %w", err)
		}
		res, err := e.c.Do(req)
		if err != nil {
			return "", hint, fmt.Errorf("unable to do request: %w", err)
		}

		gz, err := gzip.NewReader(res.Body)
		if err != nil {
			res.Body.Close()
			return "", hint, fmt.Errorf("unable to create gzip reader: %w", err)
		}
		err = ProcessAndWriteCVSS(y, ctx, gz, out)
		if err != nil {
			return "", hint, fmt.Errorf("unable to write item feed for year %d: %w", y, err)
		}
		err = gz.Close()
		if err != nil {
			zlog.Error(ctx).Msg(err.Error())
		}
	}
	nh, err := json.Marshal(cur)
	if err != nil {
		return "", "", fmt.Errorf("unable to serialize new hint: %w", err)
	}
	// After all the processing and writing to the out file
	info, err := out.Stat() // Get the file info
	if err != nil {
		// Handle error if required, or you can simply log it
		zlog.Warn(ctx).Err(err).Msg("Failed to get the file size.")
	} else {
		size := info.Size() // This gives you size in bytes
		zlog.Info(ctx).
			Str("fileName", out.Name()).
			Int64("fileSize", size).
			Msg("Size of the JSON file.")
	}
	return out.Name(), driver.Fingerprint(nh), nil
}

func metafileURL(root *url.URL, yr int) (*url.URL, error) {
	return root.Parse(fmt.Sprintf("nvdcve-1.1-%d.meta", yr))
}

func gzURL(root *url.URL, yr int) (*url.URL, error) {
	return root.Parse(fmt.Sprintf("nvdcve-1.1-%d.json.gz", yr))
}
