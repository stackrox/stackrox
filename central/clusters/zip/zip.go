package zip

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/clusters"
	siDataStore "github.com/stackrox/rox/central/serviceidentities/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
)

var (
	log = logging.LoggerForModule()
)

const (
	createUpgraderSADefault = false
)

// Handler returns a handler for the cluster zip method.
func Handler(c datastore.DataStore, s siDataStore.DataStore) http.Handler {
	return zipHandler{
		clusterStore:  c,
		identityStore: s,
	}
}

type zipHandler struct {
	clusterStore  datastore.DataStore
	identityStore siDataStore.DataStore
}

// ServeHTTP serves a ZIP file for the cluster upon request.
func (z zipHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var params apiparams.ClusterZip
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

	if identity.Service().GetType() == storage.ServiceType_SENSOR_SERVICE {
		if identity.Service().GetId() != params.ID {
			httputil.WriteGRPCStyleError(w, codes.PermissionDenied, errors.New("sensors may only download their own bundle"))
			return
		}
	}

	wrapper := zip.NewWrapper()

	// Add cluster YAML and command
	cluster, _, err := z.clusterStore.GetCluster(r.Context(), params.ID)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}
	if cluster == nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "cluster %q not found", params.ID)
		return
	}

	deployer, err := clusters.NewDeployer(cluster)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}

	ca, err := AddCertificatesToZip(wrapper, cluster, z.identityStore)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "could not add all required certificates"))
	}

	// Ignore the param unless the feature flag is enabled.
	var createUpgraderSA bool
	if params.CreateUpgraderSA == nil {
		createUpgraderSA = createUpgraderSADefault
	} else {
		createUpgraderSA = *params.CreateUpgraderSA
	}

	baseFiles, err := deployer.Render(cluster, ca, clusters.RenderOptions{CreateUpgraderSA: createUpgraderSA})
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "could not render all files"))
		return
	}

	wrapper.AddFiles(baseFiles...)

	bytes, err := wrapper.Zip()
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "unable to render zip file"))
		return
	}

	zipAttachment := fmt.Sprintf(`attachment; filename="sensor-%s.zip"`, zip.GetSafeFilename(cluster.GetName()))

	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", zipAttachment)
	_, _ = w.Write(bytes)
}
