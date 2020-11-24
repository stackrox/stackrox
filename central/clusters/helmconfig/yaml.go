package helmconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
	"gopkg.in/yaml.v3"
)

var (
	log = logging.LoggerForModule()
)

// Handler returns a handler for the cluster zip method.
func Handler(c datastore.DataStore) http.Handler {
	return helmConfigHandler{
		clusterStore: c,
	}
}

type helmConfigHandler struct {
	clusterStore datastore.DataStore
}

// Returns the cluster's Helm configuration.
func (h helmConfigHandler) deriveHelmConfig(cluster *storage.Cluster) map[string]interface{} {
	m := map[string]interface{}{
		"name": cluster.GetName(),
	}

	return m
}

// ServeHTTP serves a ZIP file for the cluster upon request.
func (h helmConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var params apiparams.ClusterHelmConfig
	err := json.NewDecoder(r.Body).Decode(&params)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}

	identity := authn.IdentityFromContext(r.Context())
	if identity == nil {
		httputil.WriteGRPCStyleError(w, codes.Unauthenticated, errors.New("no identity in context"))
		return
	}

	cluster, _, err := h.clusterStore.GetCluster(r.Context(), params.ID)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}
	if cluster == nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "cluster %q not found", params.ID)
		return
	}

	config := h.deriveHelmConfig(cluster)
	configYaml, err := yaml.Marshal(config)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}

	attachment := fmt.Sprintf(`attachment; filename="values-%s.yaml"`, zip.GetSafeFilename(cluster.GetName()))

	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", attachment)
	w.Header().Add("Content-Type", "text/yaml")
	_, _ = w.Write([]byte(configYaml))
}
