package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/externalbackups/datastore"
	"github.com/stackrox/rox/central/externalbackups/manager"
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
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Integration)): {
			"/v1.ExternalBackupService/GetExternalBackup",
			"/v1.ExternalBackupService/GetExternalBackups",
		},
		user.With(permissions.Modify(resources.Integration)): {
			"/v1.ExternalBackupService/PutExternalBackup",
			"/v1.ExternalBackupService/PostExternalBackup",
			"/v1.ExternalBackupService/TestExternalBackup",
			"/v1.ExternalBackupService/DeleteExternalBackup",
			"/v1.ExternalBackupService/TriggerExternalBackup",
			"/v1.ExternalBackupService/UpdateExternalBackup",
			"/v1.ExternalBackupService/TestUpdatedExternalBackup",
		},
	})
)

// serviceImpl is the struct that manages the external backups API
type serviceImpl struct {
	v1.UnimplementedExternalBackupServiceServer

	manager   manager.Manager
	reporter  integrationhealth.Reporter
	dataStore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterExternalBackupServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterExternalBackupServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetExternalBackup retrieves the external backup based on the id passed
func (s *serviceImpl) GetExternalBackup(ctx context.Context, request *v1.ResourceByID) (*storage.ExternalBackup, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "id must be specified when requesting an external backup")
	}
	backup, exists, err := s.dataStore.GetBackup(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "No external backup with id %q found", request.GetId())
	}
	secrets.ScrubSecretsFromStructWithReplacement(backup, secrets.ScrubReplacementStr)
	return backup, nil
}

// GetExternalBackups retrieves all external backups
func (s *serviceImpl) GetExternalBackups(ctx context.Context, _ *v1.Empty) (*v1.GetExternalBackupsResponse, error) {
	backups, err := s.dataStore.ListBackups(ctx)
	if err != nil {
		return nil, err
	}
	for _, b := range backups {
		secrets.ScrubSecretsFromStructWithReplacement(b, secrets.ScrubReplacementStr)
	}
	return &v1.GetExternalBackupsResponse{
		ExternalBackups: backups,
	}, nil
}

func validateBackup(backup *storage.ExternalBackup) error {
	errorList := errorhelpers.NewErrorList("external backup validation")

	err := endpoints.ValidateEndpoints(backup.Config)
	if err != nil {
		errorList.AddWrap(err, "invalid endpoint")
	}
	if backup.GetName() == "" {
		errorList.AddString("name field must be specified")
	}
	if backup.GetBackupsToKeep() < 1 {
		errorList.AddString("backups to keep must be >=1")
	}
	if _, err := schedule.ConvertToCronTab(backup.GetSchedule()); err != nil {
		errorList.AddError(err)
	}
	return errorList.ToError()
}

func (s *serviceImpl) testBackup(ctx context.Context, backup *storage.ExternalBackup) error {
	return s.manager.Test(ctx, backup)
}

// TestExternalBackup tests that the current config is valid, without stored credential reconciliation
func (s *serviceImpl) TestExternalBackup(ctx context.Context, externalBackup *storage.ExternalBackup) (*v1.Empty, error) {
	return s.TestUpdatedExternalBackup(ctx, &v1.UpdateExternalBackupRequest{ExternalBackup: externalBackup, UpdatePassword: true})
}

// TestUpdatedExternalBackup tests that the provided config is valid
func (s *serviceImpl) TestUpdatedExternalBackup(ctx context.Context, request *v1.UpdateExternalBackupRequest) (*v1.Empty, error) {
	if err := validateBackup(request.GetExternalBackup()); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := s.reconcileUpdateExternalBackupRequest(ctx, request); err != nil {
		return nil, err
	}
	if err := s.testBackup(ctx, request.GetExternalBackup()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) TriggerExternalBackup(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "id must be specified when triggering a backup")
	}
	if err := s.manager.Backup(ctx, request.GetId()); err != nil {
		log.Errorf("error trigger backup: %v", err)
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) upsertExternalBackup(ctx context.Context, request *storage.ExternalBackup) error {
	if err := s.manager.Upsert(ctx, request); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := s.dataStore.UpsertBackup(ctx, request); err != nil {
		s.manager.Remove(ctx, request.GetId())
		return err
	}
	return nil
}

// PutExternalBackup inserts a new external backup into the system, without stored credential reconciliation
func (s *serviceImpl) PutExternalBackup(ctx context.Context, externalBackup *storage.ExternalBackup) (*storage.ExternalBackup, error) {
	return s.UpdateExternalBackup(ctx, &v1.UpdateExternalBackupRequest{ExternalBackup: externalBackup, UpdatePassword: true})
}

// UpdateExternalBackup inserts a new external backup into the system
func (s *serviceImpl) UpdateExternalBackup(ctx context.Context, request *v1.UpdateExternalBackupRequest) (*storage.ExternalBackup, error) {
	if request.GetExternalBackup().GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "id field must be provided when updating an external backup")
	}
	if err := s.reconcileUpdateExternalBackupRequest(ctx, request); err != nil {
		return nil, err
	}
	if err := validateBackup(request.GetExternalBackup()); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	if err := s.upsertExternalBackup(ctx, request.GetExternalBackup()); err != nil {
		return nil, err
	}
	return request.GetExternalBackup(), nil
}

// PostExternalBackup adds a new external backup to the system
func (s *serviceImpl) PostExternalBackup(ctx context.Context, request *storage.ExternalBackup) (*storage.ExternalBackup, error) {
	if request.GetId() != "" {
		return nil, errors.Wrap(errox.InvalidArgs, "id field must be empty when posting a new external backup")
	}
	if err := validateBackup(request); err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}
	request.Id = uuid.NewV4().String()
	if err := s.upsertExternalBackup(ctx, request); err != nil {
		return nil, err
	}

	if err := s.reporter.Register(request.Id, request.Name, storage.IntegrationHealth_BACKUP); err != nil {
		return nil, err
	}

	return request, nil
}

// DeleteExternalBackup deletes an external backup from the system
func (s *serviceImpl) DeleteExternalBackup(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Backup id is required for deletions")
	}
	if err := s.dataStore.RemoveBackup(ctx, request.GetId()); err != nil {
		return nil, err
	}
	if err := s.reporter.RemoveIntegrationHealth(request.GetId()); err != nil {
		return nil, err
	}

	s.manager.Remove(ctx, request.GetId())

	return &v1.Empty{}, nil
}

func (s *serviceImpl) reconcileUpdateExternalBackupRequest(ctx context.Context, updateRequest *v1.UpdateExternalBackupRequest) error {
	if updateRequest.GetUpdatePassword() {
		return nil
	}
	if updateRequest.GetExternalBackup() == nil {
		return errors.Wrap(errox.InvalidArgs, "request is missing external backup config")
	}
	if updateRequest.GetExternalBackup().GetId() == "" {
		return errors.Wrap(errox.InvalidArgs, "id required for stored credential reconciliation")
	}
	existingBackupConfig, exists, err := s.dataStore.GetBackup(ctx, updateRequest.GetExternalBackup().GetId())
	if err != nil {
		return err
	}
	if !exists {
		return errors.Wrapf(errox.NotFound, "backup config %s not found", updateRequest.GetExternalBackup().GetId())
	}
	if err := reconcileExternalBackupWithExisting(updateRequest.GetExternalBackup(), existingBackupConfig); err != nil {
		return errors.Wrap(errox.InvalidArgs, err.Error())
	}
	return nil
}

func reconcileExternalBackupWithExisting(update *storage.ExternalBackup, existing *storage.ExternalBackup) error {
	return secrets.ReconcileScrubbedStructWithExisting(update, existing)
}
