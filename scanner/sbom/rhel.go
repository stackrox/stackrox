package sbom

import (
	"context"
	"log/slog"
	"strings"

	"github.com/package-url/packageurl-go"
	"github.com/quay/claircore/rhel"
	"github.com/stackrox/rox/scanner/matcher/repo2cpe"
)

func NewRHELCPETransformFunc(updater *repo2cpe.Updater) func(ctx context.Context, p *packageurl.PackageURL) error {
	return func(ctx context.Context, p *packageurl.PackageURL) error {
		qs := p.Qualifiers.Map()
		if _, has := qs[rhel.PURLRepositoryCPEs]; has {
			return nil
		}
		repoID, ok := qs[rhel.PURLRepositoryID]
		if !ok || repoID == "" {
			return nil
		}
		mapping, err := updater.Get(ctx)
		if err != nil {
			slog.WarnContext(ctx, "repo2cpe mapping unavailable; skipping CPE enrichment",
				"repository_id", repoID, "reason", err)
			return nil
		}
		cpes, found := mapping.GetCPEs(repoID)
		if !found || len(cpes) == 0 {
			return nil
		}
		qs[rhel.PURLRepositoryCPEs] = strings.Join(cpes, ",")
		p.Qualifiers = packageurl.QualifiersFromMap(qs)
		return nil
	}
}
