// Package olm provides an operatorbundle.CatalogClient backed entirely by in-cluster OLM
// data, with no dependency on the Red Hat Ecosystem Catalog (Pyxis):
//
//   - FindInstalledBundle resolves an image digest to the installed operator bundle by
//     matching the digest against the spec.relatedImages of the installed
//     ClusterServiceVersions (via the Kubernetes API). This is unambiguous: exactly one
//     version of an operator is installed at a time.
//   - FindCandidateBundles enumerates newer bundle versions of the same package from the
//     operator catalog index, served over gRPC by the CatalogSource registry
//     (operator-registry's api.Registry). Unlike PackageManifest, the index preserves the
//     per-image name that the (repository, name) diff pairing relies on.
package olm

import (
	"context"
	"encoding/json"
	"io"
	"regexp"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/operatorbundle"
	api "github.com/stackrox/rox/tools/operatorbundle/olm/registryapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// GroupVersionResources for the OLM custom resources we read (verified on OCP 4.21).
var (
	csvGVR = schema.GroupVersionResource{Group: "operators.coreos.com", Version: "v1alpha1", Resource: "clusterserviceversions"}
	subGVR = schema.GroupVersionResource{Group: "operators.coreos.com", Version: "v1alpha1", Resource: "subscriptions"}
)

// copiedFromLabel marks ClusterServiceVersions that OLM copies into other namespaces for a
// cluster-scoped operator. We keep only the original to avoid duplicate matches.
const copiedFromLabel = "olm.copiedFrom"

// propertiesAnnotation holds the olm.package property (package name + version) on a CSV.
const propertiesAnnotation = "operatorframework.io/properties"

// trailingVersionRE strips a trailing ".vX.Y.Z[...]" from a CSV name, used only as a
// fallback when the olm.package property is unavailable.
var trailingVersionRE = regexp.MustCompile(`\.v?\d+\.\d+\.\d+.*$`)

// Client implements operatorbundle.CatalogClient using in-cluster OLM data.
type Client struct {
	dyn dynamic.Interface
	reg api.RegistryClient
}

// NewClient builds a Client from a Kubernetes dynamic client (for installed CSVs and
// Subscriptions) and an operator-registry RegistryClient (for catalog candidates).
func NewClient(dyn dynamic.Interface, reg api.RegistryClient) *Client {
	return &Client{dyn: dyn, reg: reg}
}

// FindInstalledBundle resolves the installed operator bundle that ships the image with the
// given digest, by scanning installed ClusterServiceVersions. Returns (nil, nil) when no
// installed CSV references the digest.
func (c *Client) FindInstalledBundle(ctx context.Context, digest string) (*operatorbundle.Bundle, error) {
	list, err := c.dyn.Resource(csvGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "listing ClusterServiceVersions")
	}

	seen := make(map[string]struct{})
	var matches []unstructured.Unstructured
	for i := range list.Items {
		csv := list.Items[i]
		if _, copied := csv.GetLabels()[copiedFromLabel]; copied {
			continue
		}
		if _, dup := seen[csv.GetName()]; dup {
			continue
		}
		seen[csv.GetName()] = struct{}{}
		if csvShipsDigest(&csv, digest) {
			matches = append(matches, csv)
		}
	}
	if len(matches) == 0 {
		return nil, nil
	}
	// A shared base-image digest could in principle appear in more than one operator's CSV;
	// pick deterministically by name so results are stable.
	sort.Slice(matches, func(i, j int) bool { return matches[i].GetName() < matches[j].GetName() })
	csv := &matches[0]

	pkg, version := packageAndVersionFromCSV(csv)
	return &operatorbundle.Bundle{
		Package:         pkg,
		ChannelName:     c.channelForPackage(ctx, pkg),
		Version:         version,
		VersionOriginal: version,
		CSVName:         csv.GetName(),
		CSVDisplayName:  nestedString(csv, "spec", "displayName"),
		RelatedImages:   relatedImagesFromCSVSpec(csv),
	}, nil
}

// FindCandidateBundles returns all bundles of the given package known to the catalog index,
// deduplicated by CSV name across channels. The channel and sinceCreationDate parameters are
// unused: the advisor's SelectLatestPatch selects purely by semantic version, and the OLM
// index has no catalog creation date.
func (c *Client) FindCandidateBundles(ctx context.Context, pkg, _, _ string) ([]operatorbundle.Bundle, error) {
	stream, err := c.reg.ListBundles(ctx, &api.ListBundlesRequest{})
	if err != nil {
		return nil, errors.Wrap(err, "listing bundles from catalog registry")
	}

	byCSV := make(map[string]struct{})
	var out []operatorbundle.Bundle
	for {
		b, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "receiving bundle from catalog registry")
		}
		if b.GetPackageName() != pkg {
			// ListBundles streams every package in the index; filter client-side.
			continue
		}
		if _, dup := byCSV[b.GetCsvName()]; dup {
			// The same bundle appears once per channel it belongs to.
			continue
		}
		byCSV[b.GetCsvName()] = struct{}{}

		bundle, err := c.bundleFromAPI(ctx, b)
		if err != nil {
			return nil, err
		}
		out = append(out, bundle)
	}
	return out, nil
}

// bundleFromAPI maps an api.Bundle to operatorbundle.Bundle, parsing the embedded CSV JSON
// for related images. If the registry did not inline csvJson, it falls back to GetBundle.
func (c *Client) bundleFromAPI(ctx context.Context, b *api.Bundle) (operatorbundle.Bundle, error) {
	csvJSON := b.GetCsvJson()
	if csvJSON == "" {
		full, err := c.reg.GetBundle(ctx, &api.GetBundleRequest{
			PkgName: b.GetPackageName(), ChannelName: b.GetChannelName(), CsvName: b.GetCsvName(),
		})
		if err != nil {
			return operatorbundle.Bundle{}, errors.Wrapf(err, "fetching bundle %s", b.GetCsvName())
		}
		csvJSON = full.GetCsvJson()
	}

	displayName, specVersion, related := parseCSVJSON(csvJSON, b.GetBundlePath())
	version := b.GetVersion()
	if version == "" {
		version = specVersion
	}
	return operatorbundle.Bundle{
		Package:         b.GetPackageName(),
		ChannelName:     b.GetChannelName(),
		Version:         version,
		VersionOriginal: version,
		CSVName:         b.GetCsvName(),
		CSVDisplayName:  displayName,
		RelatedImages:   related,
	}, nil
}

// channelForPackage returns the subscribed channel for a package, resolved from its
// Subscription. Best-effort: an empty result still lets FindCandidateBundles enumerate the
// package (selection is by version, not channel).
func (c *Client) channelForPackage(ctx context.Context, pkg string) string {
	subs, err := c.dyn.Resource(subGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		return ""
	}
	for i := range subs.Items {
		s := &subs.Items[i]
		if nestedString(s, "spec", "name") == pkg {
			return nestedString(s, "spec", "channel")
		}
	}
	return ""
}

// csvShipsDigest reports whether any spec.relatedImages entry of the CSV references the
// given digest.
func csvShipsDigest(csv *unstructured.Unstructured, digest string) bool {
	for _, ri := range relatedImagesFromCSVSpec(csv) {
		if ri.Digest == digest {
			return true
		}
	}
	return false
}

// relatedImagesFromCSVSpec extracts spec.relatedImages from a CSV as operatorbundle
// RelatedImages, parsing the digest out of each image reference.
func relatedImagesFromCSVSpec(csv *unstructured.Unstructured) []operatorbundle.RelatedImage {
	raw, found, err := unstructured.NestedSlice(csv.Object, "spec", "relatedImages")
	if err != nil || !found {
		return nil
	}
	out := make([]operatorbundle.RelatedImage, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		image, _ := m["image"].(string)
		name, _ := m["name"].(string)
		out = append(out, operatorbundle.RelatedImage{
			Image:  image,
			Name:   name,
			Digest: digestFromRef(image),
		})
	}
	return out
}

// csvSpec is the minimal shape we parse out of a candidate bundle's CSV JSON.
type csvSpec struct {
	Spec struct {
		DisplayName   string `json:"displayName"`
		Version       string `json:"version"`
		RelatedImages []struct {
			Name  string `json:"name"`
			Image string `json:"image"`
		} `json:"relatedImages"`
	} `json:"spec"`
}

// parseCSVJSON extracts the display name, spec version and related images from a CSV JSON
// document. The bundle's own image (bundlePath) is appended to relatedImages by the catalog
// and is filtered out here.
func parseCSVJSON(csvJSON, bundlePath string) (displayName, version string, related []operatorbundle.RelatedImage) {
	if csvJSON == "" {
		return "", "", nil
	}
	var doc csvSpec
	if err := json.Unmarshal([]byte(csvJSON), &doc); err != nil {
		return "", "", nil
	}
	for _, ri := range doc.Spec.RelatedImages {
		if ri.Image == bundlePath {
			continue
		}
		related = append(related, operatorbundle.RelatedImage{
			Image:  ri.Image,
			Name:   ri.Name,
			Digest: digestFromRef(ri.Image),
		})
	}
	return doc.Spec.DisplayName, doc.Spec.Version, related
}

// packageAndVersionFromCSV derives the package name and version from a CSV, preferring the
// olm.package property annotation and falling back to spec.version and the CSV name.
func packageAndVersionFromCSV(csv *unstructured.Unstructured) (pkg, version string) {
	if raw := csv.GetAnnotations()[propertiesAnnotation]; raw != "" {
		var props struct {
			Properties []struct {
				Type  string `json:"type"`
				Value struct {
					PackageName string `json:"packageName"`
					Version     string `json:"version"`
				} `json:"value"`
			} `json:"properties"`
		}
		if json.Unmarshal([]byte(raw), &props) == nil {
			for _, p := range props.Properties {
				if p.Type == "olm.package" {
					pkg, version = p.Value.PackageName, p.Value.Version
					break
				}
			}
		}
	}
	if version == "" {
		version = nestedString(csv, "spec", "version")
	}
	if pkg == "" {
		pkg = trailingVersionRE.ReplaceAllString(csv.GetName(), "")
	}
	return pkg, version
}

// digestFromRef returns the "sha256:..." digest portion of an image reference, or "" when
// the reference is not digest-pinned.
func digestFromRef(ref string) string {
	if i := strings.Index(ref, "@"); i >= 0 {
		return ref[i+1:]
	}
	return ""
}

// nestedString is a convenience wrapper over unstructured.NestedString that drops the found
// boolean.
func nestedString(u *unstructured.Unstructured, fields ...string) string {
	s, _, _ := unstructured.NestedString(u.Object, fields...)
	return s
}
