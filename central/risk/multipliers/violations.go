package multipliers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

const (
	// PolicyViolationsHeading is the risk result name for scores calculated by this multiplier.
	PolicyViolationsHeading = "Policy Violations"

	policySaturation = 20
)

// ViolationsMultiplier is a scorer for the violations on a deployment
type ViolationsMultiplier struct {
	getter getters.AlertGetter
}

type policyFactor struct {
	name     string
	severity v1.Severity
}

// NewViolations scores the data based on the number and severity of policy violations.
func NewViolations(getter getters.AlertGetter) *ViolationsMultiplier {
	return &ViolationsMultiplier{
		getter: getter,
	}
}

// Score takes a deployment and evaluates its risk based on policy violations.
func (v *ViolationsMultiplier) Score(deployment *v1.Deployment) *v1.Risk_Result {
	qb := search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deployment.GetId()).AddStrings(search.ViolationState, v1.ViolationState_ACTIVE.String())

	alerts, err := v.getter.ListAlerts(&v1.ListAlertsRequest{
		Query: qb.Query(),
	})
	if err != nil {
		logger.Errorf("Couldn't get risk violations for %s: %s", deployment.GetId(), err)
		return nil
	}

	var severitySum float32
	var count int
	var factors []policyFactor
	for _, alert := range alerts {
		count++
		severitySum += severityImpact(alert.GetPolicy().GetSeverity())
		factors = append(factors, policyFactor{
			name:     alert.GetPolicy().GetName(),
			severity: alert.GetPolicy().GetSeverity(),
		})
	}

	// This does not contribute to the overall risk of the container
	if severitySum == 0 {
		return nil
	} else if severitySum > policySaturation {
		severitySum = policySaturation
	}
	score := (severitySum / policySaturation) + 1
	return &v1.Risk_Result{
		Name:    PolicyViolationsHeading,
		Factors: policyFactors(factors),
		Score:   score,
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

func severityString(s v1.Severity) string {
	trim := strings.TrimSuffix(s.String(), "_SEVERITY")
	return strings.ToUpper(trim[:1]) + strings.ToLower(trim[1:])
}

func policyFactors(pfs []policyFactor) (factors []*v1.Risk_Result_Factor) {
	sort.Slice(pfs, func(i, j int) bool {
		if pfs[i].severity == pfs[j].severity {
			// Break ties using the name.
			return pfs[i].name < pfs[j].name
		}
		// Otherwise use the impact score.
		return severityImpact(pfs[i].severity) > severityImpact(pfs[j].severity)
	})

	factors = make([]*v1.Risk_Result_Factor, 0, len(pfs))
	for _, pf := range pfs {
		factors = append(factors,
			&v1.Risk_Result_Factor{Message: fmt.Sprintf("%s (severity: %s)", pf.name, severityString(pf.severity))})
	}
	return
}
