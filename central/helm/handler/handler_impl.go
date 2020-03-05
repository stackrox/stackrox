package handler

import (
	"fmt"
	"net/http"

	"github.com/golang/protobuf/jsonpb"
	clustersZip "github.com/stackrox/rox/central/clusters/zip"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
)

func (h helmHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		h.post(w, r)
		return
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
}

func (h helmHandler) post(w http.ResponseWriter, r *http.Request) {
	var helmCluster storage.HelmCluster
	err := jsonpb.Unmarshal(r.Body, &helmCluster)

	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "JSON unmarshaling: %v", err)
		return
	}

	// Add the new cluster to db
	response, err := h.clusterService.PostCluster(r.Context(), helmCluster.Cluster)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "Adding cluster: %v", err)
		return
	}
	cluster := response.Cluster
	log.Infof("Added Helm cluster %s successfully", cluster.GetName())

	// Generate certificates for the cluster to be used in the Helm workflow
	wrapper := zip.NewWrapper()
	_, err = clustersZip.AddCertificatesToZip(wrapper, cluster, h.identityStore)
	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "Adding certificates to zip: %v", err)
		return
	}

	zipAttachment := fmt.Sprintf(`attachment; filename="certs-%s.zip"`, zip.GetSafeFilename(cluster.GetName()))

	bytes, err := wrapper.Zip()
	if err != nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "Unable to render zip file: %v", err)
		return
	}

	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", zipAttachment)
	_, _ = w.Write(bytes)
}
