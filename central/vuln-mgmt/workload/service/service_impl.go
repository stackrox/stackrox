package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Deployment), permissions.View(resources.Image)): {
			"/v1.VulnMgmtWorkloadService/VulnMgmtExportWorkloads",
		},
	})
	log = logging.LoggerForModule()
)

// serviceImpl provides APIs for workload vulnerabilities.
type serviceImpl struct {
	v1.UnimplementedVulnMgmtWorkloadServiceServer

	deployments deploymentDS.DataStore
	images      imageDS.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterVulnMgmtWorkloadServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterVulnMgmtWorkloadServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) VulnMgmtExportWorkloads(req *v1.VulnMgmtExportWorkloadsRequest,
	srv v1.VulnMgmtWorkloadService_VulnMgmtExportWorkloadsServer,
) error {
	parsedQuery, err := search.ParseQuery(req.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	ctx := srv.Context()
	if timeout := req.GetTimeout(); timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(srv.Context(), time.Duration(timeout)*time.Second)
		defer cancel()
	}
	return s.deployments.WalkByQuery(ctx, parsedQuery, func(d *storage.Deployment) error {
		containers := d.GetContainers()
		images := make([]*storage.Image, 0, len(containers))
		for _, container := range containers {
			img, exists, err := s.images.GetImage(ctx, container.GetImage().GetId())
			if err != nil {
				log.Errorf("Error getting image for container %s.%s: %v", d.GetId(), container.GetName(), err)
				continue
			}
			if exists {
				images = append(images, img)
			}
		}

		if err := srv.Send(&v1.VulnMgmtExportWorkloadsResponse{Deployment: d, Images: images}); err != nil {
			return err
		}
		return nil
	})
}
