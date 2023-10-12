package service

import (
	"context"
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/policycleaner"
	notifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/endpoints"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/integrationhealth"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/notifier"
	pkgNotifiers "github.com/stackrox/rox/pkg/notifiers"
	"github.com/stackrox/rox/pkg/notifiers/splunk"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/secrets"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Integration)): {
			"/v1.NotifierService/GetNotifier",
			"/v1.NotifierService/GetNotifiers",
		},
		user.With(permissions.Modify(resources.Integration)): {
			"/v1.NotifierService/PutNotifier",
			"/v1.NotifierService/PostNotifier",
			"/v1.NotifierService/TestNotifier",
			"/v1.NotifierService/DeleteNotifier",
			"/v1.NotifierService/TestUpdatedNotifier",
			"/v1.NotifierService/UpdateNotifier",
		},
	})
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct {
	v1.UnimplementedNotifierServiceServer

	storage   datastore.DataStore
	processor notifier.Processor
	reporter  integrationhealth.Reporter
	cryptoKey string

	policyCleaner policycleaner.PolicyCleaner
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
func (s *serviceImpl) GetNotifier(ctx context.Context, request *v1.ResourceByID) (*storage.Notifier, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "notifier id must be provided")
	}
	notifier, exists, err := s.storage.GetScrubbedNotifier(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "notifier %v not found", request.GetId())
	}

	return notifier, nil
}

// GetNotifiers retrieves all notifiers that match the request filters
func (s *serviceImpl) GetNotifiers(ctx context.Context, _ *v1.GetNotifiersRequest) (*v1.GetNotifiersResponse, error) {
	scrubbedNotifiers, err := s.storage.GetScrubbedNotifiers(ctx)
	if err != nil {
		return nil, err
	}

	return &v1.GetNotifiersResponse{Notifiers: scrubbedNotifiers}, nil
}

func validateNotifier(notifier *storage.Notifier) error {
	if notifier == nil {
		return errors.New("empty notifier")
	}
	errorList := errorhelpers.NewErrorList("Validation")
	if notifier.GetName() == "" {
		errorList.AddString("notifier name must be defined")
	}
	if notifier.GetType() == "" {
		errorList.AddString("notifier type must be defined")
	}
	if notifier.GetUiEndpoint() == "" {
		errorList.AddString("notifier UI endpoint must be defined")
	}
	if err := endpoints.ValidateEndpoints(notifier.Config); err != nil {
		errorList.AddWrap(err, "invalid endpoint")
	}
	return errorList.ToError()
}

// PutNotifier updates a notifier configuration, without stored credential reconciliation
func (s *serviceImpl) PutNotifier(ctx context.Context, notifier *storage.Notifier) (*v1.Empty, error) {
	return s.UpdateNotifier(ctx, &v1.UpdateNotifierRequest{Notifier: notifier, UpdatePassword: true})
}

// UpdateNotifier updates a notifier configuration
func (s *serviceImpl) UpdateNotifier(ctx context.Context, request *v1.UpdateNotifierRequest) (*v1.Empty, error) {
	if err := validateNotifier(request.GetNotifier()); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := s.reconcileUpdateNotifierRequest(ctx, request); err != nil {
		return nil, err
	}
	notifierCreator, ok := pkgNotifiers.Registry[request.GetNotifier().GetType()]
	if !ok {
		return nil, errors.Wrapf(errox.InvalidArgs, "notifier type %v is not a valid notifier type", request.GetNotifier().GetType())
	}
	upgradeNotifierConfig(request.GetNotifier())
	if request.GetUpdatePassword() {
		_, err := notifierUtils.SecureNotifier(request.GetNotifier(), s.cryptoKey)
		if err != nil {
			// Don't send out error from crypto lib
			return nil, errors.New("Error securing notifier")
		}
	}
	notifier, err := notifierCreator(request.GetNotifier())
	if err != nil {
		return nil, err
	}
	if err := s.storage.UpdateNotifier(ctx, request.GetNotifier()); err != nil {
		return nil, err
	}
	s.processor.UpdateNotifier(ctx, notifier)
	return &v1.Empty{}, nil
}

// PostNotifier inserts a new registry into the system if it doesn't already exist
func (s *serviceImpl) PostNotifier(ctx context.Context, request *storage.Notifier) (*storage.Notifier, error) {
	if err := validateNotifier(request); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if request.GetId() != "" {
		return nil, errors.Wrap(errox.InvalidArgs, "id field should be empty when posting a new notifier")
	}
	upgradeNotifierConfig(request)
	_, err := notifierUtils.SecureNotifier(request, s.cryptoKey)
	if err != nil {
		// Don't send out error from crypto lib
		return nil, errors.New("Error securing notifier")
	}
	notifier, err := pkgNotifiers.CreateNotifier(request)
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	id, err := s.storage.AddNotifier(ctx, request)
	if err != nil {
		return nil, err
	}
	request.Id = id
	s.processor.UpdateNotifier(ctx, notifier)

	if err = s.reporter.Register(request.Id, request.Name, storage.IntegrationHealth_NOTIFIER); err != nil {
		return nil, err
	}
	return request, nil
}

// TestNotifier tests to see if the config is setup properly, without stored credential reconciliation
func (s *serviceImpl) TestNotifier(ctx context.Context, notifier *storage.Notifier) (*v1.Empty, error) {
	return s.TestUpdatedNotifier(ctx, &v1.UpdateNotifierRequest{Notifier: notifier, UpdatePassword: true})
}

// TestUpdatedNotifier tests to see if the config is setup properly
func (s *serviceImpl) TestUpdatedNotifier(ctx context.Context, request *v1.UpdateNotifierRequest) (*v1.Empty, error) {
	if err := validateNotifier(request.GetNotifier()); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := s.reconcileUpdateNotifierRequest(ctx, request); err != nil {
		return nil, err
	}
	if request.GetUpdatePassword() {
		_, err := notifierUtils.SecureNotifier(request.GetNotifier(), s.cryptoKey)
		if err != nil {
			// Don't send out error from crypto lib
			return nil, errors.New("Error securing notifier")
		}
	}
	notifier, err := pkgNotifiers.CreateNotifier(request.GetNotifier())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	defer func() {
		if err := notifier.Close(ctx); err != nil {
			log.Warn("failed to close temporary notifier instance", logging.Err(err))
		}
	}()

	if err := notifier.Test(ctx); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	return &v1.Empty{}, nil
}

// DeleteNotifier deletes a notifier from the system
func (s *serviceImpl) DeleteNotifier(ctx context.Context, request *v1.DeleteNotifierRequest) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "notifier id must be provided")
	}

	n, err := s.GetNotifier(ctx, &v1.ResourceByID{Id: request.GetId()})
	if err != nil {
		return nil, err
	}

	err = s.policyCleaner.DeleteNotifierFromPolicies(n.GetId())
	if err != nil {
		log.Error(err)
		return nil, status.Error(codes.FailedPrecondition, fmt.Sprintf("notifier is still in use by policies. Error: %s", err))
	}

	if err := s.storage.RemoveNotifier(ctx, request.GetId()); err != nil {
		return nil, err
	}

	s.processor.RemoveNotifier(ctx, request.GetId())
	if err := s.reporter.RemoveIntegrationHealth(request.GetId()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) reconcileUpdateNotifierRequest(ctx context.Context, updateRequest *v1.UpdateNotifierRequest) error {
	if updateRequest.GetUpdatePassword() {
		return nil
	}
	if updateRequest.GetNotifier() == nil {
		return errors.Wrap(errox.InvalidArgs, "request is missing notifier config")
	}
	if updateRequest.GetNotifier().GetId() == "" {
		return errors.Wrap(errox.InvalidArgs, "id required for stored credential reconciliation")
	}
	existingNotifierConfig, exists, err := s.storage.GetNotifier(ctx, updateRequest.GetNotifier().GetId())
	if err != nil {
		return err
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "notifier integration %s not found", updateRequest.GetNotifier().GetId())
	}
	if err := reconcileNotifierConfigWithExisting(updateRequest.GetNotifier(), existingNotifierConfig); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	return nil
}

func reconcileNotifierConfigWithExisting(updated, existing *storage.Notifier) error {
	if updated.GetConfig() == nil {
		return errors.New("the request doesn't have a valid notifier config")
	}
	return secrets.ReconcileScrubbedStructWithExisting(updated, existing)
}

func upgradeNotifierConfig(notifier *storage.Notifier) {
	// UpgradeNotifierConfig applies upgrades to allow for legacy requests to be
	// converted to new formats
	splunk.UpgradeNotifierConfig(notifier)
}
