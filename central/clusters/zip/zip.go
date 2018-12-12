package zip

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/cluster/datastore"
	"github.com/stackrox/rox/central/clusters"
	"github.com/stackrox/rox/central/monitoring"
	serviceIdentitiesService "github.com/stackrox/rox/central/serviceidentities/service"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/zip"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	logger = logging.LoggerForModule()

	separators                = regexp.MustCompile(`[ &_=+:/]`)
	alphanumericNameAndDashes = regexp.MustCompile(`[^[:alnum:]\-]`)
	dashes                    = regexp.MustCompile(`[\-]+`)
)

// Handler returns a handler for the cluster zip method.
func Handler(c datastore.DataStore, s serviceIdentitiesService.Service) http.Handler {
	return zipHandler{
		clusterStore:    c,
		identityService: s,
	}
}

type zipHandler struct {
	clusterStore    datastore.DataStore
	identityService serviceIdentitiesService.Service
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

	baseFiles, err := deployer.Render(clusters.Wrap(*cluster))
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, fmt.Errorf("could not render all files: %v", err))
		return
	}

	wrapper.AddFiles(baseFiles...)

	// Add MTLS files for sensor
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	idReq := &v1.CreateServiceIdentityRequest{
		Id:   cluster.GetId(),
		Type: storage.ServiceType_SENSOR_SERVICE,
	}
	id, err := z.identityService.CreateServiceIdentity(ctx, idReq)
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, err)
		return
	}

	wrapper.AddFiles(
		zip.NewFile("sensor-cert.pem", id.GetCertificatePem(), 0),
		zip.NewFile("sensor-key.pem", id.GetPrivateKeyPem(), zip.Sensitive),
	)

	// Add MTLS files for collector if runtime support is enabled
	if cluster.GetRuntimeSupport() {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		idReq = &v1.CreateServiceIdentityRequest{
			Id:   cluster.GetId(),
			Type: storage.ServiceType_COLLECTOR_SERVICE,
		}
		id, err = z.identityService.CreateServiceIdentity(ctx, idReq)
		if err != nil {
			writeGRPCStyleError(w, codes.Internal, err)
			return
		}

		wrapper.AddFiles(
			zip.NewFile("collector-cert.pem", id.GetCertificatePem(), 0),
			zip.NewFile("collector-key.pem", id.GetPrivateKeyPem(), zip.Sensitive),
		)

	}

	if cluster.GetMonitoringEndpoint() != "" {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		idReq = &v1.CreateServiceIdentityRequest{
			Id:   cluster.GetId(),
			Type: storage.ServiceType_MONITORING_CLIENT_SERVICE,
		}
		id, err = z.identityService.CreateServiceIdentity(ctx, idReq)
		if err != nil {
			writeGRPCStyleError(w, codes.Internal, err)
			return
		}

		wrapper.AddFiles(
			zip.NewFile("monitoring-client-cert.pem", id.GetCertificatePem(), 0),
			zip.NewFile("monitoring-client-key.pem", id.GetPrivateKeyPem(), zip.Sensitive),
		)

		monitoringCA, err := ioutil.ReadFile(monitoring.CAPath)
		if err != nil {
			writeGRPCStyleError(w, codes.Internal, err)
			return
		}
		wrapper.AddFiles(zip.NewFile("monitoring-ca.pem", monitoringCA, 0))
	}

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	authority, err := z.identityService.GetAuthorities(ctx, &v1.Empty{})
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, err)
		return
	}
	if len(authority.GetAuthorities()) != 1 {
		writeGRPCStyleError(w, codes.Internal, fmt.Errorf("authority: got %d authorities", len(authority.GetAuthorities())))
		return
	}

	wrapper.AddFiles(zip.NewFile("ca.pem", authority.GetAuthorities()[0].GetCertificatePem(), 0))

	bytes, err := wrapper.Zip()
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, fmt.Errorf("unable to render zip file: %v", err))
		return
	}

	zipAttachment := fmt.Sprintf(`attachment; filename="sensor-%s.zip"`, getSafeFilename(cluster.GetName()))

	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", zipAttachment)
	w.Write(bytes)
}

func writeGRPCStyleError(w http.ResponseWriter, c codes.Code, err error) {
	userErr := status.New(c, err.Error()).Proto()
	m := jsonpb.Marshaler{}

	w.WriteHeader(runtime.HTTPStatusFromCode(c))
	m.Marshal(w, userErr)
}
