package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	clusterUtil "github.com/stackrox/rox/central/cluster/util"
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

// SBOMHandler returns a handler for get sbom http request
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
	// verify scanner v4 is enabled
	if !features.ScannerV4.Enabled() {
		httputil.WriteGRPCStyleError(w, codes.Unimplemented, errors.New("Scanner V4 is disabled. Enable Scanner V4 to generate SBOMs"))
		return
	}
	if !features.SBOMGeneration.Enabled() {
		httputil.WriteGRPCStyleError(w, codes.Unimplemented, errors.New("SBOM feature is not enabled"))
		return
	}
	var params apiparams.SbomRequestBody
	sbomGenMaxReqSizeBytes := env.SBOMGenerationMaxReqSizeBytes.IntegerSetting()
	// timeout api after 10 minutes
	lr := io.LimitReader(r.Body, int64(sbomGenMaxReqSizeBytes))
	err := json.NewDecoder(lr).Decode(&params)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Wrap(err, "decoding json request body"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), env.ScanTimeout.DurationSetting())
	defer cancel()
	bytes, err := h.getSbom(ctx, params)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "generating SBOM"))
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

// enrichImage enriches the image with the given name and based on the given enrichment context
func (h sbomHttpHandler) enrichImage(ctx context.Context, enrichmentCtx enricher.EnrichmentContext, imgName string) (*storage.Image, bool, error) {

	// forcedEnrichment is set to true when enrichImage forces an enrichment.
	forceEnrichment := false
	img, err := enricher.EnrichImageByName(ctx, h.enricher, enrichmentCtx, imgName)
	if err != nil {
		return nil, forceEnrichment, err
	}
	// verify that image is scanned by scanner v4 if not force enrichment using scanner v4
	scannedByV4 := h.scannedByScannerv4(img)

	if enrichmentCtx.FetchOpt != enricher.UseImageNamesRefetchCachedValues && !scannedByV4 {
		// force scan by scanner v4
		addForceToEnrichmentContext(&enrichmentCtx)
		forceEnrichment = true
		img, err = enricher.EnrichImageByName(ctx, h.enricher, enrichmentCtx, imgName)
		if err != nil {
			return nil, forceEnrichment, err
		}
	}

	// Save the image
	img.Id = utils.GetSHA(img)
	if img.GetId() != "" {
		if err := h.saveImage(img); err != nil {
			return nil, forceEnrichment, err
		}
	}

	return img, forceEnrichment, nil
}

// getSbom generates an SBOM for the specified parameters
func (h sbomHttpHandler) getSbom(ctx context.Context, params apiparams.SbomRequestBody) ([]byte, error) {
	enrichmentCtx := enricher.EnrichmentContext{
		FetchOpt:        enricher.UseCachesIfPossible,
		Delegable:       true,
		ScannerTypeHint: scannerTypes.ScannerV4,
	}

	if params.Force {
		addForceToEnrichmentContext(&enrichmentCtx)
	}

	if params.Cluster != "" {
		// The request indicates enrichment should be delegated to a specific cluster.
		clusterID, err := clusterUtil.GetClusterIDFromNameOrID(ctx, h.clusterSACHelper, params.Cluster, delegateScanPermissions)
		if err != nil {
			return nil, err
		}
		enrichmentCtx.ClusterID = clusterID
	}
	img, alreadyForcedEnrichment, err := h.enrichImage(ctx, enrichmentCtx, params.ImageName)
	if err != nil {
		return nil, err
	}
	// verify that index report exists. if not force image enrichment using scanner v4
	scannerV4, err := h.getScannerV4SBOMIntegration()
	if err != nil {
		return nil, err
	}
	sbom, found, err := scannerV4.GetSBOM(img)

	if err != nil {
		return nil, err
	}
	if !found && !params.Force && !alreadyForcedEnrichment {
		// since index report for image does not exist force scan by scanner v4
		addForceToEnrichmentContext(&enrichmentCtx)
		_, err = enricher.EnrichImageByName(ctx, h.enricher, enrichmentCtx, params.ImageName)
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

// getScannerV4SBOMIntegration returns the SBOM interface of scanner v4
func (h sbomHttpHandler) getScannerV4SBOMIntegration() (scannerTypes.SBOMer, error) {
	scanners := h.integration.ScannerSet()
	for _, scanner := range scanners.GetAll() {
		if scanner.GetScanner().Type() == scannerTypes.ScannerV4 {
			if scannerv4, ok := scanner.GetScanner().(scannerTypes.SBOMer); ok {
				return scannerv4, nil
			}
		}
	}
	return nil, errors.New("Scanner v4 integration not found")
}

// scannedByScannerv4 checks if image is scanned by scanner v4
func (h sbomHttpHandler) scannedByScannerv4(img *storage.Image) bool {
	return img.GetScan().GetDataSource().GetId() == iiStore.DefaultScannerV4Integration.GetId()
}

// saveImage saves the image to the scanner database
func (h sbomHttpHandler) saveImage(img *storage.Image) error {
	if err := h.riskManager.CalculateRiskAndUpsertImage(img); err != nil {
		log.Errorw("Error upserting image", logging.ImageName(img.GetName().GetFullName()), logging.Err(err))
		return err
	}
	return nil
}
