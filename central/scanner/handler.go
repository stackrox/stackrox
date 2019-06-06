package scanner

import (
	"bytes"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/docker/distribution/reference"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/roximages/defaults"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
)

func validateParamsAndNormalizeClusterType(p *apiparams.Scanner) (storage.ClusterType, error) {
	errorList := errorhelpers.NewErrorList("invalid params:")

	clusterType := storage.ClusterType(storage.ClusterType_value[p.ClusterType])

	if int32(clusterType) == 0 {
		var validClusterTypes []string
		for clusterString, value := range storage.ClusterType_value {
			if value > 0 {
				validClusterTypes = append(validClusterTypes, clusterString)
			}
		}
		sort.Strings(validClusterTypes)
		errorList.AddStringf("invalid cluster type: %q; valid options are %+v", p.ClusterType, validClusterTypes)
	}

	if p.ScannerImage != "" {
		if _, err := reference.ParseAnyReference(p.ScannerImage); err != nil {
			errorList.AddWrapf(err, "invalid scanner image")
		}
	}
	return clusterType, errorList.ToError()
}

func serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var params apiparams.Scanner
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(r.Body)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, err)
		return
	}
	err = json.Unmarshal(buf.Bytes(), &params)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}

	clusterType, err := validateParamsAndNormalizeClusterType(&params)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, err)
		return
	}

	if params.ScannerImage == "" {
		params.ScannerImage = defaults.ScannerImage()
	}

	config := renderer.Config{
		ClusterType: clusterType,
		K8sConfig: &renderer.K8sConfig{
			CommonConfig: renderer.CommonConfig{
				ScannerImage: params.ScannerImage,
			},
			OfflineMode: params.OfflineMode,
		},
	}

	files, err := renderer.RenderScannerOnly(config)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, err)
		return
	}

	wrapper := zip.NewWrapper()
	wrapper.AddFiles(files...)
	bytes, err := wrapper.Zip()
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
	}

	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", `attachment; filename="scanner-bundle.zip"`)
	_, _ = w.Write(bytes)

}

// Handler returns the handler that serves scanner zip files.
func Handler() http.Handler {
	return http.HandlerFunc(serveHTTP)
}
