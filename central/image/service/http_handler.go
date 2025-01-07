package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	clusterUtil "github.com/stackrox/rox/central/cluster/util"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/integration"
	"google.golang.org/grpc/codes"
)

type sbomRequestBody struct {
	Cluster   string `json:"cluster"`
	ImageName string `json:"imageName"`
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
		httputil.WriteGRPCStyleError(w, codes.Unimplemented, errors.New("Scanner V4 is disabled. Enable Scanner V4 to generate SBOMs"))
		return
	}
	if !features.SBOMGeneration.Enabled() {
		httputil.WriteGRPCStyleError(w, codes.Unimplemented, errors.New("SBOM feature is not enabled"))
		return
	}
	var params sbomRequestBody
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

	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", "attachment; sbom.json")
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Content-Length", fmt.Sprint(len(bytes)))
	_, _ = w.Write(bytes)
}

func (h sbomHttpHandler) enrichImage(ctx context.Context, params sbomRequestBody) (*storage.Image, error) {
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
	// TODO(ROX-24541): save the image to the database
	return img, nil
}

func (h sbomHttpHandler) getSbom(ctx context.Context, params sbomRequestBody) ([]byte, error) {
	// enrich image checks image metadata cache if fetchopt = UseCachesIfPossible otherwise fetches metdata from registry
	// enrich image calls get scans on image which creates index report for image if it does not exsist
	_, err := h.enrichImage(ctx, params)
	if err != nil {
		return nil, err
	}

	// get sbom from matcher
	// for testing only
	sbom := map[string]interface{}{
		"SPDXID":      "SPDXRef-DOCUMENT",
		"spdxVersion": "SPDX-2.3",
		"creationInfo": map[string]interface{}{
			"created": "2023-08-30T04:40:16Z",
			"creators": []string{
				"Organization: NA Org",
				"Tool:  N/A - PoC",
			},
		},
	}
	sbomBytes, err := json.Marshal(sbom)
	if err != nil {
		return nil, err
	}
	return sbomBytes, nil
}
