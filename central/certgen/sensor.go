package certgen

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/central/clusters/zip"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/renderer"
	pkgZip "github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *serviceImpl) getSensorCerts(r *http.Request) ([]byte, *storage.Cluster, error) {
	var params apiparams.ClusterCertGen
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "invalid params: %v", err)
	}

	clusterID := params.ID
	if clusterID == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "no cluster ID specified")
	}

	cluster, _, err := s.clusters.GetCluster(r.Context(), clusterID)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "failed to retrieve cluster: %v", err)
	}
	if cluster == nil {
		return nil, nil, status.Errorf(codes.NotFound, "cluster with ID %q not found", clusterID)
	}

	certs, err := zip.GenerateCertsAndAddToZip(nil, cluster, s.serviceIdentities)
	if err != nil {
		return nil, nil, status.Errorf(codes.Internal, "could not generate certs: %v", err)
	}

	fields, err := clusters.FieldsFromClusterAndRenderOpts(cluster, clusters.RenderOptions{})
	if err != nil {
		return nil, nil, status.Error(codes.Internal, err.Error())
	}

	rendered, err := renderer.RenderSensorTLSSecretsOnly(fields, &certs)
	if err != nil {
		return nil, nil, status.Error(codes.Internal, err.Error())
	}
	return rendered, cluster, nil
}

func (s *serviceImpl) securedClusterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httputil.WriteErrorf(w, http.StatusMethodNotAllowed, "invalid method %s, only POST allowed", r.Method)
		return
	}

	rendered, cluster, err := s.getSensorCerts(r)
	if err != nil {
		httputil.WriteError(w, err)
		return
	}

	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="cluster-%s-tls.yaml"`, pkgZip.GetSafeFilename(cluster.GetName())))
	_, _ = w.Write(rendered)
}
