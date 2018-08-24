package service

import (
	"io"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	clusterDataStore "github.com/stackrox/rox/central/cluster/datastore"
	deploymentDataStore "github.com/stackrox/rox/central/deployment/datastore"
	deployTimeDetection "github.com/stackrox/rox/central/detection/deploytime"
	imageDataStore "github.com/stackrox/rox/central/image/datastore"
	namespaceStore "github.com/stackrox/rox/central/namespace/store"
	networkPolicyStore "github.com/stackrox/rox/central/networkpolicies/store"
	"github.com/stackrox/rox/central/risk"
	"github.com/stackrox/rox/central/sensorevent/service/pipeline"
	"github.com/stackrox/rox/central/sensorevent/service/queue"
	sensorEventStore "github.com/stackrox/rox/central/sensorevent/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"github.com/stackrox/rox/pkg/logging"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

var logger = logging.LoggerForModule()

// Service is the struct that manages the SensorEvent API
type serviceImpl struct {
	detector deployTimeDetection.Detector
	scorer   risk.Scorer

	deploymentEvents sensorEventStore.Store

	images          imageDataStore.DataStore
	deployments     deploymentDataStore.DataStore
	clusters        clusterDataStore.DataStore
	networkPolicies networkPolicyStore.Store
	namespaces      namespaceStore.Store

	deploymentPipeline    pipeline.Pipeline
	networkPolicyPipeline pipeline.Pipeline
	namespacePipeline     pipeline.Pipeline
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterSensorEventServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterSensorEventServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.SensorsOnly().Authorized(ctx, fullMethodName)
}

// RecordEvent takes in a stream of deployment events and outputs a stream of alerts and enforcement actions.
func (s *serviceImpl) RecordEvent(stream v1.SensorEventService_RecordEventServer) error {
	pendingEvents := queue.NewChanneledEventQueue(s.deploymentEvents)

	identity, err := authn.FromTLSContext(stream.Context())
	if err != nil {
		return err
	}
	clientClusterID := identity.Name.Identifier

	if err := pendingEvents.Open(clientClusterID); err != nil {
		return err
	}
	go s.receiveMessages(clientClusterID, stream, pendingEvents)
	s.sendMessages(stream, pendingEvents)
	return nil
}

// receiveMessages loops over the input and adds it to the pending queue.
func (s *serviceImpl) receiveMessages(clientClusterID string, stream v1.SensorEventService_RecordEventServer, pendingEvents queue.ChanneledEventQueue) {
	// When the receive channel closes, and we return, we need to stop our intermediate processing, this causes the other
	// processing loop (sendMessages) to finish and return as well, ending the service call.
	defer pendingEvents.Close()

	for {
		event, err := stream.Recv()
		// Looping stops when the stream closes, or returns an error.
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Error("error dequeueing deployment event: ", err)
			return
		}
		event.ClusterId = clientClusterID

		pendingEvents.InChannel() <- event
	}
}

// sendMessages grabs items from the queue, processes them, and sends them back to sensor.
func (s *serviceImpl) sendMessages(stream v1.SensorEventService_RecordEventServer, pendingEvents queue.ChanneledEventQueue) {
	for {
		event, ok := <-pendingEvents.OutChannel()
		// Looping stops when the output from pending events closes.
		if !ok {
			return
		}

		var eventPipeline pipeline.Pipeline
		switch x := event.Resource.(type) {
		case *v1.SensorEvent_Deployment:
			eventPipeline = s.deploymentPipeline
		case *v1.SensorEvent_NetworkPolicy:
			eventPipeline = s.networkPolicyPipeline
		case *v1.SensorEvent_Namespace:
			eventPipeline = s.namespacePipeline
		case *v1.SensorEvent_Indicator:
			// TODO: Implement actual handling
			log.Infof("Obtained an Indicator event: %+v", x)
		case nil:
			logger.Errorf("Resource field is empty")
			return
		default:
			logger.Errorf("No resource with type %T", x)
			return
		}

		sensorResponse, err := eventPipeline.Run(event)
		if err != nil {
			log.Error("error processing response: %s", err)
			continue
		}

		if err := stream.Send(sensorResponse); err != nil {
			log.Error("error sending deployment event response", err)
		}
	}
}
