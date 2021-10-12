package check

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

type severity int

const (
	low severity = iota
	medium
	high
	critical
)

func severityFromString(s string) severity {
	switch s {
	case "LOW":
		return low
	case "MEDIUM":
		return medium
	case "HIGH":
		return high
	case "CRITICAL":
		return critical
	default:
		return 0
	}
}

// newAlertSummaryForPrinting creates a JSON object containing all summaries and information of alerts that can be used
// for printing. The returned interface CAN be passed to json.Marshal
func newAlertSummaryForPrinting(alerts []*storage.Alert, failedPolicies []*storage.Policy, numFailuresBySeverity map[string]int) interface{} {
	policies := map[string]*policyJSON{}

	for _, alert := range alerts {
		policy := alert.GetPolicy()
		_, exists := policies[policy.GetId()]
		// If the policy does not yet exist, initially create it
		if !exists {
			policies[policy.GetId()] = &policyJSON{
				Name:        policy.GetName(),
				Severity:    prettifySeverityEnum(policy.GetSeverity()),
				Description: policy.GetDescription(),
				Remediation: policy.GetRemediation(),
			}
		}
		// Multiple alerts could violate the same policy, need to ensure the violations are merged from all alerts
		policyJSONObj := policies[policy.GetId()]
		policyJSONObj.Violation += getAlertViolationsString(alert)
	}

	breakingPolicies := make([]breakingPolicyJSON, 0, len(failedPolicies))
	for _, policy := range failedPolicies {
		breakingPolicies = append(breakingPolicies, breakingPolicyJSON{
			Name:        policy.GetName(),
			Remediation: policy.GetRemediation(),
		})
	}

	return &policyJSONResult{
		Result: policyJSONStructure{
			Summary:               numFailuresBySeverity,
			ViolatedPolicies:      sortPoliciesBySeverity(getPoliciesFromMap(policies)),
			BuildBreakingPolicies: breakingPolicies,
		},
	}
}

func sortPoliciesBySeverity(policies []policyJSON) []policyJSON {
	sort.SliceStable(policies, func(i, j int) bool {
		return severityFromString(policies[i].Severity) > severityFromString(policies[j].Severity)
	})
	return policies
}

func getPoliciesFromMap(policyMap map[string]*policyJSON) []policyJSON {
	policies := make([]policyJSON, 0, len(policyMap))
	for _, policy := range policyMap {
		policies = append(policies, *policy)
	}
	return policies
}

func getAlertViolationsString(alert *storage.Alert) string {
	var res string
	for _, violation := range alert.GetViolations() {
		res += fmt.Sprintf("- %s\n", violation.Message)
	}
	return res
}

func prettifySeverityEnum(severity storage.Severity) string {
	return strings.ReplaceAll(severity.String(), "_SEVERITY", "")
}

type policyJSONResult struct {
	Result policyJSONStructure `json:"result"`
}

type policyJSONStructure struct {
	Summary               map[string]int       `json:"summary"`
	ViolatedPolicies      []policyJSON         `json:"violatedPolicies,omitempty"`
	BuildBreakingPolicies []breakingPolicyJSON `json:"buildBreakingPolicies,omitempty"`
}

type policyJSON struct {
	Name        string `json:"name"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Violation   string `json:"violation"`
	Remediation string `json:"remediation"`
}

type breakingPolicyJSON struct {
	Name        string `json:"name"`
	Remediation string `json:"remediation"`
}
