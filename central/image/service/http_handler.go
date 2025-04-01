package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	iiStore "github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
)

type sbomHttpHandler struct {
	integration      integration.Set
	enricher         enricher.ImageEnricher
	clusterSACHelper sachelper.ClusterSacHelper
	riskManager      manager.Manager
}

var _ http.Handler = (*sbomHttpHandler)(nil)

// SBOMHandler returns a handler for get sbom http request.
func SBOMHandler(integration integration.Set, enricher enricher.ImageEnricher, clusterSACHelper sachelper.ClusterSacHelper, riskManager manager.Manager) http.Handler {
	return sbomHttpHandler{
		integration:      integration,
		enricher:         enricher,
		clusterSACHelper: clusterSACHelper,
		riskManager:      riskManager,
	}
}

func (h sbomHttpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	// Verify Scanner V4 is enabled.
	if !features.ScannerV4.Enabled() {
		httputil.WriteGRPCStyleError(w, codes.Unimplemented, errors.New("Scanner V4 is disabled. Enable Scanner V4 to generate SBOMs"))
		return
	}
	if !features.SBOMGeneration.Enabled() {
		httputil.WriteGRPCStyleError(w, codes.Unimplemented, errors.New("SBOM feature is not enabled"))
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

// enrichImage enriches the image with the given name and based on the given enrichment context.
func (h sbomHttpHandler) enrichImage(ctx context.Context, enrichmentCtx enricher.EnrichmentContext, imgName string) (*storage.Image, bool, error) {
	// forcedEnrichment is set to true when enrichImage forces an enrichment.
	forcedEnrichment := false
	img, err := enricher.EnrichImageByName(ctx, h.enricher, enrichmentCtx, imgName)
	if err != nil {
		return nil, forcedEnrichment, err
	}

	// SBOM generation requires an image to have been scanned by Scanner V4, if the existing image
	// was scanned by a different scanner we force enrichment using Scanner V4.
	scannedByV4 := h.scannedByScannerV4(img)
	if enrichmentCtx.FetchOpt != enricher.UseImageNamesRefetchCachedValues && !scannedByV4 {
		// Force scan by Scanner V4.
		addForceToEnrichmentContext(&enrichmentCtx)
		forcedEnrichment = true
		img, err = enricher.EnrichImageByName(ctx, h.enricher, enrichmentCtx, imgName)
		if err != nil {
			return nil, forcedEnrichment, err
		}
	}

	err = h.saveImage(img)
	if err != nil {
		return nil, forcedEnrichment, err
	}

	return img, forcedEnrichment, nil
}

// getSBOM generates an SBOM for the specified parameters.
func (h sbomHttpHandler) getSBOM(ctx context.Context, params apiparams.SBOMRequestBody) ([]byte, error) {
	enrichmentCtx := enricher.EnrichmentContext{
		// TODO(ROX-27920): re-introduce cluster flag when SBOM generation from delegated scans is implemented.
		// Delegable:       true,
		FetchOpt:        enricher.UseCachesIfPossible,
		ScannerTypeHint: scannerTypes.ScannerV4,
	}

	if params.Force {
		addForceToEnrichmentContext(&enrichmentCtx)
	}

	// TODO(ROX-27920): re-introduce cluster flag when SBOM generation from delegated scans is implemented.
	// if params.Cluster != "" {
	//	// The request indicates enrichment should be delegated to a specific cluster.
	//	clusterID, err := clusterUtil.GetClusterIDFromNameOrID(ctx, h.clusterSACHelper, params.Cluster, delegateScanPermissions)
	//	if err != nil {
	//		return nil, err
	//	}
	//	enrichmentCtx.ClusterID = clusterID
	// }

	img, alreadyForcedEnrichment, err := h.enrichImage(ctx, enrichmentCtx, params.ImageName)
	if err != nil {
		return nil, err
	}

	// Verify the Index Report exists. If it doesn't, force image enrichment using Scanner V4.
	scannerV4, err := h.getScannerV4SBOMIntegration()
	if err != nil {
		return nil, err
	}

	sbom, found, err := scannerV4.GetSBOM(img)
	if err != nil {
		return nil, err
	}

	if !found && !params.Force && !alreadyForcedEnrichment {
		// Since the Index Report for image does not exist, force scan by Scanner V4.
		addForceToEnrichmentContext(&enrichmentCtx)
		img, err = enricher.EnrichImageByName(ctx, h.enricher, enrichmentCtx, params.ImageName)
		if err != nil {
			return nil, err
		}

		err = h.saveImage(img)
		if err != nil {
			return nil, err
		}

		sbom, _, err = scannerV4.GetSBOM(img)
		if err != nil {
			return nil, err
		}
	}

	return sbom, nil
}

func addForceToEnrichmentContext(enrichmentCtx *enricher.EnrichmentContext) {
	enrichmentCtx.FetchOpt = enricher.UseImageNamesRefetchCachedValues
}

// getScannerV4SBOMIntegration returns the SBOM interface of Scanner V4.
func (h sbomHttpHandler) getScannerV4SBOMIntegration() (scannerTypes.SBOMer, error) {
	scanners := h.integration.ScannerSet()
	for _, scanner := range scanners.GetAll() {
		if scanner.GetScanner().Type() == scannerTypes.ScannerV4 {
			if scannerv4, ok := scanner.GetScanner().(scannerTypes.SBOMer); ok {
				return scannerv4, nil
			}
		}
	}
	return nil, errors.New("Scanner V4 integration not found")
}

// scannedByScannerV4 checks if image is scanned by Scanner V4.
func (h sbomHttpHandler) scannedByScannerV4(img *storage.Image) bool {
	return img.GetScan().GetDataSource().GetId() == iiStore.DefaultScannerV4Integration.GetId()
}

// saveImage saves the image to Central's database.
func (h sbomHttpHandler) saveImage(img *storage.Image) error {
	img.Id = utils.GetSHA(img)
	if img.GetId() == "" {
		return nil
	}

	if err := h.riskManager.CalculateRiskAndUpsertImage(img); err != nil {
		log.Errorw("Error upserting image", logging.ImageName(img.GetName().GetFullName()), logging.Err(err))
		return fmt.Errorf("saving image: %w", err)
	}
	return nil
}
