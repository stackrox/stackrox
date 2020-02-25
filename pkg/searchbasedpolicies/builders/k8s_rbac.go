package builders

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/searchbasedpolicies"
)

var (
	// Ensure rbacPermissionLabels keys with PermissionLevel in 'policy.proto'.
	rbacPermissionLabels = map[storage.PermissionLevel]string{
		storage.PermissionLevel_DEFAULT:               "default permissions",
		storage.PermissionLevel_ELEVATED_IN_NAMESPACE: "elevated permissions in namespace",
		storage.PermissionLevel_ELEVATED_CLUSTER_WIDE: "elevated cluster wide permissions",
		storage.PermissionLevel_CLUSTER_ADMIN:         "cluster admin permissions",
	}
)

// K8sRBACQueryBuilder builds queries for K8s RBAC permission level.
type K8sRBACQueryBuilder struct{}

// Name implements the PolicyQueryBuilder interface.
func (p K8sRBACQueryBuilder) Name() string {
	return "query builder for k8s rbac permissions"
}

// Query takes in the fields of a policy and produces a query that will find indexed violators of the policy.s
func (p K8sRBACQueryBuilder) Query(fields *storage.PolicyFields, _ map[search.FieldLabel]*v1.SearchField) (*v1.Query, searchbasedpolicies.ViolationPrinter, error) {
	// Check that a permission level is set in the policy.
	if fields.GetPermissionPolicy().GetPermissionLevel() == storage.PermissionLevel_UNSET ||
		fields.GetPermissionPolicy().GetPermissionLevel() == storage.PermissionLevel_NONE {
		return nil, nil, nil
	}
	disallowedPermission := fields.GetPermissionPolicy().GetPermissionLevel()
	q := search.NewQueryBuilder().
		AddStrings(search.ServiceAccountPermissionLevel, fmt.Sprintf(">=%s", disallowedPermission.String())).ProtoQuery()

	v := func(ctx context.Context, result search.Result) searchbasedpolicies.Violations {
		violations := searchbasedpolicies.Violations{
			AlertViolations: []*storage.Alert_Violation{
				{
					Message: fmt.Sprintf("Deployment uses a service account with at least %s", strings.ToLower(rbacPermissionLabels[disallowedPermission])),
				},
			},
		}
		return violations
	}
	return q, v, nil
}
