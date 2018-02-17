package zip

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"bitbucket.org/stack-rox/apollo/central/service"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	logger = logging.LoggerForModule()
)

// Handler returns a handler for the cluster zip method.
func Handler(c *service.ClusterService, s *service.IdentityService) http.Handler {
	return zipHandler{
		clusterService:  c,
		identityService: s,
	}
}

type zipHandler struct {
	clusterService  *service.ClusterService
	identityService *service.IdentityService
}

// ServeHTTP serves a ZIP file for the cluster upon request.
func (z zipHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var cluster v1.Cluster
	err := jsonpb.Unmarshal(r.Body, &cluster)
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
	resp, err := z.clusterService.PostCluster(ctx, &cluster)
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, err)
		return
	}
	if !addFile(w, zipW, "sensor-deploy.yaml", resp.GetDeploymentYaml()) {
		return
	}
	if !addExecutableFile(w, zipW, "sensor-deploy.sh", resp.GetDeploymentCommand()) {
		return
	}

	// Add MTLS files
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
	if !addFile(w, zipW, "sensor-cert.pem", id.GetCertificate()) {
		return
	}
	if !addFile(w, zipW, "sensor-key.pem", id.GetPrivateKey()) {
		return
	}
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	authority, err := z.identityService.GetAuthorities(ctx, &empty.Empty{})
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, err)
		return
	}
	if len(authority.GetAuthorities()) != 1 {
		writeGRPCStyleError(w, codes.Internal, fmt.Errorf("authority: got %d authorities", len(authority.GetAuthorities())))
	}
	if !addFile(w, zipW, "central-ca.pem", authority.GetAuthorities()[0].GetCertificate()) {
		return
	}

	err = zipW.Close()
	if err != nil {
		logger.Warnf("Couldn't close zip writer: %s", err)
	}

	// Tell the browser this is a download.
	w.Header().Add("Content-Disposition", `attachment; filename="sensor-deploy.zip"`)
	w.Write(buf.Bytes())
}

func addFile(w http.ResponseWriter, zipW *zip.Writer, name, contents string) (ok bool) {
	f, err := zipW.Create(name)
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, fmt.Errorf("%s creation: %s", name, err))
		return false
	}
	_, err = f.Write([]byte(contents))
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, fmt.Errorf("%s writing: %s", name, err))
		return false
	}
	return true
}

func addExecutableFile(w http.ResponseWriter, zipW *zip.Writer, name, contents string) (ok bool) {
	hdr := &zip.FileHeader{
		Name: name,
	}
	hdr.SetMode(os.ModePerm & 0755)
	f, err := zipW.CreateHeader(hdr)
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, fmt.Errorf("%s creation: %s", name, err))
		return false
	}
	_, err = f.Write([]byte(contents))
	if err != nil {
		writeGRPCStyleError(w, codes.Internal, fmt.Errorf("%s writing: %s", name, err))
		return false
	}
	return true
}

func writeGRPCStyleError(w http.ResponseWriter, c codes.Code, err error) {
	userErr := status.New(c, err.Error()).Proto()
	m := jsonpb.Marshaler{}

	w.WriteHeader(runtime.HTTPStatusFromCode(c))
	m.Marshal(w, userErr)
}
