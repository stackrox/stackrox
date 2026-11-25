package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	clusterUtil "github.com/stackrox/rox/central/cluster/util"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scanners"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	searchPkg "github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
)

type sbomGenHttpHandler struct {
	integration      integration.Set
	enricher         enricher.ImageEnricher
	enricherV2       enricher.ImageEnricherV2
	clusterSACHelper sachelper.ClusterSacHelper
	riskManager      manager.Manager
	// images is the image datastore mapper, which dispatches to the V1 or V2
	// datastore depending on the FlattenImageData feature and the key format.
	images imageDatastore.DataStore
}

var _ http.Handler = (*sbomGenHttpHandler)(nil)

// SBOMHandler returns a handler for get sbom http request.
func SBOMGenHandler(integration integration.Set, enricher enricher.ImageEnricher, enricherV2 enricher.ImageEnricherV2, clusterSACHelper sachelper.ClusterSacHelper, riskManager manager.Manager, images imageDatastore.DataStore) http.Handler {
	return sbomGenHttpHandler{
		integration:      integration,
		enricher:         enricher,
		enricherV2:       enricherV2,
		clusterSACHelper: clusterSACHelper,
		riskManager:      riskManager,
		images:           images,
	}
}

func (h sbomGenHttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// Verify Scanner V4 is enabled.
	if !features.ScannerV4.Enabled() {
		httputil.WriteGRPCStyleError(w, codes.Unimplemented, errors.New("Scanner V4 is disabled. Enable Scanner V4 to generate SBOMs"))
		return
	}

	var params apiparams.SBOMRequestBody
	sbomGenMaxReqSizeBytes := env.SBOMGenerationMaxReqSizeBytes.IntegerSetting()
	lr := io.LimitReader(r.Body, int64(sbomGenMaxReqSizeBytes))
	err := json.NewDecoder(lr).Decode(&params)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Wrap(err, "decoding json request body"))
		return
	}
	params.ImageName = strings.TrimSpace(params.ImageName)
	params.Digest = strings.TrimSpace(params.Digest)

	ctx, cancel := context.WithTimeout(r.Context(), env.ScanTimeout.DurationSetting())
	defer cancel()
	bytes, err := h.getSBOM(ctx, params)
	if err != nil {
		// Using WriteError instead of WriteGRPCStyleError so that the HTTP status
		// is derived from the error type.
		httputil.WriteError(w, errors.Wrap(err, "generating SBOM"))
		return
	}
	if len(bytes) == 0 {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.New("SBOM not found for the image"))
		return
	}

	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s.%s", zip.GetSafeFilename(params.ImageName), "json"))
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Content-Length", fmt.Sprint(len(bytes)))
	_, _ = w.Write(bytes)
}

// enrichWithModelSwitch enriches an image by name, returning both V1 and V2 models based on the feature flag.
func (h sbomGenHttpHandler) enrichWithModelSwitch(
	ctx context.Context,
	enrichmentCtx enricher.EnrichmentContext,
	ci *storage.ContainerImage,
) (img *storage.Image, imgV2 *storage.ImageV2, err error) {
	if features.FlattenImageData.Enabled() {
		imgV2 := types.ToImageV2(ci)

		enrichmentResult, err := h.enricherV2.EnrichImage(ctx, enrichmentCtx, imgV2)
		if err = errorOrNotScanned(enrichmentResult, err); err != nil {
			return nil, nil, err
		}

		img = utils.ConvertToV1(imgV2)
		return img, imgV2, nil
	}

	img = types.ToImage(ci)

	enrichmentResult, err := h.enricher.EnrichImage(ctx, enrichmentCtx, img)
	if err = errorOrNotScanned(enrichmentResult, err); err != nil {
		return nil, nil, err
	}
	return img, nil, nil
}

func (h sbomGenHttpHandler) cachedImage(ctx context.Context, ci *storage.ContainerImage) (*storage.Image, bool, error) {
	if features.FlattenImageData.Enabled() {
		// Exact UUID fast path, used only when that row has a usable Scanner V4 scan;
		// otherwise the digest search below may find another row for the same digest.
		if idV2 := ci.GetIdV2(); idV2 != "" {
			if img, found, err := h.images.GetImage(ctx, idV2); err != nil || (found && h.scannedByScannerV4(img.GetScan())) {
				return img, found, err
			}
		}
		// Digest-based scoped search: finds rows for the digest regardless of the request's
		// name form (tag, inline digest, or tagless repo@digest) and never interprets the
		// value as a row ID, so a non-digest string in the request cannot fetch an
		// unrelated image.
		images, err := h.images.SearchRawImages(ctx, searchPkg.NewQueryBuilder().
			AddExactMatches(searchPkg.ImageSHA, ci.GetId()).ProtoQuery())
		if err != nil {
			return nil, false, err
		}
		for _, img := range images {
			if h.scannedByScannerV4(img.GetScan()) {
				return img, true, nil
			}
		}
		return nil, false, nil
	}
	return h.images.GetImage(ctx, ci.GetId())
}

// errorOrNotScanned will return an error if err is populated or the enrichmentResult indicates a scan
// did not occur.
func errorOrNotScanned(enrichmentResult enricher.EnrichmentResult, err error) error {
	if err != nil {
		return err
	}

	if enrichmentResult.ScanResult != enricher.ScanSucceeded {
		return errors.New("scan could not be completed, please check that an applicable registry is integrated")
	}

	return nil
}

// getSBOM generates an SBOM for the specified parameters.
func (h sbomGenHttpHandler) getSBOM(ctx context.Context, params apiparams.SBOMRequestBody) ([]byte, error) {
	// A scoped miss can mean access was denied. Do not retry via the enricher's privileged scan cache.
	enrichmentCtx := enricher.EnrichmentContext{
		Delegable:       true,
		FetchOpt:        enricher.UseImageNamesRefetchCachedValues,
		ScannerTypeHint: scannerTypes.ScannerV4,
	}

	if params.Cluster != "" {
		// The request indicates enrichment should be delegated to a specific cluster.
		clusterID, err := clusterUtil.GetClusterIDFromNameOrID(ctx, h.clusterSACHelper, params.Cluster, delegateScanPermissions)
		if err != nil {
			return nil, err
		}
		enrichmentCtx.ClusterID = clusterID
		enrichmentCtx.Namespace = params.Namespace
	}

	// Build a reference to the image that will be passed to the enricher.
	ci, err := containerImage(params)
	if err != nil {
		return nil, err
	}

	scannerV4, err := h.getScannerV4SBOMIntegration()
	if err != nil {
		return nil, err
	}

	if !params.Force && ci.GetId() != "" {
		img, found, err := h.cachedImage(ctx, ci)
		if err != nil {
			// A failed cache read must not block SBOM generation when fresh enrichment
			// could still succeed. Fall through without using any data from the failed read.
			log.Errorw("Reading cached image for SBOM generation", logging.ImageName(params.ImageName), logging.Err(err))
		} else if found && h.scannedByScannerV4(img.GetScan()) {
			// The SBOM is generated from the stored scan's index report, so stale pull
			// metadata does not disqualify the cached image.
			sbom, found, err := scannerV4.GetSBOM(ctx, img)
			if err != nil || found {
				return sbom, err
			}
		}
	}
	// Do not start expensive enrichment work for a request that is already gone.
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	img, imgV2, err := h.enrichWithModelSwitch(ctx, enrichmentCtx, ci)
	if err != nil {
		sendAdminEvent(err, params.ImageName)
		return nil, err
	}
	// shortcut: enrichment resolves the image name at scan time, so a moved tag can return
	// an image with a different digest than requested; a digest guard cannot distinguish
	// this from legitimate multi-arch resolution (metadata carries the platform manifest
	// digest, not the requested index digest), so the digest is not checked here.
	if err := h.saveImage(img, imgV2); err != nil {
		return nil, err
	}
	sbom, _, err := scannerV4.GetSBOM(ctx, img)
	return sbom, err
}

// containerImage will return a populated container image with fields derived from params.
func containerImage(params apiparams.SBOMRequestBody) (*storage.ContainerImage, error) {
	ci, err := utils.GenerateImageFromString(params.ImageName)
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	// Populate the image IDs if a digest was provided and the IDs
	// have not already been derived from the image name.
	if params.Digest != "" {
		if ci.GetId() == "" {
			ci.Id = params.Digest
		}

		if ci.GetIdV2() == "" && features.FlattenImageData.Enabled() {
			ci.IdV2 = utils.NewImageV2ID(ci.GetName(), params.Digest)
		}
	}

	return ci, nil
}

// sendAdminEvent will log and send an admin event for the enrichment failure. An admin
// event is sent when using `log.Errorw()`.
func sendAdminEvent(err error, imageName string) {
	if env.AdministrationEventsAdHocScans.BooleanSetting() {
		log.Errorw("Enriching image",
			logging.ImageName(imageName),
			logging.Err(err),
			logging.Bool("ad_hoc", true),
		)
	}
}

// getScannerV4SBOMIntegration returns the SBOM interface of Scanner V4.
func (h sbomGenHttpHandler) getScannerV4SBOMIntegration() (scannerTypes.SBOMer, error) {
	sbomer, _, err := getScannerV4SBOMIntegration(h.integration.ScannerSet())
	return sbomer, err
}

// scannedByScannerV4 checks if image is scanned by Scanner V4.
func (h sbomGenHttpHandler) scannedByScannerV4(scan *storage.ImageScan) bool {
	return scan.GetDataSource().GetId() == iiStore.DefaultScannerV4Integration.GetId()
}

// saveImage saves the image to Central's database.
func (h sbomGenHttpHandler) saveImage(img *storage.Image, imgV2 *storage.ImageV2) error {
	if features.FlattenImageData.Enabled() {
		return h.saveImageV2(imgV2)
	}
	return h.saveImageV1(img)
}

// saveImageV1 saves an Image V1 to Central's database.
func (h sbomGenHttpHandler) saveImageV1(img *storage.Image) error {
	img.Id = utils.GetSHA(img)
	if img.GetId() == "" {
		return nil
	}

	if err := h.riskManager.CalculateRiskAndUpsertImage(img); err != nil {
		log.Errorw("Error upserting image", logging.ImageName(img.GetName().GetFullName()), logging.ImageID(img.GetId()), logging.Err(err))
		return fmt.Errorf("saving image: %w", err)
	}
	return nil
}

// saveImageV2 saves an Image V2 to Central's database.
func (h sbomGenHttpHandler) saveImageV2(imgV2 *storage.ImageV2) error {
	if imgV2 == nil {
		return errors.New("nil images cannot be saved")
	}
	imgV2.Digest = utils.GetSHAV2(imgV2)
	var err error
	imgV2.Id, err = utils.GetImageV2ID(imgV2)
	if err != nil {
		return fmt.Errorf("getting image V2 ID: %w", err)
	}
	if imgV2.GetId() == "" {
		return nil
	}

	if err := h.riskManager.CalculateRiskAndUpsertImageV2(imgV2); err != nil {
		log.Errorw("Error upserting image", logging.ImageName(imgV2.GetName().GetFullName()), logging.ImageID(imgV2.GetId()), logging.Err(err))
		return fmt.Errorf("saving image: %w", err)
	}
	return nil
}

func getScannerV4SBOMIntegration(scanners scanners.Set) (scannerTypes.SBOMer, *storage.DataSource, error) {
	for _, scanner := range scanners.GetAll() {
		if scanner.GetScanner().Type() == scannerTypes.ScannerV4 {
			if scannerv4, ok := scanner.GetScanner().(scannerTypes.SBOMer); ok {
				return scannerv4, scanner.DataSource(), nil
			}
		}
	}
	return nil, nil, errors.New("Scanner V4 integration not found")
}
