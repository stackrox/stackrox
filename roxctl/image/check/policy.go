package check

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stackrox/rox/generated/storage"
)

//go:generate genny -in=../../../pkg/set/generic.go -out=policy-set.go -pkg check gen "KeyType=*storage.Policy"

// totalPolicyAmountKey relates to the key within the policy summary map which yields the total amount of violated
// policies
const totalPolicyAmountKey = "TOTAL"

// policySeverity is used for easier comparing the prettified string version of storage.Severity
// when sorting policies by severity.
type policySeverity int

const (
	lowSeverity policySeverity = iota
	mediumSeverity
	highSeverity
	criticalSeverity
)

func (s policySeverity) String() string {
	return [...]string{"LOW", "MEDIUM", "HIGH", "CRITICAL"}[s]
}

func policySeverityFromString(s string) policySeverity {
	switch s {
	case lowSeverity.String():
		return lowSeverity
	case mediumSeverity.String():
		return mediumSeverity
	case highSeverity.String():
		return highSeverity
	case criticalSeverity.String():
		return criticalSeverity
	default:
		return 0
	}
}

// newPolicySummaryForPrinting creates a policyJSONResult that shall be used for printing and holds
// all relevant information regarding violated policies, failing policies and a summary of all violated policies
// by severity
// NOTE: The returned *policyJSONResult CAN be passed to json.Marshal
func newPolicySummaryForPrinting(alerts []*storage.Alert, failedPolicies []*storage.Policy) *policyJSONResult {
	policies := map[string]*policyJSON{}
	numOfViolationsBySeverity := createNumOfSeverityMap()

	for _, alert := range alerts {
		policy := alert.GetPolicy()
		_, exists := policies[policy.GetId()]
		// If the policy does not yet exist, initially create it
		if !exists {
			policies[policy.GetId()] = &policyJSON{
				Name:        policy.GetName(),
				Severity:    stripSeverityEnum(policy.GetSeverity()),
				Description: policy.GetDescription(),
				Remediation: policy.GetRemediation(),
			}
			numOfViolationsBySeverity[stripSeverityEnum(policy.GetSeverity())]++
			numOfViolationsBySeverity[totalPolicyAmountKey]++
		}
		// Multiple alerts could violate the same policy, need to ensure the violations are merged from all alerts
		policyJSONObj := policies[policy.GetId()]
		policyJSONObj.Violation += getAlertViolationsString(alert)
	}

	return &policyJSONResult{
		Result: policyJSONStructure{
			Summary:               numOfViolationsBySeverity,
			ViolatedPolicies:      sortPoliciesBySeverity(getPoliciesFromMap(policies)),
			BuildBreakingPolicies: sortBreakingPoliciesByName(getBreakingPolicies(failedPolicies)),
		},
	}
}

func createNumOfSeverityMap() map[string]int {
	numOfSeverityMap := make(map[string]int, 5)
	numOfSeverityMap[totalPolicyAmountKey] = 0
	numOfSeverityMap[stripSeverityEnum(storage.Severity_LOW_SEVERITY)] = 0
	numOfSeverityMap[stripSeverityEnum(storage.Severity_MEDIUM_SEVERITY)] = 0
	numOfSeverityMap[stripSeverityEnum(storage.Severity_HIGH_SEVERITY)] = 0
	numOfSeverityMap[stripSeverityEnum(storage.Severity_CRITICAL_SEVERITY)] = 0
	return numOfSeverityMap
}

func getBreakingPolicies(failedPolicies []*storage.Policy) []breakingPolicyJSON {
	breakingPolicies := make([]breakingPolicyJSON, 0, len(failedPolicies))
	for _, policy := range failedPolicies {
		breakingPolicies = append(breakingPolicies, breakingPolicyJSON{
			Name:        policy.GetName(),
			Remediation: policy.GetRemediation(),
		})
	}
	return breakingPolicies
}

func sortBreakingPoliciesByName(breakingPolicies []breakingPolicyJSON) []breakingPolicyJSON {
	sort.SliceStable(breakingPolicies, func(i, j int) bool {
		return breakingPolicies[i].Name < breakingPolicies[j].Name
	})
	return breakingPolicies
}

// sortPoliciesBySeverity sorts policies by their policySeverity from highest (criticalSeverity) to lowest (lowSeverity)
func sortPoliciesBySeverity(policies []policyJSON) []policyJSON {
	// sort alphabetically by name first
	sort.SliceStable(policies, func(i, j int) bool {
		return policies[i].Name < policies[j].Name
	})
	// sort decreasing by severity, CRITICAL being highest - LOW being lowest
	sort.SliceStable(policies, func(i, j int) bool {
		return policySeverityFromString(policies[i].Severity) > policySeverityFromString(policies[j].Severity)
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

func stripSeverityEnum(severity storage.Severity) string {
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
