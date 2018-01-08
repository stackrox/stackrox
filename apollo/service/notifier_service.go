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
func (s *NotifierService) GetNotifier(ctx context.Context, request *v1.GetNotifierRequest) (*v1.Notifier, error) {
	if request == nil || request.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "Notifier name must be provided")
	}
	notifierWithSecret, exists, err := s.storage.GetNotifier(request.Name)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Notifier %v not found", request.Name))
	}
	notifierWithoutSecret := &v1.Notifier{
		Name:       notifierWithSecret.Name,
		Type:       notifierWithSecret.Type,
		UiEndpoint: notifierWithSecret.UiEndpoint,
		Enabled:    notifierWithSecret.Enabled,
		Config:     secrets.ScrubSecrets(notifierWithSecret.Config),
	}
	return notifierWithoutSecret, nil
}

// GetNotifiers retrieves all notifiers that match the request filters
func (s *NotifierService) GetNotifiers(ctx context.Context, request *v1.GetNotifiersRequest) (*v1.GetNotifiersResponse, error) {
	notifiersWithSecrets, err := s.storage.GetNotifiers(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	notifiersWithoutSecrets := make([]*v1.Notifier, 0, len(notifiersWithSecrets))
	for _, notifierWithSecret := range notifiersWithSecrets {
		notifiersWithoutSecrets = append(notifiersWithoutSecrets, &v1.Notifier{
			Name:       notifierWithSecret.Name,
			Type:       notifierWithSecret.Type,
			UiEndpoint: notifierWithSecret.UiEndpoint,
			Enabled:    notifierWithSecret.Enabled,
			Config:     secrets.ScrubSecrets(notifierWithSecret.Config),
		})
	}
	return &v1.GetNotifiersResponse{Notifiers: notifiersWithoutSecrets}, nil
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
func (s *NotifierService) PostNotifier(ctx context.Context, request *v1.Notifier) (*empty.Empty, error) {
	notifierCreator, ok := notifiers.Registry[request.Type]
	if !ok {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Notifier type %v is not a valid notifier type", request.Type))
	}
	notifier, err := notifierCreator(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if err := s.storage.AddNotifier(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.processor.UpdateNotifier(notifier)
	return &empty.Empty{}, nil
}

// DeleteNotifier deletes a notifier from the system
func (s *NotifierService) DeleteNotifier(ctx context.Context, request *v1.DeleteNotifierRequest) (*empty.Empty, error) {
	if request == nil || request.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "Notifier name must be provided")
	}
	if err := s.storage.RemoveNotifier(request.Name); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	s.processor.RemoveNotifier(request.Name)
	return &empty.Empty{}, nil
}
