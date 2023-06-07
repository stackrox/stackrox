package common

import (
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
)

var (
	// Authorizer is used for authorizing report configuration grpc service calls
	Authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.WorkflowAdministration)): {
			"/v1.ReportConfigurationService/GetReportConfigurations",
			"/v1.ReportConfigurationService/GetReportConfiguration",
			"/v1.ReportConfigurationService/CountReportConfigurations",
		},
		or.Or(
			user.With(permissions.Modify(resources.WorkflowAdministration), permissions.View(resources.Integration), permissions.View(resources.Access)),
			user.With(permissions.Modify(resources.WorkflowAdministration), permissions.View(resources.Integration))): {
			"/v1.ReportConfigurationService/PostReportConfiguration",
			"/v1.ReportConfigurationService/UpdateReportConfiguration",
		},
		user.With(permissions.Modify(resources.WorkflowAdministration)): {
			"/v1.ReportConfigurationService/DeleteReportConfiguration",
		},
		user.With(permissions.Modify(resources.WorkflowAdministration), permissions.View(resources.Integration)): {
			"/v2.ReportConfigurationService/PostReportConfiguration",
		},
	})
)
