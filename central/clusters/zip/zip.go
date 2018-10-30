package zip

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	clusterService "github.com/stackrox/rox/central/cluster/service"
	"github.com/stackrox/rox/central/monitoring"
	serviceIdentitiesService "github.com/stackrox/rox/central/serviceidentities/service"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
	zipPkg "github.com/stackrox/rox/pkg/zip"
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
func Handler(c clusterService.Service, s serviceIdentitiesService.Service) http.Handler {
	return zipHandler{
		clusterService:  c,
		identityService: s,
	}
}

type zipHandler struct {
	clusterService  clusterService.Service
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

	buf := new(bytes.Buffer)
	zipW := zip.NewWriter(buf)

	// Add cluster YAML and command
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := z.clusterService.GetCluster(ctx, &clusterID)
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, err)
		return
	}

	for _, f := range resp.GetFiles() {
		if err := zipPkg.AddFile(zipW, f); err != nil {
			writeGRPCStyleError(w, codes.Internal, fmt.Errorf("%s writing: %s", f.GetName(), err))
			return
		}
	}

	// Add MTLS files for sensor
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	idReq := &v1.CreateServiceIdentityRequest{
		Id:   resp.GetCluster().GetId(),
		Type: v1.ServiceType_SENSOR_SERVICE,
	}
	id, err := z.identityService.CreateServiceIdentity(ctx, idReq)
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, err)
		return
	}

	if err := zipPkg.AddFile(zipW, zipPkg.NewFile("sensor-cert.pem", id.GetCertificatePem(), false)); err != nil {
		writeGRPCStyleError(w, codes.Internal, fmt.Errorf("%s writing: %s", "sensor-cert.pem", err))
		return
	}
	if err := zipPkg.AddFile(zipW, zipPkg.NewFile("sensor-key.pem", id.GetPrivateKeyPem(), false)); err != nil {
		writeGRPCStyleError(w, codes.Internal, fmt.Errorf("%s writing: %s", "sensor-key.pem", err))
		return
	}

	// Add MTLS files for collector if runtime support is enabled
	if resp.Cluster.GetRuntimeSupport() {
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		idReq = &v1.CreateServiceIdentityRequest{
			Id:   resp.GetCluster().GetId(),
			Type: v1.ServiceType_COLLECTOR_SERVICE,
		}
		id, err = z.identityService.CreateServiceIdentity(ctx, idReq)
		if err != nil {
			writeGRPCStyleError(w, codes.Internal, err)
			return
		}

		if err := zipPkg.AddFile(zipW, zipPkg.NewFile("collector-cert.pem", id.GetCertificatePem(), false)); err != nil {
			writeGRPCStyleError(w, codes.Internal, fmt.Errorf("%s writing: %s", "collector-cert.pem", err))
			return
		}
		if err := zipPkg.AddFile(zipW, zipPkg.NewFile("collector-key.pem", id.GetPrivateKeyPem(), false)); err != nil {
			writeGRPCStyleError(w, codes.Internal, fmt.Errorf("%s writing: %s", "collector-key.pem", err))
			return
		}
	}

	if resp.GetCluster().GetMonitoringEndpoint() != "" {
		// Pass the monitoring password and CA
		password, err := ioutil.ReadFile(monitoring.PasswordPath)
		if err != nil {
			writeGRPCStyleError(w, codes.InvalidArgument,
				errors.New("Could not read monitoring password in Central. Please remove the monitoring endpoint to continue"))
			return
		}
		// Add the files required for monitoring
		if err := zipPkg.AddFile(zipW, zipPkg.NewFile("monitoring-password", password, false)); err != nil {
			writeGRPCStyleError(w, codes.Internal, fmt.Errorf("%s writing: %s", "monitoring-password", err))
			return
		}
		monitoringCA, err := ioutil.ReadFile(monitoring.CAPath)
		if err != nil {
			writeGRPCStyleError(w, codes.Internal, err)
			return
		}
		if err := zipPkg.AddFile(zipW, zipPkg.NewFile("monitoring-ca.pem", monitoringCA, false)); err != nil {
			writeGRPCStyleError(w, codes.Internal, fmt.Errorf("%s writing: %s", "monitoring-ca.pem", err))
			return
		}
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

	if err := zipPkg.AddFile(zipW, zipPkg.NewFile("ca.pem", authority.GetAuthorities()[0].GetCertificatePem(), false)); err != nil {
		writeGRPCStyleError(w, codes.Internal, fmt.Errorf("%s writing: %s", "ca.pem", err))
		return
	}

	err = zipW.Close()
	if err != nil {
		logger.Warnf("Couldn't close zip writer: %s", err)
	}

	zipAttachment := fmt.Sprintf(`attachment; filename="sensor-%s.zip"`, getSafeFilename(resp.GetCluster().GetName()))

	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", zipAttachment)
	w.Write(buf.Bytes())
}

func writeGRPCStyleError(w http.ResponseWriter, c codes.Code, err error) {
	userErr := status.New(c, err.Error()).Proto()
	m := jsonpb.Marshaler{}

	w.WriteHeader(runtime.HTTPStatusFromCode(c))
	m.Marshal(w, userErr)
}
