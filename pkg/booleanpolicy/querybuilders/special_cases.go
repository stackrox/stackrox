package querybuilders

import (
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate/basematchers"
	"github.com/stackrox/rox/pkg/utils"
)

// ForK8sRBAC returns a specific query builder for K8s RBAC.
// Note that for K8s RBAC, the semantics are that
// the user specifies a value, and the policy matches if the actual permission
// is greater than or equal to that value.
func ForK8sRBAC() QueryBuilder {
	return queryBuilderFunc(func(group *storage.PolicyGroup) []*query.FieldQuery {
		return []*query.FieldQuery{fieldQueryFromGroup(group, search.ServiceAccountPermissionLevel, func(s string) string {
			return fmt.Sprintf("%s%s", basematchers.GreaterThanOrEqualTo, s)
		})}
	})
}

// ForDropCaps returns a specific query builder for drop capabilities.
// Note that here, we always negate -- the user specifies a list of capabilities that _must_ be dropped,
// so we want to find deployments that don't drop these capabilities.
func ForDropCaps() QueryBuilder {
	return queryBuilderFunc(func(group *storage.PolicyGroup) []*query.FieldQuery {
		return []*query.FieldQuery{{
			Field:    search.DropCapabilities.String(),
			Negate:   true,
			Values:   mapValues(group, valueToStringExact),
			Operator: operatorProtoMap[group.GetBooleanOperator()],
		}}
	})
}

// ForCVE returns a query builder for CVEs.
func ForCVE() QueryBuilder {
	return queryBuilderFunc(func(group *storage.PolicyGroup) []*query.FieldQuery {
		return []*query.FieldQuery{
			fieldQueryFromGroup(group, search.CVE, valueToStringRegex),
			{
				Field:  search.CVESuppressed.String(),
				Values: []string{"false"},
			},
		}
	})
}

// ForCVSS returns a query builder for CVSS scores.
func ForCVSS() QueryBuilder {
	return queryBuilderFunc(func(group *storage.PolicyGroup) []*query.FieldQuery {
		return []*query.FieldQuery{
			fieldQueryFromGroup(group, search.CVSS, nil),
			{
				Field:  search.CVESuppressed.String(),
				Values: []string{"false"},
			},
		}
	})
}

// ForWriteableHostMount returns a query builder for writeable host mounts.
func ForWriteableHostMount() QueryBuilder {
	return queryBuilderFunc(func(group *storage.PolicyGroup) []*query.FieldQuery {
		values := mapValues(group, nil)
		// Should never happen, will be enforced by validation.
		if len(values) != 1 {
			utils.Should(errors.Errorf("received unexpected number of values for host mount field: %v", values))
			return nil
		}
		asBool, err := strconv.ParseBool(values[0])
		// Should never happen, will be enforced by validation.
		if err != nil {
			utils.Should(errors.Wrap(err, "invalid value for host mount field path"))
			return nil
		}
		return []*query.FieldQuery{
			{
				Field: search.VolumeReadonly.String(),
				// The policy specifies whether it's _writable_, while we store
				// whether the field is read-only, so we need to invert.
				Values: []string{strconv.FormatBool(!asBool)},
			},
			{
				Field:  search.VolumeType.String(),
				Values: []string{"HostPath"},
			},
		}
	})
}
