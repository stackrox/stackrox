package service

import (
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	buildTimeDetection "github.com/stackrox/rox/central/detection/buildtime"
	deployTimeDetection "github.com/stackrox/rox/central/detection/deploytime"
	runTimeDetectiomn "github.com/stackrox/rox/central/detection/runtime"
	"github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/central/notifier/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/secrets"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Notifier)): {
			"/v1.NotifierService/GetNotifier",
			"/v1.NotifierService/GetNotifiers",
		},
		user.With(permissions.Modify(resources.Notifier)): {
			"/v1.NotifierService/PutNotifier",
			"/v1.NotifierService/PostNotifier",
			"/v1.NotifierService/TestNotifier",
			"/v1.NotifierService/DeleteNotifier",
		},
	})
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct {
	storage   store.Store
	processor processor.Processor

	buildTimePolicies  buildTimeDetection.PolicySet
	deployTimePolicies deployTimeDetection.PolicySet
	runTimePolicies    runTimeDetectiomn.PolicySet
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterNotifierServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterNotifierServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetNotifier retrieves all registries that matches the request filters
func (s *serviceImpl) GetNotifier(ctx context.Context, request *v1.ResourceByID) (*v1.Notifier, error) {
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
	secrets.ScrubSecretsFromStruct(notifier)
	return notifier, nil
}

// GetNotifiers retrieves all notifiers that match the request filters
func (s *serviceImpl) GetNotifiers(ctx context.Context, request *v1.GetNotifiersRequest) (*v1.GetNotifiersResponse, error) {
	notifiers, err := s.storage.GetNotifiers(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	for _, n := range notifiers {
		secrets.ScrubSecretsFromStruct(n)
	}
	return &v1.GetNotifiersResponse{Notifiers: notifiers}, nil
}

func validateNotifier(notifier *v1.Notifier) error {
	errorList := errorhelpers.NewErrorList("Validation")
	if notifier.GetName() == "" {
		errorList.AddString("Notifier name must be defined")
	}
	if notifier.GetType() == "" {
		errorList.AddString("Notifier type must be defined")
	}
	if notifier.GetUiEndpoint() == "" {
		errorList.AddString("Notifier UI endpoint must be defined")
	}
	return errorList.ToError()
}

// PutNotifier updates a notifier in the system
func (s *serviceImpl) PutNotifier(ctx context.Context, request *v1.Notifier) (*v1.Empty, error) {
	if err := validateNotifier(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
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
	return &v1.Empty{}, nil
}

// PostNotifier inserts a new registry into the system if it doesn't already exist
func (s *serviceImpl) PostNotifier(ctx context.Context, request *v1.Notifier) (*v1.Notifier, error) {
	if err := validateNotifier(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new notifier")
	}
	notifier, err := notifiers.CreateNotifier(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	id, err := s.storage.AddNotifier(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	request.Id = id
	s.processor.UpdateNotifier(notifier)
	return request, nil
}

// TestNotifier tests to see if the config is setup properly
func (s *serviceImpl) TestNotifier(ctx context.Context, request *v1.Notifier) (*v1.Empty, error) {
	if err := validateNotifier(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	notifier, err := notifiers.CreateNotifier(request)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := notifier.Test(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &v1.Empty{}, nil
}

// DeleteNotifier deletes a notifier from the system
func (s *serviceImpl) DeleteNotifier(ctx context.Context, request *v1.DeleteNotifierRequest) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Notifier id must be provided")
	}

	n, err := s.GetNotifier(ctx, &v1.ResourceByID{Id: request.GetId()})
	if err != nil {
		return nil, err
	}

	err = s.deleteNotifiersFromPolicies(n.GetId())

	if err != nil {
		log.Error(err)
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("Notifier is still in use by policies. Error: %s", err))
	}

	if err := s.storage.RemoveNotifier(request.GetId()); err != nil {
		return nil, err
	}

	s.processor.RemoveNotifier(request.GetId())
	s.buildTimePolicies.RemoveNotifier(request.GetId())
	s.deployTimePolicies.RemoveNotifier(request.GetId())
	s.runTimePolicies.RemoveNotifier(request.GetId())
	return &v1.Empty{}, nil
}

func (s *serviceImpl) deleteNotifiersFromPolicies(notifierID string) error {

	err := s.buildTimePolicies.RemoveNotifier(notifierID)
	if err != nil {
		return err
	}

	err = s.deployTimePolicies.RemoveNotifier(notifierID)
	if err != nil {
		return err
	}

	err = s.runTimePolicies.RemoveNotifier(notifierID)
	if err != nil {
		return err
	}

	return nil
}
