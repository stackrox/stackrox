package internal

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	Authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.Integration)): {
			v1.ExternalBackupService_GetExternalBackup_FullMethodName,
			v1.ExternalBackupService_GetExternalBackups_FullMethodName,
		},
		user.With(permissions.Modify(resources.Integration)): {
			v1.ExternalBackupService_PutExternalBackup_FullMethodName,
			v1.ExternalBackupService_PostExternalBackup_FullMethodName,
			v1.ExternalBackupService_TestExternalBackup_FullMethodName,
			v1.ExternalBackupService_DeleteExternalBackup_FullMethodName,
			v1.ExternalBackupService_TriggerExternalBackup_FullMethodName,
			v1.ExternalBackupService_UpdateExternalBackup_FullMethodName,
			v1.ExternalBackupService_TestUpdatedExternalBackup_FullMethodName,
		},
	})
)
