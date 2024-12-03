package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	clusterUtil "github.com/stackrox/rox/central/cluster/util"
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	scannerV4 "github.com/stackrox/rox/pkg/scanners/scannerv4"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"google.golang.org/grpc/codes"
)

const (
	NumberBytes = 100 * 1024
	timeout     = 10 * time.Minute
)

type sbomRequestBody struct {
	Cluster   string `json:"cluster"`
	ImageName string `json:"image_name"`
	Force     bool   `json:"force"`
}

type sbomHttpHandler struct {
	integration      integration.Set
	enricher         enricher.ImageEnricher
	clusterSACHelper sachelper.ClusterSacHelper
	riskManager      manager.Manager
}

var _ http.Handler = (*sbomHttpHandler)(nil)

// Handler returns a handler for get sbom http request
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
		httputil.WriteGRPCStyleError(w, codes.NotFound, errors.New("Scanner v4 is not enabled. Enabled scanner v4 to get SBOMs"))
		return
	}
	if !features.SBOMGeneration.Enabled() {
		httputil.WriteGRPCStyleError(w, codes.NotFound, errors.New("SBOM feature is not enabled"))
		return
	}
	var params sbomRequestBody
	reqSizeLimit := NumberBytes
	reqSizeBytes := os.Getenv("ROX_SBOM_API_REQ_SIZE")
	if reqSizeBytes != "" {
		reqSizeLimitInt, err := strconv.Atoi(reqSizeBytes)
		if err == nil {
			reqSizeLimit = reqSizeLimitInt
		}
	}
	// timeout api after 10 minutes
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeout))
	defer cancel()
	log.Infof("request size is %d", reqSizeLimit)
	lr := io.LimitReader(r.Body, int64(reqSizeLimit))
	err := json.NewDecoder(lr).Decode(&params)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, errors.Wrap(err, "decoding json request body"))
		return
	}
	bytes, err := h.getSbom(ctx, params)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "error generating SBOM"))
		return
	}

	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", "attachment; sbom.json")
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Content-Length", fmt.Sprint(len(bytes)))
	_, _ = w.Write(bytes)
}

func (h sbomHttpHandler) getScannerV4Integration(img *storage.Image) (bool, *scannerV4.Scannerv4) {
	scanners := h.integration.ScannerSet()
	for _, scanner := range scanners.GetAll() {
		if scanner.GetScanner().Type() == scannerTypes.ScannerV4 {
			if scanner.DataSource().GetId() == img.GetMetadata().GetDataSource().GetId() {
				if scannerv4, ok := scanner.GetScanner().(*scannerV4.Scannerv4); ok {
					return true, scannerv4
				}
			}
		}
	}
	return false, nil
}

func (h sbomHttpHandler) enrichImage(ctx context.Context, enrichmentCtx enricher.EnrichmentContext, imgName string) (*storage.Image, error) {

	img, err := enricher.EnrichImageByName(ctx, h.enricher, enrichmentCtx, imgName)
	if err != nil {
		return nil, err
	}
	//verify that image is scanned by scanner v4 if not force enrichment using scanner v4
	scannedByV4, _ := h.getScannerV4Integration(img)

	if enrichmentCtx.FetchOpt != enricher.UseImageNamesRefetchCachedValues && !scannedByV4 {
		img, err = enricher.EnrichImageByName(ctx, h.enricher, enrichmentCtx, imgName)
		if err != nil {
			return nil, err
		}

	}

	// Save the image
	img.Id = utils.GetSHA(img)
	if img.GetId() != "" {
		if err := h.saveImage(img); err != nil {
			return nil, err
		}
	}

	return img, nil
}

func (h sbomHttpHandler) saveImage(img *storage.Image) error {
	if err := h.riskManager.CalculateRiskAndUpsertImage(img); err != nil {
		log.Errorw("Error upserting image", logging.ImageName(img.GetName().GetFullName()), logging.Err(err))
		return err
	}
	return nil
}

func (h sbomHttpHandler) getSbom(ctx context.Context, params sbomRequestBody) ([]byte, error) {
	enrichmentCtx := enricher.EnrichmentContext{
		FetchOpt:        enricher.UseCachesIfPossible,
		Delegable:       true,
		ScannerTypeHint: scannerTypes.ScannerV4,
	}

	if params.Force {
		enrichmentCtx.FetchOpt = enricher.UseImageNamesRefetchCachedValues
	}

	if params.Cluster != "" {
		// The request indicates enrichment should be delegated to a specific cluster.
		clusterID, err := clusterUtil.GetClusterIDFromNameOrID(ctx, h.clusterSACHelper, params.Cluster, delegateScanPermissions)
		if err != nil {
			return nil, err
		}
		enrichmentCtx.ClusterID = clusterID
	}
	img, err := h.enrichImage(ctx, enrichmentCtx, params.ImageName)
	if err != nil {
		return nil, err
	}
	//verify that index report exists. if not force image enrichment using scanner v4
	_, scannerV4 := h.getScannerV4Integration(img)
	sbom, err := scannerV4.GetSBOM(img)
	if err == errox.NotFound {
		img, err = enricher.EnrichImageByName(ctx, h.enricher, enrichmentCtx, params.ImageName)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return sbom, nil
}
