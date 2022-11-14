package clairv4

import (
	"strings"

	gogoTypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
)

func manifestForImage(image *storage.Image) (*claircore.Manifest, error) {
	// Use claircore.ParseDigest instead of types.Digest (see pkg/images/types/digest.go)
	// to make it easier to set the (*claircore.Manifest).Hash.
	imgDigest, err := claircore.ParseDigest(utils.GetSHA(image))
	if err != nil {
		return nil, errors.Wrapf(err, "parsing image digest for image %s", image.GetName())
	}
	manifest := &claircore.Manifest{
		Hash: imgDigest,
	}

	for _, layerSHA := range image.GetMetadata().GetLayerShas() {
		layerDigest, err := claircore.ParseDigest(layerSHA)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing image layer digest for image %s", image.GetName())
		}
		// TODO
		// Layers needs to be ordered from base -> top layer, which is the same as how the metadata sorts them.
		manifest.Layers = append(manifest.Layers, &claircore.Layer{
			Hash:    layerDigest,
			URI:     "",
			Headers: nil,
		})
	}

	return manifest, nil
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
