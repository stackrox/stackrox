package service

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/notifications"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers"
	"bitbucket.org/stack-rox/apollo/pkg/secrets"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewNotifierService returns the NotifierService API.
func NewNotifierService(storage db.NotifierStorage, processor *notifications.Processor) *NotifierService {
	return &NotifierService{
		storage:   storage,
		processor: processor,
	}
}

// NotifierService is the struct that manages the Notifier API
type NotifierService struct {
	storage   db.NotifierStorage
	processor *notifications.Processor
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *NotifierService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterNotifierServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *NotifierService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterNotifierServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// GetNotifier retrieves all registries that matches the request filters
func (s *NotifierService) GetNotifier(ctx context.Context, request *v1.ResourceByID) (*v1.Notifier, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Notifier id must be provided")
	}
	notifier, exists, err := s.storage.GetNotifier(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Notifier %v not found", request.GetId()))
	}
	notifier.Config = secrets.ScrubSecrets(notifier.Config)
	return notifier, nil
}

// GetNotifiers retrieves all notifiers that match the request filters
func (s *NotifierService) GetNotifiers(ctx context.Context, request *v1.GetNotifiersRequest) (*v1.GetNotifiersResponse, error) {
	notifiers, err := s.storage.GetNotifiers(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	for _, n := range notifiers {
		n.Config = secrets.ScrubSecrets(n.Config)
	}
	return &v1.GetNotifiersResponse{Notifiers: notifiers}, nil
}

// PutNotifier updates a notifier in the system
func (s *NotifierService) PutNotifier(ctx context.Context, request *v1.Notifier) (*empty.Empty, error) {
	notifierCreator, ok := notifiers.Registry[request.Type]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Notifier type %v is not a valid notifier type", request.Type))
	}
	notifier, err := notifierCreator(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := s.storage.UpdateNotifier(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.processor.UpdateNotifier(notifier)
	return &empty.Empty{}, nil
}

// PostNotifier inserts a new registry into the system if it doesn't already exist
func (s *NotifierService) PostNotifier(ctx context.Context, request *v1.Notifier) (*v1.Notifier, error) {
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new notifier")
	}
	notifierCreator, ok := notifiers.Registry[request.Type]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Notifier type %v is not a valid notifier type", request.Type))
	}
	notifier, err := notifierCreator(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	id, err := s.storage.AddNotifier(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	request.Id = id
	s.processor.UpdateNotifier(notifier)
	return request, nil
}

// DeleteNotifier deletes a notifier from the system
func (s *NotifierService) DeleteNotifier(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Notifier id must be provided")
	}
	if err := s.storage.RemoveNotifier(request.GetId()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.processor.RemoveNotifier(request.GetId())
	return &empty.Empty{}, nil
}
