package service

import (
	"context"
	"path/filepath"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	versionUtils "github.com/stackrox/rox/central/version/utils"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/migrations"
	"github.com/stackrox/rox/pkg/postgres/pgadmin"
	"github.com/stackrox/rox/pkg/postgres/pgconfig"
	"github.com/stackrox/rox/pkg/version"
	"google.golang.org/grpc"
)

const (
	minForceRollbackTo = "3.0.58.0"
)

var (
	authorizer = user.WithRole(accesscontrol.Admin)
)

type serviceImpl struct {
	v1.UnimplementedCentralHealthServiceServer
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterCentralHealthServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterCentralHealthServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetUpgradeStatus returns the upgrade status for Central.
func (s *serviceImpl) GetUpgradeStatus(_ context.Context, _ *v1.Empty) (*v1.GetUpgradeStatusResponse, error) {
	// Get Postgres config data
	_, adminConfig, err := pgconfig.GetPostgresConfig()
	if err != nil {
		return nil, err
	}

	upgradeStatus := &v1.CentralUpgradeStatus{
		Version: version.GetMainVersion(),
		// Due to backwards compatibility going forward we can assume
		// we can rollback after an upgrade
		CanRollbackAfterUpgrade: true,
	}

	if !pgconfig.IsExternalDatabase() {
		exists, err := pgadmin.CheckIfDBExists(adminConfig, migrations.PreviousDatabase)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to determine if %s database exists", migrations.PreviousDatabase)
		}
		if exists {
			// Get a short-lived connection for the purposes of checking the version of the previous clone.
			pool, err := pgadmin.GetClonePool(adminConfig, migrations.GetPreviousClone())
			if err != nil {
				return nil, errors.Wrap(err, "Failed to retrieve previous database version.")
			}
			defer pool.Close()

			// Get rollback to version
			migVer, err := versionUtils.ReadPreviousVersionPostgres(pool)
			if err != nil {
				log.Infof("Unable to get previous version, leaving ForceRollbackTo empty.  %v", err)
			}
			if err == nil && migVer.SeqNum > 0 && version.CompareVersionsOr(migVer.MainVersion, minForceRollbackTo, -1) >= 0 {
				upgradeStatus.ForceRollbackTo = migVer.MainVersion
			}
		} else {
			// It is possible that we had a Rocks previously, so we may be able to rollback to that version.
			// Get rollback to version
			migVer, err := migrations.Read(filepath.Join(migrations.DBMountPath(), migrations.PreviousClone))
			if err != nil {
				log.Infof("Unable to get previous version, leaving ForceRollbackTo empty.  %v", err)
			}
			if err == nil && migVer.SeqNum > 0 && version.CompareVersionsOr(migVer.MainVersion, minForceRollbackTo, -1) >= 0 {
				upgradeStatus.ForceRollbackTo = migVer.MainVersion
			}
		}
	}

	return &v1.GetUpgradeStatusResponse{
		UpgradeStatus: upgradeStatus,
	}, nil
}
