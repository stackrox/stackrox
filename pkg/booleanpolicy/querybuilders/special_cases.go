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
// so we want to find deployments that don't drop these capabilities. Deployments that DROP ALL capabilities
// implicitly drop any capabilities that are specified as values in the policy group.
func ForDropCaps() QueryBuilder {
	return queryBuilderFunc(func(group *storage.PolicyGroup) []*query.FieldQuery {
		// Do the group values already contain "ALL" as a value"?
		containsAll := false
		for _, v := range group.Values {
			if v.Value == "ALL" {
				containsAll = true
			}
		}
		var queries []*query.FieldQuery
		// If values do not contain ALL already, add it, for the implicit case.
		// If a deployment drops ALL, it drops capabilities that are specified in the values and hence
		// that deployment must not generate a violation
		if !containsAll {
			queries = append(queries, &query.FieldQuery{
				Field:  search.DropCapabilities.String(),
				Values: []string{"ALL"},
				Negate: true,
			})
		}
		queries = append(queries, &query.FieldQuery{
			Field:    search.DropCapabilities.String(),
			Negate:   true,
			Values:   mapValues(group, valueToStringExact),
			Operator: operatorProtoMap[group.GetBooleanOperator()],
		})

		return queries
	})
}

// ForCVE returns a query builder for CVEs.
func ForCVE() QueryBuilder {
	return wrapForVulnMgmt(func(group *storage.PolicyGroup) []*query.FieldQuery {
		return []*query.FieldQuery{
			fieldQueryFromGroup(group, search.CVE, valueToStringRegex),
		}
	})
}

// ForCVSS returns a query builder for CVSS scores.
func ForCVSS() QueryBuilder {
	return wrapForVulnMgmt(func(group *storage.PolicyGroup) []*query.FieldQuery {
		return []*query.FieldQuery{
			fieldQueryFromGroup(group, search.CVSS, nil),
		}
	})
}

// ForSeverity returns a query builder for Severity ratings.
func ForSeverity() QueryBuilder {
	return wrapForVulnMgmt(func(group *storage.PolicyGroup) []*query.FieldQuery {
		return []*query.FieldQuery{
			fieldQueryFromGroup(group, search.Severity, func(value string) string {
				// The full enum is `<SEVERITY>_VULNERABILITY_SEVERITY`
				// For UX purposes when people write their own JSON policies,
				// we do not require people to write the entire enum,
				// but instead just the `<SEVERITY> part.`
				return value + "_VULNERABILITY_SEVERITY"
			}),
		}
	})
}

func wrapForVulnMgmt(f queryBuilderFunc) QueryBuilder {
	return queryBuilderFunc(func(group *storage.PolicyGroup) []*query.FieldQuery {
		return append(f(group),
			&query.FieldQuery{
				Field:  search.CVESuppressed.String(),
				Values: []string{"false"},
			},
			&query.FieldQuery{
				Field:  search.VulnerabilityState.String(),
				Values: []string{storage.VulnerabilityState_OBSERVED.String()},
			})
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

// ForFixedBy returns a query builder specific to the FixedBy field. It's a regular regex field,
// except that for historic reasons, .* is special-cased and translated to .+.
func ForFixedBy() QueryBuilder {
	return wrapForVulnMgmt(func(group *storage.PolicyGroup) []*query.FieldQuery {
		return []*query.FieldQuery{
			fieldQueryFromGroup(group, search.FixedBy, mapFixedByValue),
		}
	})
}

func mapFixedByValue(s string) string {
	if s == ".*" {
		s = ".+"
	}
	return valueToStringRegex(s)
}

// ForImageSignatureVerificationStatus returns a query builder for Image
// Signature Verification Status.
func ForImageSignatureVerificationStatus() QueryBuilder {
	qbf := func(group *storage.PolicyGroup) []*query.FieldQuery {
		return []*query.FieldQuery{{
			Field:    search.ImageSignatureVerifiedBy.String(),
			Values:   mapValues(group, nil),
			Operator: operatorProtoMap[group.GetBooleanOperator()],
			Negate:   !group.Negate,
		}}
	}
	return queryBuilderFunc(qbf)
}
