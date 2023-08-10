package handler

import (
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/grpc/routes"
	"github.com/stackrox/rox/pkg/sac/resources"
)

// CustomRoutes returns custom routes registered for reports
func CustomRoutes() []routes.CustomRoute {
	return []routes.CustomRoute{
		{
			Route:         "/api/reports/jobs/download",
			Authorizer:    user.With(permissions.Modify(resources.WorkflowAdministration)),
			ServerHandler: newDownloadHandler(),
			Compression:   true,
		},
	}
}
