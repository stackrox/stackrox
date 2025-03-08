// Package nvd provides a NVD enricher.
package nvd

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/facebookincubator/nvdtools/cveapi/nvd/schema"
	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/nvd"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	_ driver.Enricher          = (*Enricher)(nil)
	_ driver.EnrichmentUpdater = (*Enricher)(nil)

	defaultFeed *url.URL
)

const (
	// DefaultFeeds is the default place to look for CVE feeds.
	DefaultFeeds = `https://services.nvd.nist.gov/rest/json/cves/2.0/`

	// First year for the yearly CVE feeds: https://nvd.nist.gov/vuln/data-feeds
	firstYear = 2002
)

func init() {
	var err error
	defaultFeed, err = url.Parse(DefaultFeeds)
	utils.CrashOnError(err)
}

// Enricher provides NVD CVE data as enrichments to a VulnerabilityReport.
//
// Configure must be called before any other methods.
type Enricher struct {
	driver.NoopUpdater
	c            *http.Client
	feed         *url.URL
	apiKey       string
	callInterval time.Duration
	feedPath     string
}

// Config is the configuration for Enricher.
type Config struct {
	FeedRoot     *string `json:"feed_root" yaml:"feed_root"`
	APIKey       *string `json:"api_key" yaml:"api_key"`
	CallInterval *string `json:"call_interval" yaml:"call_interval"`

	// FeedPath fetch NVD API v2 records from JSON files within a zip archive,
	// instead of fetching from the NVD URL.
	FeedPath *string `json:"feed_path" `
}

// NewFactory creates a Factory for the NVD enricher.
func NewFactory() driver.UpdaterSetFactory {
	set := driver.NewUpdaterSet()
	_ = set.Add(&Enricher{})
	return driver.StaticSet(set)
}

// Configure implements driver.Configurable.
func (e *Enricher) Configure(ctx context.Context, f driver.ConfigUnmarshaler, c *http.Client) error {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/nvd/Enricher/Configure")
	var cfg Config
	e.c = c
	if err := f(&cfg); err != nil {
		return err
	}
	// If a feed path is specified, we ignore the remote fetching configuration
	// parameters (they will not be used).
	if cfg.FeedPath != nil {
		st, err := os.Stat(*cfg.FeedPath)
		if err != nil {
			return err
		}
		if !st.Mode().IsRegular() {
			return fmt.Errorf("feed file is not a regular file: %s", *cfg.FeedPath)
		}
		e.feedPath = *cfg.FeedPath
		zlog.Info(ctx).
			Str("feed_path", e.feedPath).
			Msg("enricher configured with feed path")
		return nil
	}
	if cfg.APIKey != nil {
		e.apiKey = *cfg.APIKey
	}
	// NVD limits users without an API key to roughly one call every 6 seconds.
	// With an API key, it is roughly one call every 0.6 seconds.
	// We'll play it safe and do one call every 3 seconds.
	// As of writing there are ~216,000 vulnerabilities, so this whole process should take ~5.4 minutes.
	e.callInterval = 3 * time.Second
	if cfg.CallInterval != nil {
		var err error
		e.callInterval, err = time.ParseDuration(*cfg.CallInterval)
		if err != nil {
			return err
		}
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
		utils.CrashOnError(err)
	}
	zlog.Info(ctx).
		Str("feed", e.feed.String()).
		Bool("has_api_key", e.apiKey != "").
		Dur("call_interval", e.callInterval).
		Msg("enricher configured")
	return nil
}

// Name implements driver.Enricher and driver.EnrichmentUpdater.
func (*Enricher) Name() string {
	return nvd.Name
}

// FetchEnrichment implements driver.EnrichmentUpdater.
func (e *Enricher) FetchEnrichment(ctx context.Context, _ driver.Fingerprint) (io.ReadCloser, driver.Fingerprint, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/nvd/Enricher/FetchEnrichment")
	// Force a new hint, to signal updaters that this is new data.
	hint := driver.Fingerprint(uuid.NewV4().String())
	zlog.Info(ctx).Str("hint", string(hint)).Msg("starting fetch")
	out, err := os.CreateTemp("", "enricher.nvd.")
	if err != nil {
		return nil, hint, err
	}
	utils.Should(os.Remove(out.Name()))
	var success bool
	defer func() {
		if !success {
			if err := out.Close(); err != nil {
				zlog.Warn(ctx).Err(err).Msg("unable to close spool")
			}
		}
	}()
	enc := json.NewEncoder(out)
	var totalCVEs, totalSkippedCVEs int // stats for logging
	if e.feedPath == "" {
		// Fetching from NVD directly if a feed path is not specified.
		//
		// Doing this serially is slower, but much less complicated than using an
		// ErrGroup or the like.
		//
		// It may become an issue in 25-30 years.
		startDate := time.Date(firstYear, time.January, 1, 0, 0, 0, 0, time.UTC)
		now := time.Now().UTC()
		for startDate.Before(now) {
			// The maximum allowable range when using any date range parameters is 120 consecutive days.
			endDate := startDate.Add(120*24*time.Hour - time.Second)
			if endDate.After(now) {
				endDate = now
			}
			var apiResp *schema.CVEAPIJSON20
			for startIdx := 0; ; startIdx += apiResp.ResultsPerPage {
				apiResp, err = e.query(ctx, startDate, endDate, startIdx)
				if err != nil {
					return nil, hint, err
				}
				if apiResp.ResultsPerPage == 0 {
					break
				}
				var cvesFromQuery int
				// Parse vulnerabilities in the API response.
				for _, vuln := range apiResp.Vulnerabilities {
					item := filterFields(vuln.CVE)
					if item == nil {
						zlog.Warn(ctx).Str("cve", vuln.CVE.ID).Msg("skipping CVE")
						totalSkippedCVEs++
						continue
					}
					enrichment, err := json.Marshal(item)
					if err != nil {
						return nil, hint, fmt.Errorf("serializing CVE %s: %w", item.ID, err)
					}
					r := driver.EnrichmentRecord{
						Tags:       []string{item.ID},
						Enrichment: enrichment,
					}
					if err := enc.Encode(&r); err != nil {
						return nil, hint, fmt.Errorf("encoding enrichment: %w", err)
					}
					totalCVEs++
					cvesFromQuery++
				}
				zlog.Info(ctx).
					Int("count", cvesFromQuery).
					Int("total", totalCVEs).
					Msg("loaded vulnerabilities")
				// Rudimentary rate-limiting.
				time.Sleep(e.callInterval)
			}
			startDate = endDate.Add(time.Second)
		}
	} else {
		// Open a ZIP archive, and search for JSON files containing NVD CVE records. The
		// payload is expected to be of NVD API v2, with one record per line.
		zipF, err := zip.OpenReader(e.feedPath)
		if err != nil {
			return nil, hint, fmt.Errorf("opening zip: %w", err)
		}
		for i := range zipF.File {
			jsonF := zipF.File[i]
			if !strings.HasSuffix(jsonF.Name, ".nvd.json") {
				continue
			}
			// Iterates over a NVD CVE JSON file.
			var iterErr error
			jsonIter := func(yield func(vuln *schema.CVEAPIJSON20DefCVEItem) bool) {
				iterErr = nil
				f, err := jsonF.Open()
				if err != nil {
					iterErr = fmt.Errorf("opening bundle: %w", err)
					return
				}
				defer func() {
					_ = f.Close()
				}()
				dec := json.NewDecoder(f)
				for {
					var vuln schema.CVEAPIJSON20DefCVEItem
					err = dec.Decode(&vuln)
					if err != nil {
						break
					}
					if !yield(&vuln) {
						break
					}
				}
				if !errors.Is(err, io.EOF) {
					iterErr = err
				}
			}
			jsonIter(func(vuln *schema.CVEAPIJSON20DefCVEItem) bool {
				item := filterFields(vuln.CVE)
				if item == nil {
					zlog.Warn(ctx).Str("cve", vuln.CVE.ID).Msg("skipping CVE")
					totalSkippedCVEs++
					return true
				}
				var enrichment json.RawMessage
				enrichment, err = json.Marshal(item)
				if err != nil {
					err = fmt.Errorf("serializing CVE %s: %w", item.ID, err)
					return false
				}
				r := driver.EnrichmentRecord{
					Tags:       []string{item.ID},
					Enrichment: enrichment,
				}
				if err = enc.Encode(&r); err != nil {
					err = fmt.Errorf("encoding enrichment: %w", err)
					return false
				}
				totalCVEs++
				return true
			})
			if iterErr != nil {
				return nil, hint, iterErr
			}
		}
	}
	zlog.Info(ctx).
		Int("skipped", totalSkippedCVEs).
		Int("total", totalCVEs).
		Msg("loaded vulnerabilities")
	// Reset so clients can read the items.
	if _, err := out.Seek(0, io.SeekStart); err != nil {
		return nil, hint, fmt.Errorf("unable to reset item feed: %w", err)
	}
	success = true
	return out, hint, nil
}

func filterFields(cve *schema.CVEAPIJSON20CVEItem) *schema.CVEAPIJSON20CVEItem {
	var desc []*schema.CVEAPIJSON20LangString
	for _, d := range cve.Descriptions {
		if d.Lang == "en" {
			desc = append(desc, d)
			break
		}
	}
	item := &schema.CVEAPIJSON20CVEItem{
		ID:           cve.ID,
		Descriptions: desc,
		Metrics:      &schema.CVEAPIJSON20CVEItemMetrics{},
		Published:    cve.Published,
		LastModified: cve.LastModified,
	}
	// Return the item as-is if metrics are missing, as the description may still be useful.
	if cve.Metrics == nil {
		return item
	}
	for _, cvss := range cve.Metrics.CvssMetricV31 {
		if cvss.Type != "Primary" && cvss.Type != "" {
			continue
		}
		item.Metrics.CvssMetricV31 = append(item.Metrics.CvssMetricV31, &schema.CVEAPIJSON20CVSSV31{
			CvssData: &schema.CVSSV31{
				Version:      cvss.CvssData.Version,
				VectorString: cvss.CvssData.VectorString,
				BaseScore:    cvss.CvssData.BaseScore,
			},
		})
	}
	for _, cvss := range cve.Metrics.CvssMetricV30 {
		if cvss.Type != "Primary" && cvss.Type != "" {
			continue
		}
		item.Metrics.CvssMetricV30 = append(item.Metrics.CvssMetricV30, &schema.CVEAPIJSON20CVSSV30{
			CvssData: &schema.CVSSV30{
				Version:      cvss.CvssData.Version,
				VectorString: cvss.CvssData.VectorString,
				BaseScore:    cvss.CvssData.BaseScore,
			},
		})
	}
	for _, cvss := range cve.Metrics.CvssMetricV2 {
		if cvss.Type != "Primary" && cvss.Type != "" {
			continue
		}
		item.Metrics.CvssMetricV2 = append(item.Metrics.CvssMetricV2, &schema.CVEAPIJSON20CVSSV2{
			CvssData: &schema.CVSSV20{
				Version:      cvss.CvssData.Version,
				VectorString: cvss.CvssData.VectorString,
				BaseScore:    cvss.CvssData.BaseScore,
			},
		})
	}
	return item
}

func (e *Enricher) feedURL(start time.Time, end time.Time, startIdx int) string {
	// Feed URL should be validated during enricher configuration, crashing on errors.
	u, err := e.feed.Parse(".")
	utils.CrashOnError(err)
	v, err := url.ParseQuery(e.feed.RawQuery)
	utils.CrashOnError(err)
	v.Set("startIndex", strconv.Itoa(startIdx))
	v.Set("pubStartDate", start.Format("2006-01-02T15:04:05Z"))
	v.Set("pubEndDate", end.Format("2006-01-02T15:04:05Z"))
	// noRejected does not have a value, so manually append it here.
	u.RawQuery = v.Encode() + "&noRejected"
	return u.String()
}

func (e *Enricher) query(ctx context.Context, start, end time.Time, startIdx int) (*schema.CVEAPIJSON20, error) {
	ctx = zlog.ContextWithValues(ctx, "start_index", strconv.Itoa(startIdx),
		"start_time", start.String(),
		"end_time", end.String())
	u := e.feedURL(start, end, startIdx)
	zlog.Debug(ctx).Str("url", u).Msgf("starting NVD request")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}
	if e.apiKey != "" {
		req.Header.Set("apiKey", e.apiKey)
	}
	apiResp, err := e.queryWithBackoff(ctx, req)
	if err != nil {
		return nil, err
	}
	return apiResp, nil
}

func (e *Enricher) queryWithBackoff(ctx context.Context, req *http.Request) (apiResp *schema.CVEAPIJSON20, err error) {
	for i := 1; i <= 5; i++ {
		var resp *http.Response
		resp, err = e.tryQuery(ctx, req)
		if err == nil {
			apiResp, err = parseResponse(resp)
			if err == nil {
				break
			}
		}
		zlog.Warn(ctx).
			Int("attempt", i).
			Err(err).
			Msg("failed query attempt")
		// Wait some multiple of 3 seconds before next attempt.
		time.Sleep(time.Duration(3*i) * time.Second)
	}
	return apiResp, err
}

func (e *Enricher) tryQuery(ctx context.Context, req *http.Request) (*http.Response, error) {
	resp, err := e.c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching NVD API results: %w", err)
	}
	zlog.Debug(ctx).
		Int("status", resp.StatusCode).
		Str("nvd_message", req.Header.Get("message")).
		Msg("NVD response")
	if resp.StatusCode != 200 {
		_ = resp.Body.Close
		return nil, fmt.Errorf("unexpected status code when querying %s: %d", req.URL.String(), resp.StatusCode)
	}
	return resp, nil
}

func parseResponse(resp *http.Response) (*schema.CVEAPIJSON20, error) {
	defer func() {
		_ = resp.Body.Close()
	}()
	apiResp := new(schema.CVEAPIJSON20)
	if err := json.NewDecoder(resp.Body).Decode(apiResp); err != nil {
		return nil, fmt.Errorf("decoding API response: %w", err)
	}
	return apiResp, nil
}

// ParseEnrichment implements driver.EnrichmentUpdater.
func (e *Enricher) ParseEnrichment(ctx context.Context, rc io.ReadCloser) ([]driver.EnrichmentRecord, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/nvd/Enricher/ParseEnrichment")
	// Our Fetch method actually has all the smarts w/r/t to constructing the
	// records, so this is just decoding in a loop.
	defer func() {
		_ = rc.Close()
	}()
	var err error
	dec := json.NewDecoder(rc)
	ret := make([]driver.EnrichmentRecord, 0, 250_000) // Wild guess at initial capacity.
	// This is going to allocate like mad, hold onto your butts.
	for err == nil {
		ret = append(ret, driver.EnrichmentRecord{})
		err = dec.Decode(&ret[len(ret)-1])
	}
	zlog.Debug(ctx).
		Int("count", len(ret)-1).
		Msg("decoded enrichments")
	if !errors.Is(err, io.EOF) {
		return nil, err
	}
	return ret, nil
}

// This is a slightly more relaxed version of the validation pattern in the NVD
// JSON schema: https://csrc.nist.gov/schema/nvd/api/2.0/source_api_json_2.0.schema
//
// It allows for "CVE" to be case-insensitive and for dashes and underscores
// between the different segments.
var cveRegexp = regexp.MustCompile(`(?i:cve)[-_][0-9]{4}[-_][0-9]{4,}`)

// Enrich implements driver.Enricher.
func (e *Enricher) Enrich(ctx context.Context, g driver.EnrichmentGetter, r *claircore.VulnerabilityReport) (string, []json.RawMessage, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/nvd/Enricher/Enrich")

	// We return any CVSS blobs for CVEs mentioned in the free-form parts of the
	// vulnerability.
	m := make(map[string][]json.RawMessage)

	erCache := make(map[string][]driver.EnrichmentRecord)
	for id, v := range r.Vulnerabilities {
		t := make(map[string]struct{})
		ctx := zlog.ContextWithValues(ctx,
			"vuln", v.Name)
		for _, elem := range []string{
			v.Description,
			v.Name,
			v.Links,
		} {
			for _, m := range cveRegexp.FindAllString(elem, -1) {
				t[m] = struct{}{}
			}
		}
		if len(t) == 0 {
			continue
		}
		ts := make([]string, 0, len(t))
		for m := range t {
			ts = append(ts, m)
		}
		zlog.Debug(ctx).
			Strs("cve", ts).
			Msg("found CVEs")

		slices.Sort(ts)
		cveKey := strings.Join(ts, "_")
		rec, ok := erCache[cveKey]
		if !ok {
			var err error
			rec, err = g.GetEnrichment(ctx, ts)
			if err != nil {
				return "", nil, err
			}
			erCache[cveKey] = rec
		}
		zlog.Debug(ctx).
			Int("count", len(rec)).
			Msg("found records")
		for _, r := range rec {
			m[id] = append(m[id], r.Enrichment)
		}
	}
	if len(m) == 0 {
		return nvd.Type, nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nvd.Type, nil, err
	}
	return nvd.Type, []json.RawMessage{b}, nil
}
