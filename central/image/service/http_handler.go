package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/enrichment"
	"github.com/stackrox/rox/central/imageintegration"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/images/enricher"
	"github.com/stackrox/rox/pkg/images/integration"
	"github.com/stackrox/rox/pkg/scanners/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
)

type sbomHandler struct {
	integration integration.Set
	enricher    enricher.ImageEnricher
}

type SBOMRequestBody struct {
	ClusterID string `json:"clusterID"`
	ImageName string `json:"imageName"`
	force     bool   `json:"force"`
}

func ImageHandler() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var params SBOMRequestBody
		err := json.NewDecoder(r.Body).Decode(&params)
		if err != nil {
			httputil.WriteGRPCStyleError(w, codes.Internal, err)
			return
		}
		bytes, err := getSbom(params, r.Context())
		if err != nil {
			httputil.WriteGRPCStyleError(w, codes.Internal, err)
			return
		}

		// Tell the browser this is a download.
		w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="values-%s.json"`, "sbom.json"))
		w.Header().Add("Content-Type", "text/yaml")
		w.Header().Add("Content-Length", fmt.Sprint(len(bytes)))
		_, _ = w.Write(bytes)
		return
	}
}

func enrichImage(params SBOMRequestBody, ctx context.Context) (*storage.Image, error) {
	enrichmentCtx := enricher.EnrichmentContext{
		FetchOpt:  enricher.UseCachesIfPossible,
		Delegable: true,
	}

	if params.force {
		enrichmentCtx.FetchOpt = enricher.UseImageNamesRefetchCachedValues
	}

	if params.ClusterID != "" {
		enrichmentCtx.ClusterID = params.ClusterID
	}

	img, err := enricher.EnrichImageByName(ctx, enrichment.ImageEnricherSingleton(), enrichmentCtx, params.ImageName)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func getSbom(params SBOMRequestBody, ctx context.Context) ([]byte, error) {
	//verify scanner v4 is enabled
	scannerv4Enabled := false
	scanners := imageintegration.Set().ScannerSet()
	for _, scanner := range scanners.GetAll() {
		if scanner.GetScanner().Type() == types.ScannerV4 {
			scannerv4Enabled = true
		}
	}
	if !scannerv4Enabled {
		return nil, errors.New("Scanner v4 is not enabled. Enabled scanner v4 to get SBOMs")
	}

	//enrich image checks image metadata cache if fetchopt = UseCachesIfPossible otherwise fetches metdata from registry
	//enrich image calls get scans on image which creates index report for image if it does not exsist
	_, err := enrichImage(params, ctx)
	if err != nil {
		return nil, err
	}
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
