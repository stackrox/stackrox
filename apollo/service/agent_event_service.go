package service

import (
	"bitbucket.org/stack-rox/apollo/apollo/alerts"
	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/detection/image_processor"
	"bitbucket.org/stack-rox/apollo/apollo/notifications"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewAgentEventService returns the AgentEventService API.
func NewAgentEventService(imageProcessor *imageprocessor.ImageProcessor, notificationsProcessor *notifications.Processor, database db.Storage) *AgentEventService {
	return &AgentEventService{
		imageProcessor:        imageProcessor,
		notificationProcessor: notificationsProcessor,
		stalenessHandler:      alerts.NewStalenessHandler(database),
		storage:               database,
	}
}

// AgentEventService is the struct that manages the AgentEvent API
type AgentEventService struct {
	imageProcessor        *imageprocessor.ImageProcessor
	notificationProcessor *notifications.Processor
	stalenessHandler      alerts.StalenessHandler
	storage               db.Storage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *AgentEventService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterAgentEventServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *AgentEventService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterAgentEventServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// ReportDeploymentEvent receives a new deployment event from an agent.
func (s *AgentEventService) ReportDeploymentEvent(ctx context.Context, request *v1.DeploymentEvent) (*empty.Empty, error) {
	if request == nil {
		return nil, status.Error(codes.InvalidArgument, "Request must include an event")
	}
	log.Infof("Processing deployment event %+v", request)

	d := request.GetDeployment()
	if d == nil {
		return nil, status.Error(codes.InvalidArgument, "Event must include a deployment")
	}

	// If it's a create and we already have the deployment, ignore it.
	// We don't want new alerts, and don't need to bother the database again.
	if request.GetAction() == v1.ResourceAction_CREATE_RESOURCE {
		if _, ok, err := s.storage.GetDeployment(d.GetId()); err != nil && ok {
			return &empty.Empty{}, nil
		}
	}

	if err := s.handlePersistence(request); err != nil {
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}

	s.stalenessHandler.UpdateStaleness(request)

	alerts, err := s.imageProcessor.Process(d)
	if err != nil {
		log.Error(err)
		return &empty.Empty{}, status.Error(codes.Internal, err.Error())
	}
	for _, i := range images.FromContainers(d.GetContainers()).Images() {
		if err := s.storage.AddImage(i); err != nil {
			log.Error(err)
		}
	}
	for _, alert := range alerts {
		log.Warnf("Alert Generated: %v with Severity %v due to image policy %v", alert.Id, alert.GetPolicy().GetSeverity().String(), alert.GetPolicy().GetName())
		for _, violation := range alert.GetViolations() {
			log.Warnf("\t %v", violation.Message)
		}
		if err := s.storage.AddAlert(alert); err != nil {
			log.Error(err)
		}
		s.notificationProcessor.Process(alert)
	}
	return &empty.Empty{}, nil
}

func (s *AgentEventService) handlePersistence(event *v1.DeploymentEvent) error {
	action := event.GetAction()
	deployment := event.GetDeployment()
	switch action {
	case v1.ResourceAction_CREATE_RESOURCE:
		if err := s.storage.UpdateDeployment(deployment); err != nil {
			log.Errorf("unable to add deployment %s: %s", deployment.GetId(), err)
			return err
		}
	case v1.ResourceAction_UPDATE_RESOURCE:
		if err := s.storage.UpdateDeployment(deployment); err != nil {
			log.Errorf("unable to update deployment %s: %s", deployment.GetId(), err)
			return err
		}
	case v1.ResourceAction_REMOVE_RESOURCE:
		if err := s.storage.RemoveDeployment(deployment.GetId()); err != nil {
			log.Errorf("unable to remove deployment %s: %s", deployment.GetId(), err)
			return err
		}
	default:
		log.Warnf("unknown action: %s", action)
		return nil // Be interoperable: don't reject these requests.
	}
	return nil
}
