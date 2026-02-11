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
		user.With(
			permissions.View(resources.Cluster),
			permissions.View(resources.Namespace),
			permissions.View(resources.Deployment),
			permissions.View(resources.Image)): {
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

func (s *serviceImpl) ImageVulnerabilities(ctx context.Context, req *v1.ImageVulnerabilitiesRequest) (*v1.ImageVulnerabilitiesResponse, error) {
	parsedQuery, err := search.ParseQuery(req.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errox.InvalidArgs.CausedBy(err)
	}

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

	images := make(map[string]*v1.ImageVulnerabilitiesResponse_Image)

	err = s.images.WalkByQuery(txCtx, parsedQuery, func(img *storage.Image) error {
		components, err := s.getVulnerableImageComponents(img)
		if err != nil {
			return err
		}
		if len(components) == 0 {
			return nil
		}
		workloadIDs, err := s.getImageWorkloadIDs(ctx, parsedQuery, img.GetId())
		if err != nil {
			return errors.Wrapf(err, "failed to get workload IDs for image %s", img.GetId())
		}
		if !req.GetIncludeUndeployed() && len(workloadIDs) == 0 {
			return nil
		}

		images[img.GetId()] = &v1.ImageVulnerabilitiesResponse_Image{
			Components:  components,
			WorkloadIds: workloadIDs,
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

// getVulnerableImageComponents returns vulnerable image components.
func (s *serviceImpl) getVulnerableImageComponents(img *storage.Image) ([]*v1.ImageVulnerabilitiesResponse_Image_Component, error) {
	components := img.GetScan().GetComponents()
	if len(components) == 0 {
		return nil, nil
	}

	responseComponents := make([]*v1.ImageVulnerabilitiesResponse_Image_Component, 0, len(components))

	for _, comp := range components {
		if responseComp := transformComponentToResponse(comp); responseComp != nil {
			responseComponents = append(responseComponents, responseComp)
		}
	}

	return responseComponents, nil
}

// transformComponentToResponse converts a storage.EmbeddedImageScanComponent to
// the response format.
// Returns nil if the component has no vulnerabilities to report.
func transformComponentToResponse(comp *storage.EmbeddedImageScanComponent) *v1.ImageVulnerabilitiesResponse_Image_Component {
	vulns := comp.GetVulns()
	if len(vulns) == 0 {
		return nil
	}

	responseVulns := make([]*v1.ImageVulnerabilitiesResponse_Image_Component_Vulnerability, 0, len(vulns))
	for _, vuln := range vulns {
		if responseVuln := transformVulnerabilityToResponse(vuln); responseVuln != nil {
			responseVulns = append(responseVulns, responseVuln)
		}
	}

	if len(responseVulns) == 0 {
		return nil
	}
	layer := int32(-1)
	if comp.GetHasLayerIndex() != nil {
		layer = comp.GetLayerIndex()
	}
	return &v1.ImageVulnerabilitiesResponse_Image_Component{
		Name:            comp.GetName(),
		Version:         comp.GetVersion(),
		LayerIndex:      layer,
		Location:        comp.GetLocation(),
		Vulnerabilities: responseVulns,
	}
}

// transformVulnerabilityToResponse converts a storage.EmbeddedVulnerability to
// the response format.
// Returns nil if the vulnerability has no CVE ID.
func transformVulnerabilityToResponse(vuln *storage.EmbeddedVulnerability) *v1.ImageVulnerabilitiesResponse_Image_Component_Vulnerability {
	if vuln.GetCve() == "" {
		return nil
	}

	vulnerability := &v1.ImageVulnerabilitiesResponse_Image_Component_Vulnerability{
		Id:                    vuln.GetCve(),
		FirstSystemOccurrence: vuln.GetFirstSystemOccurrence(),
		FirstImageOccurrence:  vuln.GetFirstImageOccurrence(),
	}

	if vuln.GetSuppressed() {
		vulnerability.Suppression = &v1.ImageVulnerabilitiesResponse_Image_Component_Vulnerability_Suppression{
			SuppressActivation: vuln.GetSuppressActivation(),
			SuppressExpiry:     vuln.GetSuppressExpiry(),
		}
	}

	return vulnerability
}

func (s *serviceImpl) getImageWorkloadIDs(ctx context.Context, query *v1.Query, imageID string) ([]string, error) {
	imageQuery := search.NewQueryBuilder().
		AddExactMatches(search.ImageSHA, imageID).
		ProtoQuery()

	combinedQuery := search.ConjunctionQuery(query, imageQuery)

	workloadIDs := set.NewStringSet()

	err := s.deployments.WalkByQuery(ctx, combinedQuery, func(deployment *storage.Deployment) error {
		for _, container := range deployment.GetContainers() {
			if container.GetImage().GetId() == imageID {
				workloadIDs.Add(deployment.GetId())
				break
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return workloadIDs.AsSlice(), nil
}
