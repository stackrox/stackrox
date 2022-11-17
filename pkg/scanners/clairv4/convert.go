package clairv4

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	gogoTypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/utils"
)

// manifestForImage returns a ClairCore image manifest for the given image.
func manifestForImage(rc *registryTypes.Config, image *storage.Image) (*claircore.Manifest, error) {
	// Use claircore.ParseDigest instead of types.Digest (see pkg/images/types/digest.go)
	// to make it easier to set the (*claircore.Manifest).Hash.
	imgDigest, err := claircore.ParseDigest(imageUtils.GetSHA(image))
	if err != nil {
		return nil, errors.Wrapf(err, "parsing image digest for image %s", image.GetName())
	}
	manifest := &claircore.Manifest{
		Hash: imgDigest,
	}

	// Create the image registry client once to be used for each layer.
	registryClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: rc.Insecure,
			},
			Proxy: proxy.FromConfig(),
			// The following values are taken from http.DefaultTransport as of go1.19.3.
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		// Specify a different timeout from requestTimeout to decouple this from Clair v4 requests.
		Timeout: 30 * time.Second,
	}
	url := strings.TrimRight(rc.URL, "/")

	imgName := image.GetName().GetFullName()
	for _, layerSHA := range image.GetMetadata().GetLayerShas() {
		layerDigest, err := claircore.ParseDigest(layerSHA)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing image layer digest for image %s", imgName)
		}

		uri, header, err := getLayerURIAndHeader(registryClient, url, image)
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
func getLayerURIAndHeader(client *http.Client, url string, image *storage.Image) (string, http.Header, error) {
	imgName := image.GetName().GetFullName()

	path := fmt.Sprintf("/v2/%s/blobs/%s", image.GetName().GetRegistry(), imageUtils.Reference(image))
	req, err := http.NewRequest(http.MethodGet, url+path, nil)
	if err != nil {
		return "", nil, errors.Wrapf(err, "creating image pull request for imge %s", imgName)
	}
	// We don't actually want the layer. We just want the headers.
	req.Header.Add("Range", "bytes=0-0")
	res, err := client.Do(req)
	if err != nil {
		return "", nil, errors.Wrapf(err, "fetching image layer for image %s", imgName)
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
