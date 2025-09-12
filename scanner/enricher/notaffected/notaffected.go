// Package notaffected provides an enricher which lists all the OCI (container-first) Red Hat products unaffected by a CVE.
//
// The implementation is strongly based on https://github.com/quay/claircore/tree/v1.5.39/rhel/vex.
package notaffected

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/rhel/rhcc"
	"github.com/quay/zlog"
	pkgnotaffected "github.com/stackrox/rox/pkg/scannerv4/enricher/notaffected"
	"github.com/stackrox/rox/pkg/set"
)

var (
	_ driver.Enricher          = (*Enricher)(nil)
	_ driver.EnrichmentUpdater = (*Enricher)(nil)
)

// The following match Claircore's VEX https://github.com/quay/claircore/blob/v1.5.39/rhel/vex/updater.go.
const (
	// baseURL is the base url for the Red Hat VEX security data.
	baseURL = "https://security.access.redhat.com/data/csaf/v2/vex/"

	latestFile              = "archive_latest.txt"
	changesFile             = "changes.csv"
	lookBackToYear          = 2015
	updaterVersion          = "6"
	defaultMaxCVEsPerRecord = 1000
)

var (
	// legacyRHCCRepoName is the name of the "Gold Repository".
	legacyRHCCRepoName = rhcc.GoldRepo.Name
	// legacyRHCCRepoURI is the URI of the "Gold Repository".
	legacyRHCCRepoURI = rhcc.GoldRepo.URI
)

// Enricher provides unaffected OCI Red Hat product data as enrichments to a VulnerabilityReport.
//
// Configure must be called before any other methods.
type Enricher struct {
	driver.NoopUpdater
	c    *http.Client
	base *url.URL

	// maxCVEsPerRecord limits how many CVEs are encoded in a single enrichment
	// record. A value of 0 uses the default.
	maxCVEsPerRecord int
}

// NewFactory creates a Factory for the Not Affected enricher.
func NewFactory() driver.UpdaterSetFactory {
	set := driver.NewUpdaterSet()
	_ = set.Add(&Enricher{})
	return driver.StaticSet(set)
}

// Config is the configuration for Enricher.
type Config struct {
	// URL indicates the base URL for the VEX.
	//
	// Must include the trailing slash.
	URL              string `json:"url" yaml:"url"`
	MaxCVEsPerRecord int    `json:"max_cves_per_record" yaml:"max_cves_per_record"`
}

// Configure implements driver.Configurable.
func (e *Enricher) Configure(_ context.Context, f driver.ConfigUnmarshaler, c *http.Client) error {
	e.c = c
	e.maxCVEsPerRecord = defaultMaxCVEsPerRecord
	var cfg Config
	if err := f(&cfg); err != nil {
		return err
	}
	if cfg.MaxCVEsPerRecord > 0 {
		e.maxCVEsPerRecord = cfg.MaxCVEsPerRecord
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
	return pkgnotaffected.Name
}

// Enrich implements driver.Enricher.
//
// Return the vulnerabilities which, according to Red Hat's VEX data, do not affect the image.
func (e *Enricher) Enrich(ctx context.Context, g driver.EnrichmentGetter, r *claircore.VulnerabilityReport) (string, []json.RawMessage, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/notaffected/Enricher/Enrich")

	// Find the repository ID which identifies this image as a (or based on a) Red Hat product.
	var rhccID string
	for id, repo := range r.Repositories {
		if repo.Key == repositoryKey || repo.Name == legacyRHCCRepoName && repo.URI == legacyRHCCRepoURI {
			rhccID = id
			break
		}
	}
	if rhccID == "" {
		// Not an official Red Hat product nor based on one.
		return pkgnotaffected.Type, nil, nil
	}

	// Identify the name of each discovered Red Hat image by the RHCC repository
	// scanner. Notice that for older image RH builds this can identify not only the
	// final Red Hat image, but also the images on which that final image was based.
	// We don't expect this on current images.
	pkgIDs := set.NewStringSet()
	for pkgID, envs := range r.Environments {
		for _, env := range envs {
			for _, repoID := range env.RepositoryIDs {
				if repoID == rhccID {
					pkgIDs.Add(pkgID)
				}
			}
		}
	}
	pkgNames := make([]string, 0, len(pkgIDs))
	for pkgID := range pkgIDs {
		pkg, exists := r.Packages[pkgID]
		if !exists {
			continue
		}
		pkgNames = append(pkgNames, pkg.Name)
	}

	m := make(map[string][]json.RawMessage)
	// Include the special "red_hat_products" package (or product version in VEX
	// terms), it's a wildcard for "all red hat products".
	for _, pkgName := range append(pkgNames, pkgnotaffected.RedHatProducts) {
		ts := []string{pkgName}
		rec, err := g.GetEnrichment(ctx, ts)
		if err != nil {
			return pkgnotaffected.Type, nil, err
		}
		// We expect more than one record since we chunk them during parsing.
		for _, r := range rec {
			m[pkgName] = append(m[pkgName], r.Enrichment)
		}
	}

	if len(m) == 0 {
		return pkgnotaffected.Type, nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return pkgnotaffected.Type, nil, err
	}
	return pkgnotaffected.Type, []json.RawMessage{b}, nil
}
