package zip

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/central/monitoring"
	"github.com/stackrox/rox/central/role/resources"
	siDataStore "github.com/stackrox/rox/central/serviceidentities/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
)

const (
	additionalCAsDir       = "/usr/local/share/ca-certificates"
	additionalCAsZipSubdir = "additional-cas"
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

func (z zipHandler) createIdentity(wrapper *zip.Wrapper, id string, servicePrefix string, serviceType storage.ServiceType) error {
	srvIDAllAccessCtx := sac.WithGlobalAccessScopeChecker(context.Background(),
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS, storage.Access_READ_WRITE_ACCESS),
			sac.ResourceScopeKeys(resources.ServiceIdentity)))

	issuedCert, err := mtls.IssueNewCert(mtls.NewSubject(id, serviceType))
	if err != nil {
		return err
	}
	if err := z.identityStore.AddServiceIdentity(srvIDAllAccessCtx, issuedCert.ID); err != nil {
		return err
	}
	wrapper.AddFiles(
		zip.NewFile(fmt.Sprintf("%s-cert.pem", servicePrefix), issuedCert.CertPEM, 0),
		zip.NewFile(fmt.Sprintf("%s-key.pem", servicePrefix), issuedCert.KeyPEM, zip.Sensitive),
	)
	return nil
}

func (z zipHandler) getAdditionalCAs() ([]*zip.File, error) {
	certFileInfos, err := ioutil.ReadDir(additionalCAsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []*zip.File
	for _, fileInfo := range certFileInfos {
		if fileInfo.IsDir() || filepath.Ext(fileInfo.Name()) != ".crt" {
			continue
		}
		fullPath := path.Join(additionalCAsDir, fileInfo.Name())
		contents, err := ioutil.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		files = append(files, zip.NewFile(path.Join(additionalCAsZipSubdir, fileInfo.Name()), contents, 0))
	}

	return files, nil
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

	ca, err := mtls.CACertPEM()
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, errors.Wrap(err, "unable to retrieve CA Cert"))
		return
	}
	wrapper.AddFiles(zip.NewFile("ca.pem", ca, 0))

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

	// Add MTLS files for sensor

	if err := z.createIdentity(wrapper, cluster.GetId(), "sensor", storage.ServiceType_SENSOR_SERVICE); err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}

	// Add MTLS files for collector
	if err := z.createIdentity(wrapper, cluster.GetId(), "collector", storage.ServiceType_COLLECTOR_SERVICE); err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}

	if cluster.GetMonitoringEndpoint() != "" {
		if err := z.createIdentity(wrapper, cluster.GetId(), "monitoring-client", storage.ServiceType_MONITORING_CLIENT_SERVICE); err != nil {
			httputil.WriteGRPCStyleError(w, codes.Internal, err)
			return
		}

		monitoringCA, err := ioutil.ReadFile(monitoring.CAPath)
		if err != nil {
			httputil.WriteGRPCStyleError(w, codes.Internal, err)
			return
		}
		wrapper.AddFiles(zip.NewFile("monitoring-ca.pem", monitoringCA, 0))
	}

	additionalCAFiles, err := z.getAdditionalCAs()
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}
	wrapper.AddFiles(additionalCAFiles...)

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
