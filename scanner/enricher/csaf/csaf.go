// Package csaf provides a Red Hat CSAF enricher.
// The contents are strongly based on https://github.com/quay/claircore/tree/v1.5.34/rhel/vex.
//
// This exists as a temporary solution at this point, but there is potential for repurposing.
// TODO(ROX-26672): This enricher may no longer be needed one this is done.
package csaf

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/rhel/vex"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/features"
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

	// The following consts match Claircore's VEX https://github.com/quay/claircore/blob/v1.5.34/rhel/vex/updater.go.

	latestFile     = "archive_latest.txt"
	changesFile    = "changes.csv"
	deletionsFile  = "deletions.csv"
	lookBackToYear = 2014
)

type Record struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	CVSSv3      CVSS   `json:"cvssv3"`
	CVSSv2      CVSS   `json:"cvssv2"`
}

type CVSS struct {
	Score  float32 `json:"score"`
	Vector string  `json:"vector"`
}

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
	return "stackrox.rhel-csaf"
}

// Enrich implements driver.Enricher.
func (e *Enricher) Enrich(ctx context.Context, g driver.EnrichmentGetter, r *claircore.VulnerabilityReport) (string, []json.RawMessage, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/csaf/Enricher/Enrich")

	m := make(map[string][]json.RawMessage)

	erCache := make(map[string][]driver.EnrichmentRecord)
	for id, v := range r.Vulnerabilities {
		vulnName := vulnerabilityName(v)
		ctx := zlog.ContextWithValues(ctx, "original_vuln", v.Name, "vuln", vulnName)
		rec, ok := erCache[vulnName]
		if !ok {
			ts := []string{vulnName}
			var err error
			rec, err = g.GetEnrichment(ctx, ts)
			if err != nil {
				return "", nil, err
			}
			erCache[vulnName] = rec
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

var (
	// The following vars match pkg/scannerv4/mappers/mappers.go and are copied here to prevent
	// circular dependencies.

	// Updater patterns are used to determine the security updater the
	// vulnerability was detected.

	rhelUpdaterName = (*vex.Updater)(nil).Name()

	// Name patterns are regexes to match against vulnerability fields to
	// extract their name according to their updater.

	// cveIDPattern captures CVEs.
	cveIDPattern = regexp.MustCompile(`CVE-\d{4}-\d+`)
	// rhelVulnNamePattern captures known Red Hat advisory patterns.
	// TODO(ROX-26672): Remove this and show CVE as the vulnerability name.
	rhelVulnNamePattern = regexp.MustCompile(`(RHSA|RHBA|RHEA)-\d{4}:\d+`)

	// vulnNamePatterns is a default prioritized list of regexes to match
	// vulnerability names.
	vulnNamePatterns = []*regexp.Regexp{
		// CVE
		cveIDPattern,
		// GHSA, see: https://github.com/github/advisory-database#ghsa-ids
		regexp.MustCompile(`GHSA(-[2-9cfghjmpqrvwx]{4}){3}`),
		// Catchall
		regexp.MustCompile(`[A-Z]+-\d{4}[-:]\d+`),
	}
)

// vulnerabilityName searches the best known candidate for the vulnerability name
// in the vulnerability details. It works by matching data against well-known
// name patterns, and defaults to the original name if nothing is found.
//
// TODO: This is modified from pkg/scannerv4/mappers/mappers.go to prevent circular dependencies.
// We should either combine these two, or better yet, remove the need for these.
// Any changes done here should be considered for the source, too.
func vulnerabilityName(vuln *claircore.Vulnerability) string {
	// Attempt per-updater patterns.
	switch {
	// TODO(ROX-26672): Remove this to show CVE as the vuln name.
	case strings.EqualFold(vuln.Updater, rhelUpdaterName):
		if !features.ScannerV4RedHatCVEs.Enabled() {
			if v, ok := findName(vuln, rhelVulnNamePattern); ok {
				return v
			}
		}
	}
	// Default patterns.
	for _, p := range vulnNamePatterns {
		if v, ok := findName(vuln, p); ok {
			return v
		}
	}
	return vuln.Name
}

// findName searches for a vulnerability name using the specified regex in
// pre-determined fields of the vulnerability, returning the name if found.
//
// TODO: This is modified from pkg/scannerv4/mappers/mappers.go to prevent circular dependencies.
// We should either combine these two, or better yet, remove the need for these.
// Any changes done here should be considered for the source, too.
func findName(vuln *claircore.Vulnerability, p *regexp.Regexp) (string, bool) {
	v := p.FindString(vuln.Name)
	if v != "" {
		return v, true
	}
	v = p.FindString(vuln.Links)
	if v != "" {
		return v, true
	}
	return "", false
}
