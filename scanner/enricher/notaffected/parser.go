package notaffected

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"strings"

	"github.com/klauspost/compress/snappy"
	"github.com/package-url/packageurl-go"
	"github.com/quay/claircore/libvuln/driver"
	"github.com/quay/claircore/toolkit/types/csaf"
	"github.com/quay/zlog"
	pkgnotaffected "github.com/stackrox/rox/pkg/scannerv4/enricher/notaffected"
)

// RepositoryKey should be used for every indexed repository coming from this package. It is
// used when persisting Red Hat VEX data pertaining to container images and referenced in the
// RHCC matching logic.
//
// TODO: This is defined in Claircore's rhcc package in versions post-v1.5.39.
const repositoryKey = "rhcc-container-repository"

// record represents enrichment data of a chunk of non-affected CVEs for a
// given product.
type record struct {
	prod  string
	chunk int
	cves  []string
}

// EnrichmentRecord marshals a record into a driver.EnrichmentRecord
func (r *record) EnrichmentRecord() (*driver.EnrichmentRecord, error) {
	b, err := json.Marshal(r.cves)
	if err != nil {
		return nil, err
	}
	return &driver.EnrichmentRecord{
		Tags: []string{
			r.prod,
			fmt.Sprintf("%s:%d", r.prod, r.chunk),
		},
		Enrichment: b,
	}, nil
}

// parseRecords parses VEX data to yield records for each chunk of non-affected enrichment data (i.e., a record).
func (e *Enricher) parseRecords(ctx context.Context, contents io.ReadCloser) iter.Seq2[*record, error] {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/notaffected/Enricher/parseRecords")
	return func(yield func(*record, error) bool) {
		pc := newProductCache()
		chunks := make(map[string]*record)

		r := bufio.NewReader(snappy.NewReader(contents))
		for b, err := r.ReadBytes('\n'); err == nil; b, err = r.ReadBytes('\n') {
			c, err := csaf.Parse(bytes.NewReader(b))
			if err != nil {
				yield(nil, fmt.Errorf("error parsing CSAF: %w", err))
				return
			}
			if c.Document.Tracking.Status == "deleted" {
				continue
			}
			var selfLink string
			for _, r := range c.Document.References {
				if r.Category == "self" {
					selfLink = r.URL
				}
			}
			ctx = zlog.ContextWithValues(ctx, "link", selfLink)
			creator := newCreator(c, pc)
			for _, v := range c.Vulnerabilities {
				prods, err := creator.knownNotAffectedVulnerabilities(ctx, v)
				if err != nil {
					yield(nil, err)
					return
				}
				for _, prod := range prods {
					// Get or create current chunk for this product.
					chunk := chunks[prod]
					if chunk == nil {
						chunk = &record{prod: prod, chunk: 0}
						chunks[prod] = chunk
					}
					// Add CVE to current chunk.
					chunk.cves = append(chunk.cves, v.CVE)
					// If we've reached the chunk size, yield the record.
					if len(chunk.cves) == e.maxCVEsPerRecord {
						// Reset to new chunk for next batch.
						chunks[prod] = &record{
							prod:  prod,
							chunk: chunk.chunk + 1,
						}
						if !yield(chunk, nil) {
							return
						}
					}
				}
			}
		}
		// Yield any remaining data.
		for _, chunk := range chunks {
			if len(chunk.cves) > 0 {
				if !yield(chunk, nil) {
					return
				}
			}
		}
	}
}

// ParseEnrichment implements driver.EnrichmentUpdater.
// The contents should be a line-delimited list of CSAF data, all of which is Snappy-compressed.
// This method parses out the data the enricher cares about and marshals the result into JSON.
func (e *Enricher) ParseEnrichment(ctx context.Context, contents io.ReadCloser) ([]driver.EnrichmentRecord, error) {
	ctx = zlog.ContextWithValues(ctx, "component", "enricher/notaffected/Enricher/ParseEnrichment")
	var out []driver.EnrichmentRecord
	for r, err := range e.parseRecords(ctx, contents) {
		if err != nil {
			return nil, err
		}
		er, err := r.EnrichmentRecord()
		if err != nil {
			return nil, err
		}
		out = append(out, *er)
	}
	return out, nil
}

// productCache keeps a cache of all seen csaf.Products.
type productCache struct {
	cache map[string]*csaf.Product
}

// NewProductCache returns a productCache with the backing
// map instantiated.
func newProductCache() *productCache {
	return &productCache{
		cache: make(map[string]*csaf.Product),
	}
}

// Get is a wrapper around the FindProductByID method that
// attempts to return from the cache before traversing the
// CSAF object.
func (pc *productCache) Get(productID string, c *csaf.CSAF) *csaf.Product {
	if p, ok := pc.cache[productID]; ok {
		return p
	}
	p := c.ProductTree.FindProductByID(productID)
	pc.cache[productID] = p
	return p
}

// NewCreator returns a creator object used for processing parts of a VEX file
// and returning claircore.Vulnerabilities.
func newCreator(c *csaf.CSAF, pc *productCache) *creator {
	return &creator{
		c:  c,
		pc: pc,
	}
}

// creator attempts to lessen the memory burden when creating vulnerability objects
// by caching objects that are used multiple times during processing.
type creator struct {
	c  *csaf.CSAF
	pc *productCache
}

// WalkRelationships attempts to resolve a relationship until we have a package product_id
// and a repo product_id. Relationships can be nested.
// If we don't get an initial relationship, or we don't get two component parts, we cannot
// create a vulnerability. We never see more than 3 components in the wild but if we did
// we'd assume the component next to the repo product_id is the package module product_id.
func walkRelationships(productID string, doc *csaf.CSAF) (string, error) {
	prodRel := doc.FindRelationship(productID, "default_component_of")
	if prodRel == nil {
		return "", fmt.Errorf("cannot determine initial relationship for %q", productID)
	}
	comps := extractProductNames(prodRel.ProductRef, prodRel.RelatesToProductRef, []string{}, doc)
	switch {
	case len(comps) == 2:
		// We have a package and repo
		return comps[0], nil
	case len(comps) > 2:
		// We have a package, module and repo
		return "", nil
	default:
		return "", fmt.Errorf("cannot determine relationships for %q", productID)
	}
}

// ExtractProductNames recursively looks up product_id relationships and adds them to a
// component slice in order. prodRef (and it's potential children) are leftmost in the return
// slice and relatesToProdRef (and it's potential children) are rightmost.
// For example: prodRef=a_pkg and relatesToProdRef=a_repo:a_module and a Relationship where
// Relationship.ProductRef=a_module and Relationship.RelatesToProductRef=a_repo the return
// slice would be: ["a_pkg", "a_module", "a_repo"].
func extractProductNames(prodRef, relatesToProdRef string, comps []string, c *csaf.CSAF) []string {
	prodRel := c.FindRelationship(prodRef, "default_component_of")
	if prodRel != nil {
		comps = extractProductNames(prodRel.ProductRef, prodRel.RelatesToProductRef, comps, c)
	} else {
		comps = append(comps, prodRef)
	}
	repoRel := c.FindRelationship(relatesToProdRef, "default_component_of")
	if repoRel != nil {
		comps = extractProductNames(repoRel.ProductRef, repoRel.RelatesToProductRef, comps, c)
	} else {
		comps = append(comps, relatesToProdRef)
	}
	return comps
}

// KnownNotAffectedVulnerabilities processes the "known_not_affected" array of products in the
// VEX object.
func (c *creator) knownNotAffectedVulnerabilities(ctx context.Context, v csaf.Vulnerability) ([]string, error) {
	knownNotAffected, exists := v.ProductStatus["known_not_affected"]
	if !exists {
		return nil, nil
	}

	if len(knownNotAffected) == 1 && knownNotAffected[0] == pkgnotaffected.RedHatProducts {
		return []string{pkgnotaffected.RedHatProducts}, nil
	}

	var prods []string
	for _, pc := range knownNotAffected {
		pkgName, err := walkRelationships(pc, c.c)
		if err != nil {
			// It's possible to get here due to middleware not having a defined component:package
			// relationship.
			continue
		}
		if pkgName == "" {
			// It's possible we got here due to empty fields or the presence of a module.
			// Either way, we are not interested in this.
			continue
		}
		if strings.HasPrefix(pkgName, "kernel") {
			// We don't want to ingest kernel advisories as
			// containers have no say in the kernel.
			continue
		}

		// pkgName will be overridden if we find a valid pURL
		compProd := c.pc.Get(pkgName, c.c)
		if compProd == nil {
			// Should never get here, error in data
			zlog.Warn(ctx).
				Str("pkg", pkgName).
				Msg("could not find package in product tree")
			continue
		}

		// It is possible that we will not find a pURL, in that case
		// the package.Name will be reported as-is.
		purlHelper, ok := compProd.IdentificationHelper["purl"]
		if ok {
			purl, err := packageurl.FromString(purlHelper)
			if err != nil {
				zlog.Warn(ctx).
					Str("purlHelper", purlHelper).
					Err(err).
					Msg("could not parse PURL")
				continue
			}
			if !checkPURL(purl) {
				continue
			}
			if purl.Type == packageurl.TypeRPM {
				// Not interested in RPMs.
				continue
			}
			if pn, err := extractPackageName(purl); err != nil {
				zlog.Warn(ctx).
					Str("purl", purl.String()).
					Err(err).
					Msg("could not extract package name from pURL")
			} else {
				// TODO: These may be binary packages?
				// pkgName = pn + "." + purl.Qualifiers.Map()["arch"]
				pkgName = pn
			}
		}

		// It is possible we could not determine a repository
		// when there is no pURL product ID helper.
		// In that case, we just accept it and hope for the best.
		prods = append(prods, pkgName)
	}

	return prods, nil
}

// ExtractPackageName deals with 1 pURL types: TypeOCI
//   - TypeOCI: check if there is Namespace and Name i.e. rhel7/rhel-atomic
//     and return that, if not, check for a repository_url qualifier. If the
//     repository_url exists then use the namespace/name part, if not, use
//     the purl.Name.
func extractPackageName(purl packageurl.PackageURL) (string, error) {
	switch purl.Type {
	case packageurl.TypeOCI:
		if purl.Namespace != "" {
			return purl.Namespace + "/" + purl.Name, nil
		}
		// Try finding an image name from the tag qualifier
		ru, ok := purl.Qualifiers.Map()["repository_url"]
		if !ok {
			return purl.Name, nil
		}
		_, image, found := strings.Cut(ru, "/")
		if !found {
			return "", fmt.Errorf("invalid repository_url for OCI pURL type %s", purl.String())
		}
		return image, nil
	default:
		return "", fmt.Errorf("unexpected purl type %s", purl.Type)
	}
}

var acceptedTypes = map[string]bool{
	packageurl.TypeOCI: true,
}

// CheckPURL checks if purl is something we're interested in.
//  1. Check the purl.Type is in the acceptable types.
//  2. Check if an advisory related to the kernel.
func checkPURL(purl packageurl.PackageURL) bool {
	if ok := acceptedTypes[purl.Type]; !ok {
		return false
	}
	if strings.HasPrefix(purl.Name, "kernel") {
		// We don't want to ingest kernel advisories as
		// containers have no say in the kernel.
		return false
	}
	return true
}
