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
	"github.com/stackrox/rox/image/sensor"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/grpc/authn"
	helmUtil "github.com/stackrox/rox/pkg/helm/util"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
	"helm.sh/helm/v3/pkg/chartutil"
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

func renderBaseFiles(cluster *storage.Cluster, renderOpts clusters.RenderOptions, certs sensor.Certs) ([]*zip.File, error) {
	imageFlavor := defaults.GetImageFlavorFromEnv()
	fields, err := clusters.FieldsFromClusterAndRenderOpts(cluster, &imageFlavor, renderOpts)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get required cluster information")
	}

	opts := helmUtil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      "stackrox-secured-cluster-services",
			Namespace: "stackrox",
			IsInstall: true,
		},
	}
	if renderOpts.IstioVersion != "" {
		istioAPIResources, err := istioutils.GetAPIResourcesByVersion(renderOpts.IstioVersion)
		if err != nil {
			return nil, errors.Wrap(err, "unable to retrieve Istio API resources")
		}
		opts.APIVersions = helmUtil.VersionSetFromResources(istioAPIResources...)
	}

	baseFiles, err := renderer.RenderSensor(fields, &certs, opts)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get required cluster information")
	}

	return baseFiles, nil
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

	identity, err := authn.IdentityFromContext(r.Context())
	if err != nil {
		httputil.WriteError(w, err)
		return
	}

	clusterID := params.ID
	if identity.Service().GetType() == storage.ServiceType_SENSOR_SERVICE {
		var err error
		clusterID, err = centralsensor.GetClusterID(clusterID, identity.Service().GetId())
		if err != nil {
			httputil.WriteGRPCStyleError(w, codes.PermissionDenied, errors.Wrapf(err, "sensor is not authorized to download bundle for cluster %q", params.ID))
			return
		}
	}

	wrapper := zip.NewWrapper()

	// Add cluster YAML and command
	cluster, _, err := z.clusterStore.GetCluster(r.Context(), clusterID)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}
	if cluster == nil {
		httputil.WriteGRPCStyleErrorf(w, codes.Internal, "cluster %q not found", params.ID)
		return
	}

	certs, err := GenerateCertsAndAddToZip(wrapper, cluster, z.identityStore)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "could not add all required certificates"))
		return
	}

	// Ignore the param unless the feature flag is enabled.
	var createUpgraderSA bool
	if params.CreateUpgraderSA == nil {
		createUpgraderSA = createUpgraderSADefault
	} else {
		createUpgraderSA = *params.CreateUpgraderSA
	}

	var slimCollector bool
	if params.SlimCollector == nil {
		// In case it is not provided in the request we use the value as persisted for the cluster.
		slimCollector = cluster.GetSlimCollector()
	} else {
		slimCollector = *params.SlimCollector
	}

	renderOpts := clusters.RenderOptions{
		CreateUpgraderSA: createUpgraderSA,
		SlimCollector:    slimCollector,
		IstioVersion:     params.IstioVersion,

		DisablePodSecurityPolicies: params.DisablePodSecurityPolicies,
	}

	baseFiles, err := renderBaseFiles(cluster, renderOpts, certs)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
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
