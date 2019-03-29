package zip

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/golang/protobuf/jsonpb"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/central/monitoring"
	serviceIDStore "github.com/stackrox/rox/central/serviceidentities/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.LoggerForModule()

	separators                = regexp.MustCompile(`[ &_=+:/]`)
	alphanumericNameAndDashes = regexp.MustCompile(`[^[:alnum:]\-]`)
	dashes                    = regexp.MustCompile(`[\-]+`)
)

// Handler returns a handler for the cluster zip method.
func Handler(c datastore.DataStore, s serviceIDStore.Store) http.Handler {
	return zipHandler{
		clusterStore:  c,
		identityStore: s,
	}
}

type zipHandler struct {
	clusterStore  datastore.DataStore
	identityStore serviceIDStore.Store
}

func getSafeFilename(s string) string {
	// Lowercase to be compatible with all systems. Don't end with a space
	s = strings.ToLower(strings.TrimSpace(s))
	// Replace separators with dash
	s = separators.ReplaceAllString(s, "-")
	// Remove all unknown chars
	s = alphanumericNameAndDashes.ReplaceAllString(s, "")
	// multiple dashes to 1 dash
	s = dashes.ReplaceAllString(s, "-")
	return s
}

func (z zipHandler) createIdentity(wrapper *zip.Wrapper, id string, servicePrefix string, serviceType storage.ServiceType) error {
	issuedCert, err := mtls.IssueNewCert(mtls.NewSubject(id, serviceType), z.identityStore)
	if err != nil {
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
		w.WriteHeader(http.StatusBadRequest)
		writeGRPCStyleError(w, codes.InvalidArgument, err)
		return
	}

	wrapper := zip.NewWrapper()

	// Add cluster YAML and command
	cluster, _, err := z.clusterStore.GetCluster(clusterID.GetId())
	if cluster == nil {
		if err == nil {
			err = fmt.Errorf("cluster %q not found", clusterID.GetId())
		}
		writeGRPCStyleError(w, codes.Internal, err)
		return
	}

	deployer, err := clusters.NewDeployer(cluster)
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, err)
		return
	}

	ca, err := mtls.CACertPEM()
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, errors.Wrap(err, "unable to retrieve CA Cert"))
		return
	}
	wrapper.AddFiles(zip.NewFile("ca.pem", ca, 0))

	baseFiles, err := deployer.Render(clusters.Wrap(*cluster), ca)
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, errors.Wrap(err, "could not render all files"))
		return
	}

	wrapper.AddFiles(baseFiles...)

	// Add MTLS files for sensor

	if err := z.createIdentity(wrapper, cluster.GetId(), "sensor", storage.ServiceType_SENSOR_SERVICE); err != nil {
		writeGRPCStyleError(w, codes.Internal, err)
		return
	}

	if err := z.createIdentity(wrapper, cluster.GetId(), "benchmark", storage.ServiceType_BENCHMARK_SERVICE); err != nil {
		writeGRPCStyleError(w, codes.Internal, err)
		return
	}

	// Add MTLS files for collector if runtime support is enabled
	if cluster.GetRuntimeSupport() {
		if err := z.createIdentity(wrapper, cluster.GetId(), "collector", storage.ServiceType_COLLECTOR_SERVICE); err != nil {
			writeGRPCStyleError(w, codes.Internal, err)
			return
		}
	}

	if cluster.GetMonitoringEndpoint() != "" {
		if err := z.createIdentity(wrapper, cluster.GetId(), "monitoring-client", storage.ServiceType_MONITORING_CLIENT_SERVICE); err != nil {
			writeGRPCStyleError(w, codes.Internal, err)
			return
		}

		monitoringCA, err := ioutil.ReadFile(monitoring.CAPath)
		if err != nil {
			writeGRPCStyleError(w, codes.Internal, err)
			return
		}
		wrapper.AddFiles(zip.NewFile("monitoring-ca.pem", monitoringCA, 0))
	}

	bytes, err := wrapper.Zip()
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, errors.Wrap(err, "unable to render zip file"))
		return
	}

	zipAttachment := fmt.Sprintf(`attachment; filename="sensor-%s.zip"`, getSafeFilename(cluster.GetName()))

	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", zipAttachment)
	_, _ = w.Write(bytes)
}

func writeGRPCStyleError(w http.ResponseWriter, c codes.Code, err error) {
	userErr := status.New(c, err.Error()).Proto()
	m := jsonpb.Marshaler{}

	w.WriteHeader(runtime.HTTPStatusFromCode(c))
	_ = m.Marshal(w, userErr)
}
