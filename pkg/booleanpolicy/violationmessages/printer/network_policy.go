package printer

import (
	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/augmentedobjs"
)

const (
	missingIngressNetworkPolicy = `The deployment is missing Ingress Network Policy.`
	missingEgressNetworkPolicy  = `The deployment is missing Egress Network Policy.`
)

const (
	// PolicyID is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote a policy ID.
	PolicyID = "Policy ID"
	// PolicyName is used as key in storage.Alert_Violation_KeyValueAttrs_KeyValueAttr to denote a policy name.
	PolicyName = "Policy name"
)

// TODO(ROX-9760): Implement these functions according to UX decision on how to display violations for missing network policies.
// This is implemented with place-holder messages for now just to unblock further developments on the evaluation of
// this policy.

func missingIngressNetworkPolicyPrinter(fieldMap map[string][]string) ([]string, error) {
	return executeTemplate(missingIngressNetworkPolicy, nil)
}

func missingEgressNetworkPolicyPrinter(fieldMap map[string][]string) ([]string, error) {
	return executeTemplate(missingEgressNetworkPolicy, nil)
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
