package partial

import (
	"context"
	"encoding/json"

	"github.com/quay/claircore"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scanners/scannerv4"
)

const (
	// Type is the type of data returned from the Enricher's Enrich method.
	Type = "message/vnd.stackrox.scannerv4.partial; enricher=partial"

	// This appears above and must be the same.
	name = "partial"
)

var (
	_ driver.Enricher = (*Enricher)(nil)
)

// Enricher filters out all packages which are not
// known to be affected by any vulnerabilities.
//
// At this time, this is only supported for Node.js packages.
type Enricher struct{}

// Name implements driver.Enricher and driver.EnrichmentUpdater.
func (e Enricher) Name() string { return name }

// Enrich returns a list of Node.js package IDs which have no known vulnerabilities.
func (e Enricher) Enrich(ctx context.Context, _ driver.EnrichmentGetter, vr *claircore.VulnerabilityReport) (string, []json.RawMessage, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/partial/Enricher.Enrich")
	// ids tracks the package IDs of invulnerable Node.js packages.
	//
	// Note: it'd be nice if we could just delete the entries here, but the docs say modifying the vuln report is
	// not allowed. Instead, we simply identify invulnerable Node.js packages.
	var ids []string
	for pkgID, pkg := range vr.Packages {
		if srcType, _ := scannerv4.ParsePackageDB(pkg.PackageDB); srcType != storage.SourceType_NODEJS {
			continue
		}
		if vulnIDs := vr.PackageVulnerabilities[pkgID]; len(vulnIDs) == 0 {
			ids = append(ids, pkgID)
		}
	}

	if len(ids) == 0 {
		return Type, nil, nil
	}

	b, err := json.Marshal(ids)
	if err != nil {
		return Type, nil, err
	}

	return Type, []json.RawMessage{b}, nil
}
