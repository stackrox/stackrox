package clairv4

import (
	"fmt"
	"net/http"
	"strings"

	gogoTypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/stackrox/rox/generated/storage"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/utils"
)

// manifestForImage returns a ClairCore image manifest for the given image.
func manifestForImage(registry registryTypes.Registry, image *storage.Image) (*claircore.Manifest, error) {
	// Ensure this exists before bothering to continue.
	cfg := registry.Config()
	if cfg == nil {
		return nil, errors.New("registry configuration does not exist")
	}

	// Use claircore.ParseDigest instead of types.Digest (see pkg/images/types/digest.go)
	// to make it easier to set the (*claircore.Manifest).Hash.
	imgDigest, err := claircore.ParseDigest(imageUtils.GetSHA(image))
	if err != nil {
		return nil, errors.Wrap(err, "parsing image digest")
	}
	manifest := &claircore.Manifest{
		Hash: imgDigest,
	}

	for _, layerSHA := range image.GetMetadata().GetLayerShas() {
		layerDigest, err := claircore.ParseDigest(layerSHA)
		if err != nil {
			return nil, errors.Wrap(err, "parsing image layer digest")
		}

		uri, header, err := getLayerURIAndHeader(registry.HTTPClient(), cfg.URL, image)
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

// getLayerURIAndHeader is based on the clairctl v4.5.0 implementation for creating manifests:
// https://github.com/quay/clair/blob/v4.5.0/cmd/clairctl/manifest.go#L76.
// It is assumed the given client handles auth.
func getLayerURIAndHeader(client *http.Client, url string, image *storage.Image) (string, http.Header, error) {
	path := fmt.Sprintf("/v2/%s/blobs/%s", image.GetName().GetRemote(), imageUtils.Reference(image))
	req, err := http.NewRequest(http.MethodGet, url+path, nil)
	if err != nil {
		return "", nil, errors.Wrap(err, "creating image pull request")
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

func imageScanFromReport(report *claircore.VulnerabilityReport) *storage.ImageScan {
	scan := &storage.ImageScan{
		ScanTime:        gogoTypes.TimestampNow(),
		Components:      getComponents(report),
		OperatingSystem: getOS(report),
	}

	if scan.GetOperatingSystem() == "unknown" {
		scan.Notes = append(scan.Notes, storage.ImageScan_OS_UNAVAILABLE)
	}

	return scan
}

func getComponents(report *claircore.VulnerabilityReport) []*storage.EmbeddedImageScanComponent {
	components := make([]*storage.EmbeddedImageScanComponent, 0, len(report.PackageVulnerabilities))
	for pkgID, vulnIDs := range report.PackageVulnerabilities {
		pkg := report.Packages[pkgID]
		components = append(components, &storage.EmbeddedImageScanComponent{
			Name:    pkg.Name,
			Version: pkg.Version,
			Vulns:   getVulns(report.Vulnerabilities, vulnIDs),
		})
	}

	return components
}

func getVulns(vulnerabilities map[string]*claircore.Vulnerability, ids []string) []*storage.EmbeddedVulnerability {
	vulns := make([]*storage.EmbeddedVulnerability, 0, len(ids))
	for _, id := range ids {
		ccVuln := vulnerabilities[id]
		// Ignore the error, as publishedTime will just be `nil` if the given time is invalid.
		publishedTime, _ := gogoTypes.TimestampProto(ccVuln.Issued)
		vuln := &storage.EmbeddedVulnerability{
			Cve:               ccVuln.Name,
			Summary:           ccVuln.Description,
			Link:              getLink(ccVuln.Links),
			PublishedOn:       publishedTime,
			VulnerabilityType: storage.EmbeddedVulnerability_IMAGE_VULNERABILITY,
			Severity:          normalizeSeverity(ccVuln.NormalizedSeverity),
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

// getLink returns a single link to use for the vulnerability,
// Clair v4 can possibly send over multiple, space-separated links
// for a single vulnerability.
func getLink(links string) string {
	link, _, _ := strings.Cut(links, " ")
	return link
}

func normalizeSeverity(severity claircore.Severity) storage.VulnerabilitySeverity {
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

// getOS retrieves the OS name:version for the image represented by the given
// vulnerability report.
// If there are zero known distributions for the image or if there are multiple distributions,
// return "unknown", as StackRox only supports a single base-OS at this time.
func getOS(report *claircore.VulnerabilityReport) string {
	if len(report.Distributions) == 1 {
		for _, dist := range report.Distributions {
			return dist.VersionCodeName + ":" + dist.VersionID
		}
	}

	return "unknown"
}
