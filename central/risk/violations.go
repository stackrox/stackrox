package risk

import (
	"fmt"
	"sort"
	"strings"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
)

const (
	policyViolationsHeuristic = "Policy Violations Heuristic"
	policySaturation          = 20
)

// ViolationsMultiplier is a scorer for the violations on a deployment
type ViolationsMultiplier struct {
	getter AlertGetter
}

// An AlertGetter provides the required access to alerts for risk scoring.
type AlertGetter interface {
	GetAlerts(request *v1.GetAlertsRequest) ([]*v1.Alert, error)
}

// NewViolationsMultiplier scores the data based on the number and severity of policy violations.
func NewViolationsMultiplier(getter AlertGetter) *ViolationsMultiplier {
	return &ViolationsMultiplier{
		getter: getter,
	}
}

func severityImpact(severity v1.Severity) float32 {
	switch severity {
	case v1.Severity_LOW_SEVERITY:
		return 1
	case v1.Severity_MEDIUM_SEVERITY:
		return 2
	case v1.Severity_HIGH_SEVERITY:
		return 3
	case v1.Severity_CRITICAL_SEVERITY:
		return 4
	default:
		return 0
	}
}

// Score takes a deployment and evaluates its risk based on policy violations.
func (v *ViolationsMultiplier) Score(deployment *v1.Deployment) *v1.Risk_Result {
	var severitySum float32
	var count int
	var policies []*v1.Policy
	alerts, err := v.getter.GetAlerts(&v1.GetAlertsRequest{
		DeploymentId: deployment.GetId(),
		Stale:        []bool{false},
	})
	if err != nil {
		logger.Errorf("Couldn't get risk violations for %s: %s", deployment.GetId(), err)
		return nil
	}
	for _, alert := range alerts {
		count++
		severitySum += severityImpact(alert.GetPolicy().GetSeverity())
		policies = append(policies, alert.GetPolicy())
	}
	// This does not contribute to the overall risk of the container
	if severitySum == 0 {
		return nil
	} else if severitySum > policySaturation {
		severitySum = policySaturation
	}
	score := (severitySum / policySaturation) + 1
	return &v1.Risk_Result{
		Name:    policyViolationsHeuristic,
		Factors: policyFactors(policies),
		Score:   score,
	}
}

func severityString(s v1.Severity) string {
	trim := strings.TrimSuffix(s.String(), "_SEVERITY")
	return strings.ToUpper(trim[:1]) + strings.ToLower(trim[1:])
}

func policyFactors(policies []*v1.Policy) (factors []string) {
	for _, p := range policies {
		factors = append(factors, fmt.Sprintf("Deployment violates policy %s (severity: %s)", p.GetName(), severityString(p.GetSeverity())))
	}
	sort.Strings(factors)
	return
}
