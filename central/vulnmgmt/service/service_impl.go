package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	lru "github.com/hashicorp/golang-lru/v2"
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
	"github.com/stackrox/rox/pkg/set"
	"google.golang.org/grpc"
)

const (
	cacheSize = 1000
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Deployment), permissions.View(resources.Image)): {
			"/v1.VulnMgmtService/VulnMgmtExportWorkloads",
		},
	})
	log = logging.LoggerForModule()
)

// serviceImpl provides APIs for vulnerability management.
type serviceImpl struct {
	v1.UnimplementedVulnMgmtServiceServer

	deployments deploymentDS.DataStore
	images      imageDS.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterVulnMgmtServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterVulnMgmtServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this service.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) VulnMgmtExportWorkloads(req *v1.VulnMgmtExportWorkloadsRequest,
	srv v1.VulnMgmtService_VulnMgmtExportWorkloadsServer,
) error {
	parsedQuery, err := search.ParseQuery(req.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	ctx := srv.Context()
	if timeout := req.GetTimeout(); timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Millisecond)
		defer cancel()
	}
	imageCache, err := lru.New[string, *storage.Image](cacheSize)
	if err != nil {
		return errors.Wrap(errox.ServerError, err.Error())
	}

	return s.deployments.WalkByQuery(ctx, parsedQuery, func(d *storage.Deployment) error {
		containers := d.GetContainers()
		images := make([]*storage.Image, 0, len(containers))
		imageIDs := set.NewStringSet()
		for _, container := range containers {
			imgID := container.GetImage().GetId()
			// Deduplicate images by their ID.
			if imageIDs.Contains(imgID) {
				continue
			}
			imageIDs.Add(imgID)

			if img, found := imageCache.Get(imgID); found {
				images = append(images, img)
				continue
			}

			log.Infof("get image %q", imgID)
			time, _ := ctx.Deadline()
			log.Infof("context deadline: %+v", time)
			newCtx, cancel := context.WithCancel(ctx)
			defer cancel()
			img, found, err := s.images.GetImage(newCtx, imgID)
			log.Infof("get image %q DONE", imgID)
			if err != nil {
				log.Errorf("Error getting image for container %q (SHA: %s): %v", d.GetName(), container.GetId(), err)
				continue
			}
			if found {
				images = append(images, img)
				imageCache.Add(imgID, img)
			} else {
				log.Warnf("Image %q for container %q (SHA: %s) not found", imgID, d.GetName(), container.GetId())
			}
		}

		if err := srv.Send(&v1.VulnMgmtExportWorkloadsResponse{Deployment: d, Images: images}); err != nil {
			return err
		}
		return nil
	})
}
