// Package csaf provides a Red Hat CSAF enricher.
// The contents are strongly based on https://github.com/quay/claircore/tree/v1.5.34/rhel/vex.
//
// This exists as a temporary solution at this point, but there is potential for repurposing.
// TODO(ROX-26672): This enricher may no longer be needed once this is done.
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
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/scannerv4/enricher/csaf"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
)

var (
	_ driver.Enricher          = (*Enricher)(nil)
	_ driver.EnrichmentUpdater = (*Enricher)(nil)
)

const (
	// baseURL is the base url for the Red Hat CSAF security data.
	baseURL = "https://security.access.redhat.com/data/csaf/v2/advisories/"

	updaterVersion = "1"

	// The following consts match Claircore's VEX https://github.com/quay/claircore/blob/v1.5.34/rhel/vex/updater.go.

	latestFile     = "archive_latest.txt"
	changesFile    = "changes.csv"
	lookBackToYear = 2014
)

// Enricher provides Red Hat CSAF data as enrichments to a VulnerabilityReport.
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
	u := baseURL
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
	return csaf.Name
}

// Enrich implements driver.Enricher.
// For each vulnerability in the report, determine if there is a Red Hat advisory associated with it and map that vulnerability
// to the advisory's data, if applicable.
func (e *Enricher) Enrich(ctx context.Context, g driver.EnrichmentGetter, r *claircore.VulnerabilityReport) (string, []json.RawMessage, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/csaf/Enricher/Enrich")

	m := make(map[string][]json.RawMessage)

	erCache := make(map[string][]driver.EnrichmentRecord)
	for id, v := range r.Vulnerabilities {
		advisoryName := advisory(v)
		if advisoryName == "" {
			// Could not determine a related Red Hat advisory for this vulnerability,
			// so there is no point to attempt to fetch a related CSAF enrichment.
			// Skipping...
			continue
		}
		ctx := zlog.ContextWithValues(ctx, "original_vuln", v.Name, "advisory", advisoryName)
		rec, ok := erCache[advisoryName]
		if !ok {
			ts := []string{advisoryName}
			var err error
			rec, err = g.GetEnrichment(ctx, ts)
			if err != nil {
				return "", nil, err
			}
			erCache[advisoryName] = rec
		}
		zlog.Debug(ctx).
			Int("count", len(rec)).
			Msg("found records")
		for _, r := range rec {
			m[id] = append(m[id], r.Enrichment)
		}
	}
	if len(m) == 0 {
		return csaf.Type, nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return csaf.Type, nil, err
	}
	return csaf.Type, []json.RawMessage{b}, nil
}

// advisory determines the vulnerability's related Red Hat advisory name.
// Returns "" if an advisory cannot be determined.
func advisory(vuln *claircore.Vulnerability) string {
	// If we are not interested in Red Hat advisories,
	// then there is no point in continuing.
	if features.ScannerV4RedHatCVEs.Enabled() {
		return ""
	}
	// We only find Red Hat advisories in - you guessed it - the Red Hat updater.
	// End here if this vulnerability came from a different source.
	if !strings.EqualFold(vuln.Updater, mappers.RedHatUpdaterName) {
		return ""
	}

	name, found := mappers.FindName(vuln, mappers.RedHatAdvisoryPattern)
	if !found {
		return ""
	}

	return name
}
