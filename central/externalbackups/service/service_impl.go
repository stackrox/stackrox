package service

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/externalbackups/manager"
	backupStore "github.com/stackrox/rox/central/externalbackups/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/secrets"
	"github.com/stackrox/rox/pkg/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.BackupPlugins)): {
			"/v1.ExternalBackupService/GetExternalBackup",
			"/v1.ExternalBackupService/GetExternalBackups",
		},
		user.With(permissions.Modify(resources.BackupPlugins)): {
			"/v1.ExternalBackupService/PutExternalBackup",
			"/v1.ExternalBackupService/PostExternalBackup",
			"/v1.ExternalBackupService/TestExternalBackup",
			"/v1.ExternalBackupService/DeleteExternalBackup",
			"/v1.ExternalBackupService/TriggerExternalBackup",
		},
	})
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct {
	manager manager.Manager
	store   backupStore.Store
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
		return nil, status.Error(codes.InvalidArgument, "Id must be specified when requesting an external backup")
	}
	backup, err := s.store.GetBackup(request.GetId())
	if err != nil {
		return nil, err
	}
	if backup == nil {
		return nil, status.Errorf(codes.NotFound, "No external backup with id %q found", request.GetId())
	}
	secrets.ScrubSecretsFromStruct(backup)
	return backup, nil
}

// GetExternalBackups retrieves all external backups
func (s *serviceImpl) GetExternalBackups(context.Context, *v1.Empty) (*v1.GetExternalBackupsResponse, error) {
	backups, err := s.store.ListBackups()
	if err != nil {
		return nil, err
	}
	for _, b := range backups {
		secrets.ScrubSecretsFromStruct(b)
	}
	return &v1.GetExternalBackupsResponse{
		ExternalBackups: backups,
	}, nil
}

func validateBackup(backup *storage.ExternalBackup) error {
	errorList := errorhelpers.NewErrorList("external backup validation")

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

func (s *serviceImpl) testBackup(backup *storage.ExternalBackup) error {
	return s.manager.Test(backup)
}

// TestExternalBackup tests that the current config is valid
func (s *serviceImpl) TestExternalBackup(ctx context.Context, request *storage.ExternalBackup) (*v1.Empty, error) {
	if err := validateBackup(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.testBackup(request); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) TriggerExternalBackup(_ context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Errorf(codes.InvalidArgument, "id must be specified when triggering a backup")
	}
	if err := s.manager.Backup(request.GetId()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) upsertExternalBackup(request *storage.ExternalBackup) error {
	if err := s.manager.Upsert(request); err != nil {
		return status.Errorf(codes.InvalidArgument, err.Error())
	}

	if err := s.store.UpsertBackup(request); err != nil {
		s.manager.Remove(request.GetId())
		return err
	}
	return nil
}

// PutExternalBackup inserts a new external backup into the system
func (s *serviceImpl) PutExternalBackup(ctx context.Context, request *storage.ExternalBackup) (*storage.ExternalBackup, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Id field must be provided when updating an external backup")
	}
	if err := validateBackup(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.upsertExternalBackup(request); err != nil {
		return nil, err
	}
	return request, nil
}

// PostExternalBackup adds a new external backup to the system
func (s *serviceImpl) PostExternalBackup(ctx context.Context, request *storage.ExternalBackup) (*storage.ExternalBackup, error) {
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field must be empty when posting a new external backup")
	}
	if err := validateBackup(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	request.Id = uuid.NewV4().String()
	if err := s.upsertExternalBackup(request); err != nil {
		return nil, err
	}
	return request, nil
}

// DeleteExternalBackup deletes an external backup from the system
func (s *serviceImpl) DeleteExternalBackup(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Backup id is required for deletions")
	}
	s.manager.Remove(request.GetId())
	if err := s.store.RemoveBackup(request.GetId()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}
