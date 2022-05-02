package service

import (
	"context"
	"net/mail"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/reportconfigurations/datastore"
	"github.com/stackrox/rox/central/reports/manager"
	accessScopeStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
	"google.golang.org/grpc"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.VulnerabilityReports)): {
			"/v1.ReportConfigurationService/GetReportConfigurations",
			"/v1.ReportConfigurationService/GetReportConfiguration",
			"/v1.ReportConfigurationService/CountReportConfigurations",
		},
		user.With(permissions.Modify(resources.VulnerabilityReports), permissions.View(resources.Notifier), permissions.View(resources.Role)): {
			"/v1.ReportConfigurationService/PostReportConfiguration",
			"/v1.ReportConfigurationService/UpdateReportConfiguration",
		},
		user.With(permissions.Modify(resources.VulnerabilityReports)): {
			"/v1.ReportConfigurationService/DeleteReportConfiguration",
		},
	})
)

var (
	log = logging.LoggerForModule()
)

type serviceImpl struct {
	manager           manager.Manager
	reportConfigStore datastore.DataStore
	notifierStore     notifierDataStore.DataStore
	accessScopeStore  accessScopeStore.DataStore
}

func (s *serviceImpl) GetReportConfigurations(ctx context.Context, query *v1.RawQuery) (*v1.GetReportConfigurationsResponse, error) {
	// Fill in Query.
	parsedQuery, err := search.ParseQuery(query.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	// Fill in pagination.
	paginated.FillPagination(parsedQuery, query.GetPagination(), 1000)

	reportConfigs, err := s.reportConfigStore.GetReportConfigurations(ctx, parsedQuery)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve report configurations")
	}
	return &v1.GetReportConfigurationsResponse{ReportConfigs: reportConfigs}, nil
}

func (s *serviceImpl) GetReportConfiguration(ctx context.Context, id *v1.ResourceByID) (*v1.GetReportConfigurationResponse, error) {
	reportConfig, exists, err := s.reportConfigStore.GetReportConfiguration(ctx, id.GetId())
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Wrapf(errox.NotFound, "report configuration with id '%s' does not exist", id)
	}
	return &v1.GetReportConfigurationResponse{
		ReportConfig: reportConfig,
	}, nil
}

func (s *serviceImpl) PostReportConfiguration(ctx context.Context, request *v1.PostReportConfigurationRequest) (*v1.PostReportConfigurationResponse, error) {
	if err := s.validateReportConfiguration(ctx, request.GetReportConfig()); err != nil {
		return nil, err
	}
	id, err := s.reportConfigStore.AddReportConfiguration(ctx, request.GetReportConfig())
	if err != nil {
		return nil, err
	}

	createdReportConfig, _, err := s.reportConfigStore.GetReportConfiguration(ctx, id)
	if err := s.manager.Upsert(ctx, createdReportConfig); err != nil {
		return nil, err
	}

	return &v1.PostReportConfigurationResponse{
		ReportConfig: createdReportConfig,
	}, err
}

func (s *serviceImpl) UpdateReportConfiguration(ctx context.Context, request *v1.UpdateReportConfigurationRequest) (*v1.Empty, error) {
	if err := s.validateReportConfiguration(ctx, request.GetReportConfig()); err != nil {
		return &v1.Empty{}, err
	}
	if err := s.manager.Upsert(ctx, request.GetReportConfig()); err != nil {
		return nil, err
	}

	err := s.reportConfigStore.UpdateReportConfiguration(ctx, request.GetReportConfig())
	if err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}

func (s *serviceImpl) DeleteReportConfiguration(ctx context.Context, id *v1.ResourceByID) (*v1.Empty, error) {
	if id.GetId() == "" {
		return nil, errors.Wrap(errox.InvalidArgs, "Report configuration id is required for deletion")
	}
	if err := s.reportConfigStore.RemoveReportConfiguration(ctx, id.GetId()); err != nil {
		return &v1.Empty{}, err
	}
	return &v1.Empty{}, s.manager.Remove(ctx, id.GetId())
}

func (s *serviceImpl) CountReportConfigurations(ctx context.Context, request *v1.RawQuery) (*v1.CountReportConfigurationsResponse, error) {
	parsedQuery, err := search.ParseQuery(request.GetQuery(), search.MatchAllIfEmpty())
	if err != nil {
		return nil, errors.Wrap(errox.InvalidArgs, err.Error())
	}

	numReportConfigs, err := s.reportConfigStore.Count(ctx, parsedQuery)
	if err != nil {
		return nil, err
	}
	return &v1.CountReportConfigurationsResponse{Count: int32(numReportConfigs)}, nil
}

func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterReportConfigurationServiceServer(grpcServer, s)
}

func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterReportConfigurationServiceHandler(ctx, mux, conn)
}

func (*serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

func (s *serviceImpl) validateReportConfiguration(ctx context.Context, config *storage.ReportConfiguration) error {
	if config.GetName() == "" {
		return errors.Wrap(errox.InvalidArgs, "Report configuration name empty")
	}

	if config.GetSchedule() == nil {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must have a schedule")
	}

	schedule := config.GetSchedule()

	switch schedule.GetIntervalType() {
	case storage.Schedule_UNSET:
	case storage.Schedule_DAILY:
		return errors.Wrap(errox.InvalidArgs, "Report configuration must have a valid schedule type")
	case storage.Schedule_WEEKLY:
		if schedule.GetDaysOfWeek() == nil {
			return errors.Wrap(errox.InvalidArgs, "Report configuration must specify days of week for the schedule")
		}
		for _, day := range schedule.GetDaysOfWeek().GetDays() {
			if day < 0 || day > 6 {
				return errors.Wrap(errox.InvalidArgs, "Invalid schedule: Days of the week can be Sunday (0) - Saturday(6)")
			}
		}
	case storage.Schedule_MONTHLY:
		if schedule.GetDaysOfMonth() == nil || schedule.GetDaysOfMonth().GetDays() == nil {
			return errors.Wrap(errox.InvalidArgs, "Report configuration must specify days of the month for the schedule")
		}
		for _, day := range schedule.GetDaysOfMonth().GetDays() {
			if day != 1 && day != 15 {
				return errors.Wrap(errox.InvalidArgs, "Reports can be sent out only 1st or 15th of the month")
			}
		}
	}
	if config.GetEmailConfig() == nil {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify an email notifier configuration")
	}
	if config.GetEmailConfig().GetNotifierId() == "" {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify a valid email notifier")
	}
	if len(config.GetEmailConfig().GetMailingLists()) == 0 {
		return errors.Wrap(errox.InvalidArgs, "Report configuration must specify one more recipients to send the report to")
	}

	for _, addr := range config.GetEmailConfig().GetMailingLists() {
		if _, err := mail.ParseAddress(addr); err != nil {
			return errors.Wrapf(errox.InvalidArgs, "Invalid mailing list address: %s", addr)
		}
	}

	_, found, err := s.accessScopeStore.GetAccessScope(ctx, config.GetScopeId())
	if !found || err != nil {
		return errors.Wrapf(errox.NotFound, "Access scope %s not found. Error: %s", config.GetScopeId(), err)
	}

	_, found, err = s.notifierStore.GetNotifier(ctx, config.GetEmailConfig().GetNotifierId())
	if err != nil {
		return errors.Wrapf(errox.NotFound, "Failed to fetch notifier %s with error %s", config.GetEmailConfig().GetNotifierId(), err)
	}
	if !found {
		return errors.Wrapf(errox.NotFound, "Notifier %s not found", config.GetEmailConfig().GetNotifierId())
	}

	return nil
}
