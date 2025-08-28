package externaldb

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/stackrox/rox/central/externalbackups/service/internal"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"google.golang.org/grpc"
)

const cause = "the scheduled backup service is not applicable when using an external database. " +
	"Please manage backups directly with your database provider."

type serviceImpl struct {
	v1.UnimplementedExternalBackupServiceServer
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterExternalBackupServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterExternalBackupServiceHandler(ctx, mux, conn)
}

func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, internal.Authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) GetExternalBackup(_ context.Context, _ *v1.ResourceByID) (*storage.ExternalBackup, error) {
	return nil, errox.NotFound.CausedBy(cause)
}

func (s *serviceImpl) GetExternalBackups(_ context.Context, _ *v1.Empty) (*v1.GetExternalBackupsResponse, error) {
	return nil, errox.NotFound.CausedBy(cause)
}

func (s *serviceImpl) TestExternalBackup(_ context.Context, _ *storage.ExternalBackup) (*v1.Empty, error) {
	return nil, errox.NotFound.CausedBy(cause)
}

func (s *serviceImpl) TestUpdatedExternalBackup(_ context.Context, _ *v1.UpdateExternalBackupRequest) (*v1.Empty, error) {
	return nil, errox.NotFound.CausedBy(cause)
}

func (s *serviceImpl) TriggerExternalBackup(_ context.Context, _ *v1.ResourceByID) (*v1.Empty, error) {
	return nil, errox.NotFound.CausedBy(cause)
}

func (s *serviceImpl) PutExternalBackup(_ context.Context, _ *storage.ExternalBackup) (*storage.ExternalBackup, error) {
	return nil, errox.NotImplemented.CausedBy(cause)
}

func (s *serviceImpl) UpdateExternalBackup(_ context.Context, _ *v1.UpdateExternalBackupRequest) (*storage.ExternalBackup, error) {
	return nil, errox.NotFound.CausedBy(cause)
}

func (s *serviceImpl) PostExternalBackup(_ context.Context, _ *storage.ExternalBackup) (*storage.ExternalBackup, error) {
	return nil, errox.NotImplemented.CausedBy(cause)
}

func (s *serviceImpl) DeleteExternalBackup(_ context.Context, _ *v1.ResourceByID) (*v1.Empty, error) {
	return nil, errox.NotFound.CausedBy(cause)
}
