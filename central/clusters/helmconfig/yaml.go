package helmconfig

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/pkg/apiparams"
	helmConfig "github.com/stackrox/rox/pkg/helm/config"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
	"gopkg.in/yaml.v3"
)

// Handler returns a handler for the helm-config method.
func Handler(c datastore.DataStore) http.Handler {
	return helmConfigHandler{
		clusterStore: c,
	}
}

type helmConfigHandler struct {
	clusterStore datastore.DataStore
}

// ServeHTTP serves a cluster's Helm chart configuration as YAML.
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

	cluster, _, err := h.clusterStore.GetCluster(r.Context(), params.ID)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}
	if cluster == nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "cluster %q not found", params.ID)
		return
	}

	flavor := defaults.GetImageFlavorFromEnv()
	config, err := helmConfig.FromCluster(cluster, flavor)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "deriving Helm configuration for cluster"))
		return
	}
	configYaml, err := yaml.Marshal(config)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "marshalling cluster configuration as YAML"))
		return
	}
	configYamlBytes := []byte(configYaml)

	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="values-%s.yaml"`, zip.GetSafeFilename(cluster.GetName())))
	w.Header().Add("Content-Type", "text/yaml")
	w.Header().Add("Content-Length", fmt.Sprint(len(configYamlBytes)))
	_, _ = w.Write(configYamlBytes)
}
