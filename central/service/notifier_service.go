package service

import (
	"fmt"
	"sort"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/notifications"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/notifications/notifiers"
	"bitbucket.org/stack-rox/apollo/pkg/secrets"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewNotifierService returns the NotifierService API.
func NewNotifierService(storage db.NotifierStorage, processor *notifications.Processor, detector policyDetector) *NotifierService {
	return &NotifierService{
		storage:   storage,
		processor: processor,
		detector:  detector,
	}
}

// NotifierService is the struct that manages the Notifier API
type NotifierService struct {
	storage   db.NotifierStorage
	processor *notifications.Processor
	detector  policyDetector
}

type policyDetector interface {
	RemoveNotifier(id string)
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
	s.populatePolicies(notifier)
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
		s.populatePolicies(n)
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
func (s *NotifierService) DeleteNotifier(ctx context.Context, request *v1.DeleteNotifierRequest) (*empty.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Notifier id must be provided")
	}
	n, err := s.GetNotifier(ctx, &v1.ResourceByID{Id: request.GetId()})
	if err != nil {
		return nil, err
	}

	if !request.GetForce() && len(n.Policies) != 0 {
		m := jsonpb.Marshaler{}
		policiesOnly := &v1.Notifier{
			Policies: n.GetPolicies(),
		}
		jsonString, err := m.MarshalToString(policiesOnly)

		if err != nil {
			log.Error(err)
			return nil, status.Error(codes.FailedPrecondition, "Notifier is in use by policies")
		}

		return nil, status.Errorf(codes.FailedPrecondition, "Notifier is in use by policies: %s", jsonString)
	}

	if err := s.storage.RemoveNotifier(request.GetId()); err != nil {
		return nil, returnErrorCode(err)
	}
	s.processor.RemoveNotifier(request.GetId())
	s.detector.RemoveNotifier(request.GetId())
	return &empty.Empty{}, nil
}

func (s *NotifierService) populatePolicies(notifier *v1.Notifier) {
	policies := s.processor.GetIntegratedPolicies(notifier.GetId())

	for _, p := range policies {
		notifier.Policies = append(notifier.Policies, &v1.Notifier_Policy{Id: p.GetId(), Name: p.GetName()})
	}

	sort.Slice(notifier.Policies, func(i, j int) bool {
		return notifier.Policies[i].Name < notifier.Policies[j].Name
	})
}
