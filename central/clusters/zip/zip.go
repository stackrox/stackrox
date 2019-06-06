package zip

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/protobuf/jsonpb"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/central/monitoring"
	"github.com/stackrox/rox/central/role/resources"
	siDataStore "github.com/stackrox/rox/central/serviceidentities/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
)

var (
	log = logging.LoggerForModule()
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

// ServeHTTP serves a ZIP file for the cluster upon request.
func (z zipHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var clusterID v1.ResourceByID
	err := jsonpb.Unmarshal(r.Body, &clusterID)
	if err != nil {
		httputil.WriteGRPCStyleError(w, codes.InvalidArgument, err)
		return
	}

	wrapper := zip.NewWrapper()

	// Add cluster YAML and command
	cluster, _, err := z.clusterStore.GetCluster(r.Context(), clusterID.GetId())
	if cluster == nil {
		if err == nil {
			err = fmt.Errorf("cluster %q not found", clusterID.GetId())
		}
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
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

	baseFiles, err := deployer.Render(clusters.Wrap(*cluster), ca)
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

	if err := z.createIdentity(wrapper, cluster.GetId(), "benchmark", storage.ServiceType_BENCHMARK_SERVICE); err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}

	// Add MTLS files for collector if runtime support is enabled
	if cluster.GetCollectionMethod() != storage.CollectionMethod_NO_COLLECTION {
		if err := z.createIdentity(wrapper, cluster.GetId(), "collector", storage.ServiceType_COLLECTOR_SERVICE); err != nil {
			httputil.WriteGRPCStyleError(w, codes.Internal, err)
			return
		}
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
