package service

import (
	"io"

	clusterDataStore "bitbucket.org/stack-rox/apollo/central/cluster/datastore"
	deploymentDataStore "bitbucket.org/stack-rox/apollo/central/deployment/datastore"
	deploymentEventStore "bitbucket.org/stack-rox/apollo/central/deploymentevent/store"
	"bitbucket.org/stack-rox/apollo/central/detection"
	imageDataStore "bitbucket.org/stack-rox/apollo/central/image/datastore"
	"bitbucket.org/stack-rox/apollo/central/risk"
	"bitbucket.org/stack-rox/apollo/central/sensorevent/service/pipeline"
	"bitbucket.org/stack-rox/apollo/central/sensorevent/service/queue"
	"bitbucket.org/stack-rox/apollo/central/service"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authn"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/idcheck"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Service is the struct that manages the SensorEvent API
type serviceImpl struct {
	detector *detection.Detector
	scorer   *risk.Scorer

	deploymentEvents deploymentEventStore.Store

	images      imageDataStore.DataStore
	deployments deploymentDataStore.DataStore
	clusters    clusterDataStore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterSensorEventServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterSensorEventServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, service.ReturnErrorCode(idcheck.SensorsOnly().Authorized(ctx))
}

// RecordEvent takes in a stream of deployment events and outputs a stream of alerts and enforcement actions.
func (s *serviceImpl) RecordEvent(stream v1.SensorEventService_RecordEventServer) error {
	eventProcessor := pipeline.NewPipeline(stream.Context(), s.clusters, s.deployments, s.images, s.detector)
	pendingEvents := queue.NewChanneledEventQueue(s.deploymentEvents)

	identity, err := authn.FromTLSContext(stream.Context())
	if err != nil {
		return err
	}
	clientClusterID := identity.Name.Identifier

	if err := pendingEvents.Open(clientClusterID); err != nil {
		return err
	}

	go receiveMessages(clientClusterID, stream, pendingEvents)
	sendMessages(stream, pendingEvents, eventProcessor)
	return nil
}

// receiveMessages loops over the input and adds it to the pending queue.
func receiveMessages(clientClusterID string, stream v1.SensorEventService_RecordEventServer, pendingEvents queue.ChanneledEventQueue) {
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
			log.Error("error dequeueing deployment event", err)
			return
		}

		// Fill the cluster id.
		event.Deployment.ClusterId = clientClusterID
		pendingEvents.InChannel() <- event
	}
}

// sendMessages grabs items from the queue, processes them, and sends them back to sensor.
func sendMessages(stream v1.SensorEventService_RecordEventServer, pendingEvents queue.ChanneledEventQueue, eventProcessor pipeline.Pipeline) {
	for {
		event, ok := <-pendingEvents.OutChannel()
		// Looping stops when the output from pending events closes.
		if !ok {
			return
		}

		response, err := eventProcessor.Run(event)
		if err != nil {
			log.Error("error processing deployment event response", err)
			continue
		}

		if err := stream.Send(response); err != nil {
			log.Error("error sending deployment event response", err)
		}
	}
}
