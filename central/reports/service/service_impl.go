package service

import (
	"context"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reports/common"
	reportConfigDS "github.com/stackrox/rox/central/reports/config/datastore"
	"github.com/stackrox/rox/central/reports/manager"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac/resources"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.WorkflowAdministration), permissions.View(resources.Integration),
			permissions.View(resources.Image)): {
			"/v1.ReportService/RunReport",
		},
	})
)

var (
	log = logging.LoggerForModule()
)

type serviceImpl struct {
	v1.UnimplementedReportServiceServer

	manager           manager.Manager
	reportConfigStore reportConfigDS.DataStore
	notifierStore     notifierDataStore.DataStore
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterReportServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterReportServiceHandler(ctx, mux, conn)
}

func (*serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) RunReport(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	rc, found, err := s.reportConfigStore.GetReportConfiguration(ctx, id.GetId())
	if err != nil {
		return &v1.Empty{}, errors.Wrapf(err, "error finding report configuration %s", id)
	}
	if !found {
		return &v1.Empty{}, errors.Wrapf(errox.NotFound, "unable to find report configuration %s", id)
	}
	if !common.IsV1ReportConfig(rc) {
		return &v1.Empty{}, errors.Wrap(errox.InvalidArgs, "report configuration does not belong to reporting version 1.0")
	}

	if err := s.manager.RunReport(ctx, rc); err != nil {
		return &v1.Empty{}, err
	}
	return &v1.Empty{}, nil
}
