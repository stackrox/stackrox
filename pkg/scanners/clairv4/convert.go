package clairv4

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	gogotypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/stackrox/rox/generated/storage"
	imageutils "github.com/stackrox/rox/pkg/images/utils"
	registrytypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

var vulnNamePattern = regexp.MustCompile(`((CVE|ALAS|DSA)-\d{4}-\d+)|((RHSA|RHBA|RHEA)-\d{4}:\d+)`)

// manifest returns a ClairCore image manifest for the given image.
func manifest(registry registrytypes.Registry, image *storage.Image) (*claircore.Manifest, error) {
	// Ensure this exists before bothering to continue.
	cfg := registry.Config()
	if cfg == nil {
		return nil, errors.Errorf("registry configuration does not exist for registry %s", registry.Name())
	}

	// Use claircore.ParseDigest instead of types.Digest (see pkg/images/types/digest.go)
	// to make it easier to set the (*claircore.Manifest).Hash.
	imgDigest, err := claircore.ParseDigest(imageutils.GetSHA(image))
	if err != nil {
		return nil, errors.Wrap(err, "parsing image digest")
	}
	manifest := &claircore.Manifest{
		Hash: imgDigest,
	}

	client := registry.HTTPClient()
	remote := image.GetName().GetRemote()
	for _, layerSHA := range image.GetMetadata().GetLayerShas() {
		layerDigest, err := claircore.ParseDigest(layerSHA)
		if err != nil {
			return nil, errors.Wrap(err, "parsing image layer digest")
		}

		uri, header, err := fetchLayerURIAndHeader(client, cfg.URL, remote, layerDigest.String())
		if err != nil {
			return nil, err
		}

		// Layers needs to be ordered from base -> top layer, which is the same as how the metadata sorts them.
		manifest.Layers = append(manifest.Layers, &claircore.Layer{
			Hash:    layerDigest,
			URI:     uri,
			Headers: header,
		})
	}

	return manifest, nil
}

// fetchLayerURIAndHeader is based on the clairctl v4.5.0 implementation for creating manifests:
// https://github.com/quay/clair/blob/v4.5.0/cmd/clairctl/manifest.go#L76.
// It is assumed the given client handles auth.
func fetchLayerURIAndHeader(client *http.Client, url, repository, digest string) (string, http.Header, error) {
	path := fmt.Sprintf("/v2/%s/blobs/%s", repository, digest)
	req, err := http.NewRequest(http.MethodGet, url+path, nil)
	if err != nil {
		return "", nil, errors.Wrap(err, "creating image layer pull request")
	}
	// We don't actually want the layer. We just want the headers.
	req.Header.Add("Range", "bytes=0-0")
	res, err := client.Do(req)
	if err != nil {
		return "", nil, errors.Wrap(err, "fetching image layer")
	}
	utils.IgnoreError(res.Body.Close)

	res.Request.Header.Del("User-Agent")
	res.Request.Header.Del("Range")

	return res.Request.URL.String(), res.Request.Header, nil
}

// imageScan converts the given report to an image scan.
func imageScan(report *claircore.VulnerabilityReport) *storage.ImageScan {
	scan := &storage.ImageScan{
		ScanTime:        gogotypes.TimestampNow(),
		Components:      components(report),
		OperatingSystem: os(report),
	}

	if scan.GetOperatingSystem() == "unknown" {
		scan.Notes = append(scan.Notes, storage.ImageScan_OS_UNAVAILABLE)
	}

	return scan
}

func components(report *claircore.VulnerabilityReport) []*storage.EmbeddedImageScanComponent {
	components := make([]*storage.EmbeddedImageScanComponent, 0, len(report.PackageVulnerabilities))
	for id, pkg := range report.Packages {
		vulnIDs := report.PackageVulnerabilities[id]
		component := &storage.EmbeddedImageScanComponent{
			Name:    pkg.Name,
			Version: pkg.Version,
			Vulns:   vulnerabilities(report.Vulnerabilities, vulnIDs),
		}

		components = append(components, component)
	}

	return components
}

// vulnerabilities returns a list of vulnerabilities from the given map indexed by the given ids.
func vulnerabilities(vulnerabilities map[string]*claircore.Vulnerability, ids []string) []*storage.EmbeddedVulnerability {
	// vulns will have at most len(ids) entries.
	vulns := make([]*storage.EmbeddedVulnerability, 0, len(ids))
	uniqueVulns := set.NewStringSet()
	for _, id := range ids {
		ccVuln := vulnerabilities[id]
		if !uniqueVulns.Add(ccVuln.Name) {
			// Already added this vulnerability, so ignore it.
			continue
		}

		var publishedTime *gogotypes.Timestamp
		if !ccVuln.Issued.IsZero() {
			// Ignore the error, as publishedTime will just be `nil` if the given time is invalid.
			publishedTime, _ = gogotypes.TimestampProto(ccVuln.Issued)
		}
		vuln := &storage.EmbeddedVulnerability{
			Cve:               vulnName(ccVuln.Name),
			Summary:           ccVuln.Description,
			Link:              link(ccVuln.Links),
			PublishedOn:       publishedTime,
			VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			Severity:          normalizedSeverity(ccVuln.NormalizedSeverity),
		}

		if ccVuln.FixedInVersion != "" {
			vuln.SetFixedBy = &storage.EmbeddedVulnerability_FixedBy{
				FixedBy: ccVuln.FixedInVersion,
			}
		}

		vulns = append(vulns, vuln)
	}

	return vulns
}

func vulnName(original string) string {
	vulnID := vulnNamePattern.FindString(original)
	if vulnID == "" {
		return original
	}

	return vulnID
}

// link returns a single link to use for the vulnerability,
// Clair v4 can possibly send over multiple, space-separated links
// for a single vulnerability.
func link(links string) string {
	l, _, _ := strings.Cut(links, " ")
	return l
}

func normalizedSeverity(severity claircore.Severity) storage.VulnerabilitySeverity {
	switch severity {
	case claircore.Negligible, claircore.Low:
		return storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	case claircore.Medium:
		return storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY
	case claircore.High:
		return storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY
	case claircore.Critical:
		return storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	default:
		return storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	}
}

// os retrieves the OS name:version for the image represented by the given
// vulnerability report.
// If there are zero known distributions for the image or if there are multiple distributions,
// return "unknown", as StackRox only supports a single base-OS at this time.
func os(report *claircore.VulnerabilityReport) string {
	if len(report.Distributions) == 1 {
		for _, dist := range report.Distributions {
			return dist.DID + ":" + dist.VersionID
		}
	}

	return "unknown"
}
