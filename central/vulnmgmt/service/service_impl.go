package service

import (
	"context"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	podDS "github.com/stackrox/rox/central/pod/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres"
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
			v1.VulnMgmtService_VulnMgmtExportWorkloads_FullMethodName,
		},
		user.With(permissions.View(resources.Image)): {
			v1.VulnMgmtService_ImageVulnerabilities_FullMethodName,
		},
	})
	log = logging.LoggerForModule()
)

// serviceImpl provides APIs for vulnerability management.
type serviceImpl struct {
	v1.UnimplementedVulnMgmtServiceServer

	db          postgres.DB
	deployments deploymentDS.DataStore
	pods        podDS.DataStore
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
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		defer cancel()
	}

	// Begin a transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return errors.Wrap(errox.ServerError, "failed to begin transaction")
	}
	var committed bool
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	// Add transaction to context
	txCtx := postgres.ContextWithTx(ctx, tx)

	imageCache, err := lru.New[string, *storage.Image](cacheSize)
	if err != nil {
		return errors.Wrap(errox.ServerError, err.Error())
	}

	err = s.deployments.WalkByQuery(txCtx, parsedQuery, func(d *storage.Deployment) error {
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

			img, found, err := s.images.GetImage(txCtx, imgID)
			if err != nil {
				log.Errorf("Error getting image for container %q (SHA: %s): %v", d.GetName(), container.GetId(), err)
				continue
			}
			if found {
				utils.StripDatasourceNoClone(img.GetScan())
				images = append(images, img)
				imageCache.Add(imgID, img)
			} else {
				log.Warnf("Image %q for container %q (SHA: %s) not found", imgID, d.GetName(), container.GetId())
			}
		}

		// Container Image Digest is a field in pods_live_instances table which is connected to pods table via FK.
		// So the below query should return the number of pods that have live instances.
		livePodsQ := search.NewQueryBuilder().
			AddExactMatches(search.DeploymentID, d.GetId()).
			AddRegexes(search.ContainerImageDigest, ".*").
			ProtoQuery()

		livePods, err := s.pods.Count(txCtx, livePodsQ)
		if err != nil {
			log.Errorf("Error getting live pod count for deployment ID '%s'", d.GetId())
		}

		if err := srv.Send(&v1.VulnMgmtExportWorkloadsResponse{Deployment: d, Images: images, LivePods: int32(livePods)}); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Commit the transaction
	if err := tx.Commit(txCtx); err != nil {
		return err
	}
	committed = true
	return nil
}

func (s *serviceImpl) ImageVulnerabilities(ctx context.Context, _ *v1.Empty) (*v1.ImageVulnerabilitiesResponse, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, errors.Wrap(errox.ServerError, "failed to begin transaction")
	}
	var committed bool
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	txCtx := postgres.ContextWithTx(ctx, tx)

	var images []*v1.ImageVulnerabilitiesResponse_Image

	err = s.images.WalkByQuery(txCtx, search.EmptyQuery(), func(img *storage.Image) error {
		scan := img.GetScan()
		if scan == nil {
			return nil
		}

		components := scan.GetComponents()
		if len(components) == 0 {
			return nil
		}

		var responseComponents []*v1.ImageVulnerabilitiesResponse_Image_Component
		for _, comp := range components {
			vulns := comp.GetVulns()
			if len(vulns) == 0 {
				continue
			}

			vulnIDs := make([]string, 0, len(vulns))
			for _, vuln := range vulns {
				if cve := vuln.GetCve(); cve != "" {
					vulnIDs = append(vulnIDs, cve)
				}
			}

			if len(vulnIDs) == 0 {
				continue
			}

			var layerIndex int32
			if li, ok := comp.GetHasLayerIndex().(*storage.EmbeddedImageScanComponent_LayerIndex); ok {
				layerIndex = li.LayerIndex
			}

			responseComponents = append(responseComponents, &v1.ImageVulnerabilitiesResponse_Image_Component{
				Name:             comp.GetName(),
				Version:          comp.GetVersion(),
				LayerIndex:       layerIndex,
				Location:         comp.GetLocation(),
				VulnerabilityIds: vulnIDs,
			})
		}

		if len(responseComponents) > 0 {
			images = append(images, &v1.ImageVulnerabilitiesResponse_Image{
				Sha:        img.GetId(),
				Components: responseComponents,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(txCtx); err != nil {
		return nil, err
	}
	committed = true

	return &v1.ImageVulnerabilitiesResponse{Images: images}, nil
}
