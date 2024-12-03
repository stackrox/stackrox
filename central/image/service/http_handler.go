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
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/integration"
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
}

var _ http.Handler = (*sbomHttpHandler)(nil)

// SBOMHandler returns a handler for get sbom http request
func SBOMHandler(integration integration.Set, enricher enricher.ImageEnricher, clusterSACHelper sachelper.ClusterSacHelper) http.Handler {
	return sbomHttpHandler{
		integration:      integration,
		enricher:         enricher,
		clusterSACHelper: clusterSACHelper,
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
	//for testing only
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

func (h sbomHttpHandler) enrichImage(params sbomRequestBody, ctx context.Context) (*storage.Image, error) {
	enrichmentCtx := enricher.EnrichmentContext{
		FetchOpt:  enricher.UseCachesIfPossible,
		Delegable: true,
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

	img, err := enricher.EnrichImageByName(ctx, h.enricher, enrichmentCtx, params.ImageName)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (h sbomHttpHandler) getSbom(ctx context.Context, params sbomRequestBody) ([]byte, error) {
	// enrich image checks image metadata cache if fetchopt = UseCachesIfPossible otherwise fetches metdata from registry
	// enrich image calls get scans on image which creates index report for image if it does not exsist
	_, err := h.enrichImage(params, ctx)
	if err != nil {
		return nil, err
	}
	// how to verify enrichimage failed because index report not found
	// get sbom from matcher
	// for testing only
	sbom := map[string]interface{}{
		"SPDXID":      "SPDXRef-DOCUMENT",
		"spdxVersion": "SPDX-2.3",
		"creationInfo": map[string]interface{}{
			"created": "2023-08-30T04:40:16Z",
			"creators": []string{
				"Organization: Uchiha Cortez",
				"Tool: FOSSA v0.12.0",
			},
		},
	}
	sbomBytes, err := json.Marshal(sbom)
	if err != nil {
		return nil, err
	}
	return sbomBytes, nil
}
