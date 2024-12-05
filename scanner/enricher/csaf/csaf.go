// Package csaf provides a CSAF enricher.
// The contents are strongly based on https://github.com/quay/claircore/tree/v1.5.33/rhel/vex.
//
// This exists as a temporary solution TODO...
package csaf

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
)

var (
	_ driver.Enricher          = (*Enricher)(nil)
	_ driver.EnrichmentUpdater = (*Enricher)(nil)
)

const (
	// Type is the type of data returned from the Enricher's Enrich method.
	Type = `message/vnd.stackrox.scannerv4.map.csaf; enricher=stackrox.scannerv4.csaf`

	// BaseURL is the base url for the Red Hat CSAF security data.
	BaseURL = "https://security.access.redhat.com/data/csaf/v2/advisories/"

	updaterVersion = "1"

	// The following consts match Claircore's VEX https://github.com/quay/claircore/blob/v1.5.33/rhel/vex/updater.go.

	latestFile     = "archive_latest.txt"
	changesFile    = "changes.csv"
	deletionsFile  = "deletions.csv"
	lookBackToYear = 2014
)

type Record struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	CVSSv3      struct {
		Score  float64 `json:"score"`
		Vector string  `json:"vector"`
	} `json:"cvssv3"`
	CVSSv2 struct {
		Score  float64 `json:"score"`
		Vector string  `json:"vector"`
	} `json:"cvssv2"`
}

// Enricher provides NVD CVE data as enrichments to a VulnerabilityReport.
//
// Configure must be called before any other methods.
type Enricher struct {
	driver.NoopUpdater
	c    *http.Client
	base *url.URL
}

// NewFactory creates a Factory for the CSAF enricher.
func NewFactory() driver.UpdaterSetFactory {
	set := driver.NewUpdaterSet()
	_ = set.Add(&Enricher{})
	return driver.StaticSet(set)
}

// Config is the configuration for Enricher.
type Config struct {
	// URL indicates the base URL for the CSAF.
	//
	// Must include the trailing slash.
	URL string `json:"url" yaml:"url"`
}

// Configure implements driver.Configurable.
func (e *Enricher) Configure(_ context.Context, f driver.ConfigUnmarshaler, c *http.Client) error {
	e.c = c
	var cfg Config
	if err := f(&cfg); err != nil {
		return err
	}
	u := BaseURL
	if cfg.URL != "" {
		u = cfg.URL
		if !strings.HasSuffix(u, "/") {
			u += "/"
		}
	}
	var err error
	e.base, err = url.Parse(u)
	if err != nil {
		return err
	}
	return nil
}

// Name implements driver.Enricher and driver.EnrichmentUpdater.
func (*Enricher) Name() string {
	return "rhel-csaf"
}

// Enrich implements driver.Enricher.
func (e *Enricher) Enrich(ctx context.Context, g driver.EnrichmentGetter, r *claircore.VulnerabilityReport) (string, []json.RawMessage, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/csaf/Enricher/Enrich")

	// We return any CVSS blobs for CVEs mentioned in the free-form parts of the
	// vulnerability.
	m := make(map[string][]json.RawMessage)

	erCache := make(map[string][]driver.EnrichmentRecord)
	for id, v := range r.Vulnerabilities {
		ctx := zlog.ContextWithValues(ctx,
			"vuln", v.Name)
		rec, ok := erCache[v.Name]
		if !ok {
			ts := []string{v.Name}
			var err error
			rec, err = g.GetEnrichment(ctx, ts)
			if err != nil {
				return "", nil, err
			}
			erCache[v.Name] = rec
		}
		zlog.Debug(ctx).
			Int("count", len(rec)).
			Msg("found records")
		for _, r := range rec {
			m[id] = append(m[id], r.Enrichment)
		}
	}
	if len(m) == 0 {
		return Type, nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return Type, nil, err
	}
	return Type, []json.RawMessage{b}, nil
}
