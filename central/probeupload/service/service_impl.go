package service

import (
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/gogo/protobuf/proto"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/probesources"
	"github.com/stackrox/rox/central/probeupload/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/probeupload"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Administration)): {
			"/v1.ProbeUploadService/GetExistingProbes",
		},
	})

	log = logging.LoggerForModule()
)

type service struct {
	v1.UnimplementedProbeUploadServiceServer

	mgr manager.Manager

	probeServerHandler http.Handler
}

func newService(mgr manager.Manager) *service {
	probeSources := probesources.Singleton().CopyAsSlice()
	return &service{
		mgr:                mgr,
		probeServerHandler: probeupload.NewProbeServerHandler(probeupload.LogCallback(log), probeSources...),
	}
}

func (s *service) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterProbeUploadServiceServer(grpcServer, s)
}

func (s *service) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterProbeUploadServiceHandler(ctx, mux, conn)
}

func (s *service) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *service) GetExistingProbes(ctx context.Context, req *v1.GetExistingProbesRequest) (*v1.GetExistingProbesResponse, error) {
	fileInfos, err := s.mgr.GetExistingProbeFiles(ctx, req.GetFilesToCheck())
	if err != nil {
		return nil, err
	}
	return &v1.GetExistingProbesResponse{
		ExistingFiles: fileInfos,
	}, nil
}

func (s *service) CustomRoutes() []routes.CustomRoute {
	return []routes.CustomRoute{
		{
			Route:      "/api/extensions/probeupload",
			Authorizer: user.With(permissions.Modify(resources.Administration)),
			ServerHandler: utils.IfThenElse[http.Handler](
				env.EnableKernelPackageUpload.BooleanSetting(), http.HandlerFunc(s.handleProbeUpload),
				httputil.NotImplementedHandler("api is not supported because kernel package upload is disabled.")),
			Compression: false,
		},
		{
			Route:         "/kernel-objects/",
			Authorizer:    idcheck.SensorsOnly(),
			ServerHandler: http.StripPrefix("/kernel-objects", s.probeServerHandler),
			Compression:   false,
		},
	}
}

func (s *service) handleProbeUpload(w http.ResponseWriter, req *http.Request) {
	if err := s.doHandleProbeUpload(req); err != nil {
		httputil.WriteGRPCStyleError(w, codes.Internal, err)
		return
	}
}

func (s *service) doHandleProbeUpload(req *http.Request) error {
	if req.Method != http.MethodPost {
		return errors.New("only post requests are allowed")
	}

	manifestLenStr := req.URL.Query().Get("manifestLen")
	manifestLen, err := strconv.Atoi(manifestLenStr)
	if err != nil {
		return errors.Wrapf(err, "unparseable manifest length %q", manifestLenStr)
	}

	manifestBytes := make([]byte, manifestLen)
	if _, err := io.ReadFull(req.Body, manifestBytes); err != nil {
		return errors.Wrap(err, "error reading manifest")
	}

	var manifest v1.ProbeUploadManifest
	if err := proto.Unmarshal(manifestBytes, &manifest); err != nil {
		return errors.Wrap(err, "failed to unmarshal manifest")
	}

	totalSize, err := probeupload.AnalyzeManifest(&manifest)
	if err != nil {
		return errors.Wrap(err, "manifest is invalid")
	}

	if req.ContentLength > 0 {
		if expectedLen := totalSize + int64(manifestLen); req.ContentLength != expectedLen {
			return errors.Errorf("request payload has invalid length %d, expected %d", req.ContentLength, expectedLen)
		}
	}

	for _, file := range manifest.GetFiles() {
		nextChunk := io.LimitReader(req.Body, file.GetSize_())
		if err := s.mgr.StoreFile(req.Context(), file.GetName(), nextChunk, file.GetSize_(), file.GetCrc32()); err != nil {
			return errors.Wrapf(err, "failed to write file %s", file.GetName())
		}
	}

	return nil
}
