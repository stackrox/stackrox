package clairv4

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/quay/claircore/sbom/spdx"
	"github.com/stackrox/rox/generated/storage"
	imageutils "github.com/stackrox/rox/pkg/images/utils"
	registrytypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/scanners/scannerv4"
	"github.com/stackrox/rox/pkg/scannerv4/mappers"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
)

// manifest returns a ClairCore image manifest for the given image.
func manifest(registry registrytypes.Registry, image *storage.Image) (*claircore.Manifest, error) {
	cfg := registry.Config(context.Background())
	if cfg == nil {
		return nil, errors.Errorf("registry configuration does not exist for registry %s", registry.Name())
	}

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

		manifest.Layers = append(manifest.Layers, &claircore.Layer{
			Hash:    layerDigest,
			URI:     uri,
			Headers: header,
		})
	}

	return manifest, nil
}

// fetchLayerURIAndHeader is based on the clairctl v4.5.0 implementation for creating manifests.
func fetchLayerURIAndHeader(client *http.Client, url, repository, digest string) (string, http.Header, error) {
	path := fmt.Sprintf("/v2/%s/blobs/%s", repository, digest)
	req, err := http.NewRequest(http.MethodGet, url+path, nil)
	if err != nil {
		return "", nil, errors.Wrap(err, "creating image layer pull request")
	}
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

// clairV4ScannerVersion identifies scans performed by the Clair V4 integration.
const clairV4ScannerVersion = "clairv4"

// imageScan converts a claircore report to storage.ImageScan using Scanner V4's
// mappers and converter. This ensures Clair V4 and Scanner V4 produce identical results.
func imageScan(ctx context.Context, metadata *storage.ImageMetadata, report *claircore.VulnerabilityReport) (*storage.ImageScan, error) {
	v4Report, err := mappers.ToProtoV4VulnerabilityReport(ctx, report)
	if err != nil {
		return nil, errors.Wrap(err, "converting to v4 report")
	}

	return scannerv4.ImageScan(metadata, v4Report, clairV4ScannerVersion), nil
}

func encodeSPDX(ir *claircore.IndexReport, image *storage.Image) ([]byte, error) {
	imgName := image.GetName()
	name := imgName.GetRegistry() + "/" + imgName.GetRemote()
	namespace := "https://" + name + "-" + uuid.NewV4().String()

	encoder := spdx.NewDefaultEncoder(
		spdx.WithDocumentName(name),
		spdx.WithDocumentNamespace(namespace),
	)
	encoder.Creators = append(encoder.Creators, spdx.Creator{Creator: "clair-v4", CreatorType: "Tool"})
	encoder.Version = spdx.V2_3
	encoder.Format = spdx.FormatJSON

	var buf bytes.Buffer
	if err := encoder.Encode(context.Background(), &buf, ir); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
