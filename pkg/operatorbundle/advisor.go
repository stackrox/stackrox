package operatorbundle

import (
	"context"
	"sort"
	"time"

	"github.com/pkg/errors"
)

// ProgressFunc receives human-readable progress messages as the analysis proceeds. It is
// optional; when unset the Advisor produces no progress output.
type ProgressFunc func(format string, args ...any)

// AdvisorOption configures an Advisor.
type AdvisorOption func(*Advisor)

// WithProgress sets a progress callback invoked at each step of the analysis.
func WithProgress(fn ProgressFunc) AdvisorOption {
	return func(a *Advisor) {
		a.progressFn = fn
	}
}

// Advisor orchestrates the operator bundle CVE upgrade analysis. It is constructed with the
// three external data sources and is safe to reuse across Advise calls.
type Advisor struct {
	catalog    CatalogClient
	scanner    ImageScanner
	installed  InstalledImageSource
	progressFn ProgressFunc
}

// NewAdvisor creates an Advisor backed by the given data sources.
func NewAdvisor(catalog CatalogClient, scanner ImageScanner, installed InstalledImageSource, opts ...AdvisorOption) *Advisor {
	a := &Advisor{catalog: catalog, scanner: scanner, installed: installed}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// progress emits a progress message when a callback is configured; otherwise it is a no-op.
func (a *Advisor) progress(format string, args ...any) {
	if a.progressFn != nil {
		a.progressFn(format, args...)
	}
}

// Advise analyzes the given vulnerable image digests and returns one BundleDiffReport per
// distinct installed bundle they resolve to. Digests that do not map to any bundle are
// collected in the returned unresolved slice. Reports are sorted by package name.
func (a *Advisor) Advise(ctx context.Context, digests []string) (reports []BundleDiffReport, unresolved []string, err error) {
	a.progress("Analyzing %d image digest(s)…", len(digests))

	// Group input digests by the installed bundle they belong to so a bundle shared by many
	// images is analyzed once.
	installedBundles := make(map[string]*Bundle)
	digestsByBundle := make(map[string][]string)
	var bundleKeys []string

	for _, digest := range digests {
		a.progress("Resolving installed bundle for %s", digest)
		bundle, err := a.catalog.FindInstalledBundle(ctx, digest)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "finding installed bundle for digest %s", digest)
		}
		if bundle == nil {
			a.progress("  → no operator bundle matched")
			unresolved = append(unresolved, digest)
			continue
		}
		a.progress("  → %s %s (channel %s)", bundle.Package, bundleVersionString(bundle), bundle.ChannelName)
		key := bundle.CSVName
		if key == "" {
			key = bundle.Package + "@" + bundle.Version
		}
		if _, seen := installedBundles[key]; !seen {
			installedBundles[key] = bundle
			bundleKeys = append(bundleKeys, key)
		}
		digestsByBundle[key] = append(digestsByBundle[key], digest)
	}

	for _, key := range bundleKeys {
		report, err := a.analyzeBundle(ctx, installedBundles[key], digestsByBundle[key])
		if err != nil {
			return nil, nil, err
		}
		reports = append(reports, report)
	}

	sort.Slice(reports, func(i, j int) bool {
		return reports[i].Package < reports[j].Package
	})
	return reports, unresolved, nil
}

func (a *Advisor) analyzeBundle(ctx context.Context, installed *Bundle, triggeringDigests []string) (BundleDiffReport, error) {
	report := BundleDiffReport{
		Package:           installed.Package,
		ChannelName:       installed.ChannelName,
		InstalledBundle:   *installed,
		TriggeringDigests: triggeringDigests,
	}

	a.progress("Package %s: fetching candidate versions since %s", installed.Package, installed.CreationDate)
	candidates, err := a.catalog.FindCandidateBundles(ctx, installed.Package, installed.ChannelName, installed.CreationDate)
	if err != nil {
		return report, errors.Wrapf(err, "finding candidate bundles for package %s", installed.Package)
	}
	a.progress("  found %d candidate version(s)", len(candidates))

	candidate, reason, err := SelectLatestPatch(bundleVersionString(installed), candidates)
	if err != nil {
		return report, errors.Wrapf(err, "selecting update candidate for package %s", installed.Package)
	}
	if candidate == nil {
		a.progress("  no update candidate: %s", reason)
		report.NoUpdateReason = reason
		return report, nil
	}
	a.progress("  selected update candidate %s → %s", bundleVersionString(installed), bundleVersionString(candidate))
	report.UpdateCandidate = candidate

	installedImages, err := a.collectInstalledCVEs(ctx, installed.RelatedImages)
	if err != nil {
		return report, err
	}
	candidateImages, err := a.collectCandidateCVEs(ctx, candidate.RelatedImages)
	if err != nil {
		return report, err
	}

	a.progress("Computing CVE diff…")
	report.ImageDiffs = ComputeDiff(installedImages, candidateImages)
	return report, nil
}

// collectInstalledCVEs looks up CVEs for already-scanned installed-bundle images by digest.
// Images not found in the source are skipped.
func (a *Advisor) collectInstalledCVEs(ctx context.Context, images []RelatedImage) ([]ImageCVEs, error) {
	a.progress("Fetching CVEs for %d installed image(s)", len(images))
	result := make([]ImageCVEs, 0, len(images))
	for i, img := range images {
		a.progress("  [%d/%d] installed %s (%s)", i+1, len(images), RepositoryFromReference(img.Image), img.Digest)
		cves, err := a.installed.GetCVEsByDigest(ctx, img.Digest)
		if err != nil {
			return nil, errors.Wrapf(err, "getting CVEs for installed image %s", img.Digest)
		}
		if cves == nil {
			a.progress("      not found in Central, skipping")
			continue
		}
		result = append(result, withCatalogMetadata(*cves, img))
	}
	return result, nil
}

// collectCandidateCVEs scans update-candidate images to obtain their CVEs.
func (a *Advisor) collectCandidateCVEs(ctx context.Context, images []RelatedImage) ([]ImageCVEs, error) {
	a.progress("Scanning %d candidate image(s) — this may take a while", len(images))
	result := make([]ImageCVEs, 0, len(images))
	for i, img := range images {
		ref := candidateReference(img)
		a.progress("  [%d/%d] scanning %s …", i+1, len(images), ref)
		start := time.Now()
		cves, err := a.scanner.ScanImage(ctx, ref)
		if err != nil {
			return nil, errors.Wrapf(err, "scanning candidate image %s", ref)
		}
		a.progress("      done in %s", time.Since(start).Round(time.Millisecond))
		if cves == nil {
			continue
		}
		result = append(result, withCatalogMetadata(*cves, img))
	}
	return result, nil
}

// candidateReference builds the fully-qualified reference used to scan a related image,
// pinning to the digest when available.
func candidateReference(img RelatedImage) string {
	repo := RepositoryFromReference(img.Image)
	if img.Digest != "" {
		return repo + "@" + img.Digest
	}
	return img.Image
}

// withCatalogMetadata attaches catalog-sourced identity to a scanned/looked-up ImageCVEs: the
// bundle logical Name (used together with Repository as the diff pairing key) and, when the
// data source did not populate one, a Repository derived from the catalog reference. The Name
// is only known from the catalog RelatedImage, not from the scanned image itself.
func withCatalogMetadata(cves ImageCVEs, img RelatedImage) ImageCVEs {
	if cves.Repository == "" {
		cves.Repository = RepositoryFromReference(img.Image)
	}
	cves.Name = img.Name
	return cves
}
