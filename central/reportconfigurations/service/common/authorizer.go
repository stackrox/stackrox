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
		// TODO: ROX-13888 Replace VulnerabilityReports with WorkflowAdministration.
		or.Or(
			user.With(permissions.View(resources.VulnerabilityReports)),
			user.With(permissions.View(resources.WorkflowAdministration))): {
			"/v1.ReportConfigurationService/GetReportConfigurations",
			"/v1.ReportConfigurationService/GetReportConfiguration",
			"/v1.ReportConfigurationService/CountReportConfigurations",
		},
		// TODO: ROX-13888 Replace VulnerabilityReports with WorkflowAdministration.
		// TODO: ROX-14398 Replace Role with Access
		or.Or(
			user.With(permissions.Modify(resources.VulnerabilityReports), permissions.View(resources.Integration), permissions.View(resources.Role)),
			user.With(permissions.Modify(resources.WorkflowAdministration), permissions.View(resources.Integration))): {
			"/v1.ReportConfigurationService/PostReportConfiguration",
			"/v1.ReportConfigurationService/UpdateReportConfiguration",
		},
		// TODO: ROX-13888 Replace VulnerabilityReports with WorkflowAdministration.
		or.Or(
			user.With(permissions.Modify(resources.VulnerabilityReports)),
			user.With(permissions.Modify(resources.WorkflowAdministration))): {
			"/v1.ReportConfigurationService/DeleteReportConfiguration",
		},
		or.Or(
			user.With(permissions.Modify(resources.VulnerabilityReports), permissions.View(resources.Integration), permissions.View(resources.Role)),
			user.With(permissions.Modify(resources.WorkflowAdministration), permissions.View(resources.Integration))): {
			"/v2.ReportConfigurationService/PostReportConfiguration",
		},
	})
)
