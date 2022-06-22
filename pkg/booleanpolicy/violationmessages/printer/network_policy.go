package printer

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
	"github.com/stackrox/rox/pkg/booleanpolicy/fieldnames"
)

const (
	hasIngressNetworkPolicy = `The deployment{{if .HasIngress}} has{{else}} is missing{{end}} Ingress Network Policy.`
	hasEgressNetworkPolicy  = `The deployment{{if .HasEgress}} has{{else}} is missing{{end}} Egress Network Policy.`
)

const (
	// PolicyID is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote a policy ID.
	PolicyID = "Policy ID"
	// PolicyName is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote a policy name.
	PolicyName = "Policy name"
)

func hasIngressNetworkPolicyPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		HasIngress bool
	}
	hasIngress, err := getSingleValueFromFieldMap(fieldnames.HasIngressNetworkPolicy, fieldMap)
	if err != nil {
		return []string{}, err
	}
	r := resultFields{
		HasIngress: false,
	}
	if hasIngress == "true" {
		r.HasIngress = true
	}
	return executeTemplate(hasIngressNetworkPolicy, r)
}

func hasEgressNetworkPolicyPrinter(fieldMap map[string][]string) ([]string, error) {
	type resultFields struct {
		HasEgress bool
	}
	hasEgress, err := getSingleValueFromFieldMap(fieldnames.HasEgressNetworkPolicy, fieldMap)
	if err != nil {
		return []string{}, err
	}
	r := resultFields{
		HasEgress: false,
	}
	if hasEgress == "true" {
		r.HasEgress = true
	}
	return executeTemplate(hasEgressNetworkPolicy, r)
}

// EnhanceNetworkPolicyViolations enriches each violation object with Alert_Violation_KeyValueAttrs containing policy-id and policy-name
func EnhanceNetworkPolicyViolations(violations []*storage.Alert_Violation, np *augmentedobjs.NetworkPoliciesApplied) []*storage.Alert_Violation {
	kvAttrs := make([]*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr, 0, len(np.Policies))
	for id, p := range np.Policies {
		attrs := []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
			{Key: PolicyID, Value: id},
			{Key: PolicyName, Value: p.GetName()},
		}
		kvAttrs = append(kvAttrs, attrs...)
	}
	for _, violation := range violations {
		violation.Time = types.TimestampNow()
		if len(kvAttrs) > 0 {
			violation.MessageAttributes = &storage.Alert_Violation_KeyValueAttrs_{
				KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
					Attrs: kvAttrs,
				},
			}
		}
	}
	return violations
}
