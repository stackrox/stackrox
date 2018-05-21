package service

import (
	"bitbucket.org/stack-rox/apollo/central/datastore"
	"bitbucket.org/stack-rox/apollo/central/detection"
	"bitbucket.org/stack-rox/apollo/central/risk"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authn"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/idcheck"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewSensorEventService returns the SensorEventService API.
func NewSensorEventService(detector *detection.Detector, images datastore.ImageDataStore, deployments datastore.DeploymentDataStore, clusters datastore.ClusterDataStore, scorer *risk.Scorer) *SensorEventService {
	return &SensorEventService{
		detector:    detector,
		scorer:      scorer,
		images:      images,
		deployments: deployments,
		clusters:    clusters,
	}
}

// SensorEventService is the struct that manages the SensorEvent API
type SensorEventService struct {
	detector    *detection.Detector
	scorer      *risk.Scorer
	images      datastore.ImageDataStore
	deployments datastore.DeploymentDataStore
	clusters    datastore.ClusterDataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *SensorEventService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterSensorEventServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *SensorEventService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterSensorEventServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *SensorEventService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, returnErrorCode(idcheck.SensorsOnly().Authorized(ctx))
}

// ReportDeploymentEvent receives a new deployment event from a sensor.
func (s *SensorEventService) ReportDeploymentEvent(ctx context.Context, request *v1.DeploymentEvent) (*v1.DeploymentEventResponse, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "Request must include an event")
	}
	log.Infof("Processing deployment event: deployment: %s (%s), action: %s", request.GetDeployment().Id, request.GetDeployment().GetName(), request.GetAction().String())

	d := request.GetDeployment()
	if d == nil {
		return nil, status.Error(codes.InvalidArgument, "Event must include a deployment")
	}
	// We do not want to trust what clients tell us their cluster ID is;
	// let their certificates do the talking.
	s.resetClusterData(ctx, d)

	response := new(v1.DeploymentEventResponse)
	// If it's a create and we already have the deployment, ignore it.
	// We don't want new alerts, and don't need to bother the database again.
	if request.GetAction() == v1.ResourceAction_CREATE_RESOURCE {
		if _, ok, err := s.deployments.GetDeployment(d.GetId()); err != nil && ok {
			return response, nil
		}
	}

	if err := s.handlePersistence(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Update the deployment with the most recent image version
	for _, c := range d.GetContainers() {
		img, exists, err := s.images.GetImage(c.GetImage().GetName().GetSha())
		if err != nil {
			log.Error(err)
			continue
		}
		if exists {
			c.Image = img
		}
	}
	alertID, enforcement := s.detector.ProcessDeploymentEvent(d, request.GetAction())

	for _, i := range images.FromContainers(d.GetContainers()).Images() {
		if i.GetName().GetSha() == "" {
			log.Debugf("Skipping persistence of image without sha: %+v", i)
			continue
		}

		if err := s.images.UpdateImage(i); err != nil {
			log.Error(err)
		}
	}

	response.AlertId = alertID
	response.Enforcement = enforcement
	if enforcement != v1.EnforcementAction_UNSET_ENFORCEMENT {
		log.Warnf("Taking enforcement action %s against deployment %s", enforcement, request.GetDeployment().GetName())
	}

	return response, nil
}

func (s *SensorEventService) resetClusterData(ctx context.Context, d *v1.Deployment) {
	d.ClusterId = ""
	d.ClusterName = ""

	identity, err := authn.FromTLSContext(ctx)
	if err != nil {
		// This should be impossible, because we have already passed through MTLS auth.
		log.Errorf("Couldn't get cluster identity: %s", err)
		return
	}

	d.ClusterId = identity.Name.Identifier
	cluster, clusterExists, err := s.clusters.GetCluster(d.ClusterId)
	switch {
	case err != nil:
		log.Warnf("Couldn't get name of cluster: %s", err)
	case !clusterExists:
		log.Warnf("Couldn't find cluster '%s'", d.ClusterId)
	default:
		d.ClusterName = cluster.GetName()
	}
}

func (s *SensorEventService) handlePersistence(event *v1.DeploymentEvent) error {
	action := event.GetAction()
	deployment := event.GetDeployment()
	switch action {
	case v1.ResourceAction_PREEXISTING_RESOURCE:
		fallthrough
	case v1.ResourceAction_CREATE_RESOURCE:
		if err := s.deployments.UpdateDeployment(deployment); err != nil {
			log.Errorf("unable to add deployment %s: %s", deployment.GetId(), err)
			return err
		}
	case v1.ResourceAction_UPDATE_RESOURCE:
		if err := s.deployments.UpdateDeployment(deployment); err != nil {
			log.Errorf("unable to update deployment %s: %s", deployment.GetId(), err)
			return err
		}
	case v1.ResourceAction_REMOVE_RESOURCE:
		if err := s.deployments.RemoveDeployment(deployment.GetId()); err != nil {
			log.Errorf("unable to remove deployment %s: %s", deployment.GetId(), err)
			return err
		}
	default:
		log.Warnf("unknown action: %s", action)
		return nil // Be interoperable: don't reject these requests.
	}
	return nil
}
