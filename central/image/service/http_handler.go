package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
	clusterUtil "github.com/stackrox/rox/central/cluster/util"
	"github.com/stackrox/rox/central/role/sachelper"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/integration"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
)

const (
	NumberBytes = 100
)

type sbomRequestBody struct {
	Cluster   string `json:"cluster"`
	ImageName string `json:"image_name"`
	Force     bool   `json:"force"`
}

type httpHandler struct {
	integration      integration.Set
	enricher         enricher.ImageEnricher
	clusterSACHelper sachelper.ClusterSacHelper
}

var _ http.Handler = (*httpHandler)(nil)

// Handler returns a handler for policy http requests
func Handler(integration integration.Set, enricher enricher.ImageEnricher, clusterSACHelper sachelper.ClusterSacHelper) http.Handler {
	return httpHandler{
		integration:      integration,
		enricher:         enricher,
		clusterSACHelper: clusterSACHelper,
	}

}
func (h httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var params sbomRequestBody
	const maxSize = NumberBytes * 1024 // (100 KiB is way more than needed for this request, ideally make this configurable)

	lr := io.LimitReader(r.Body, maxSize)
	err := json.NewDecoder(lr).Decode(&params)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}
	bytes, err := h.getSbom(params, r.Context())
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "decoding json request body"))
		return
	}

	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", "attachment; sbom.json")
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Content-Length", fmt.Sprint(len(bytes)))
	_, _ = w.Write(bytes)
	return
}

func (h httpHandler) enrichImage(params sbomRequestBody, ctx context.Context) (*storage.Image, error) {
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

	log.Infof("params %s", params)

	img, err := enricher.EnrichImageByName(ctx, h.enricher, enrichmentCtx, params.ImageName)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func (h httpHandler) getSbom(params sbomRequestBody, ctx context.Context) ([]byte, error) {
	//verify scanner v4 is enabled
	if !features.ScannerV4.Enabled() {
		return nil, errors.New("Scanner v4 is not enabled. Enabled scanner v4 to get SBOMs")
	}

	//enrich image checks image metadata cache if fetchopt = UseCachesIfPossible otherwise fetches metdata from registry
	//enrich image calls get scans on image which creates index report for image if it does not exsist
	//_, err := h.enrichImage(params, ctx)
	//if err != nil {
	//	return nil, err
	//}
	//how to verify enrichimage failed because index report not found
	//get sbom from matcher
	//for testing only
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
