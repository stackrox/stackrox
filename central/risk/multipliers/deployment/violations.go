package deployment

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/search"
)

const (
	// PolicyViolationsHeading is the risk result name for scores calculated by this multiplier.
	PolicyViolationsHeading = "Policy Violations"

	policySaturation = 50
	policyMaxValue   = 100
)

var (
	log = logging.LoggerForModule()
)

// ViolationsMultiplier is a scorer for the violations on a deployment
type ViolationsMultiplier struct {
	searcher getters.AlertSearcher
}

type policyFactor struct {
	name     string
	severity storage.Severity
}

// NewViolations scores the data based on the number and severity of policy violations.
func NewViolations(searcher getters.AlertSearcher) *ViolationsMultiplier {
	return &ViolationsMultiplier{
		searcher: searcher,
	}
}

// Score takes a deployment and evaluates its risk based on policy violations.
func (v *ViolationsMultiplier) Score(ctx context.Context, deployment *storage.Deployment, _ map[string][]*storage.Risk_Result) *storage.Risk_Result {
	qb := search.NewQueryBuilder().AddExactMatches(search.DeploymentID, deployment.GetId()).AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String())

	alerts, err := v.searcher.SearchListAlerts(ctx, qb.ProtoQuery())
	if err != nil {
		log.Errorf("Couldn't get risk violations for %s: %s", deployment.GetId(), err)
		return nil
	}

	var severitySum float32
	var count int
	var factors []policyFactor
	for _, alert := range alerts {
		// Some alerts have explicit risk set and some do not.
		if alert.GetPolicy().GetRisk() != nil && alert.GetPolicy().GetRisk().GetBaseScore() != 0 {
			score := alert.GetPolicy().GetRisk().GetBaseScore()
			if alert.GetPolicy().GetRisk().GetCumulative() {
				score *= float32(alert.GetViolationCount())
			}
			if score > alert.GetPolicy().GetRisk().GetMaxScore() {
				score = alert.GetPolicy().GetRisk().GetMaxScore()
			}
			severitySum += score
		} else {
			severitySum += severityImpact(alert.GetPolicy().GetSeverity())
		}

		count++
		factors = append(factors, policyFactor{
			name:     alert.GetPolicy().GetName(),
			severity: alert.GetPolicy().GetSeverity(),
		})
	}

	// This does not contribute to the overall risk of the container
	if severitySum == 0 {
		return nil
	}
	score := multipliers.NormalizeScore(severitySum, policySaturation, policyMaxValue)
	return &storage.Risk_Result{
		Name:    PolicyViolationsHeading,
		Factors: policyFactors(factors),
		Score:   score,
	}
}

func severityImpact(severity storage.Severity) float32 {
	return float32(severity) * float32(severity)
}

func severityString(s storage.Severity) string {
	trim := strings.TrimSuffix(s.String(), "_SEVERITY")
	return strings.ToUpper(trim[:1]) + strings.ToLower(trim[1:])
}

func policyFactors(pfs []policyFactor) (factors []*storage.Risk_Result_Factor) {
	sort.Slice(pfs, func(i, j int) bool {
		if pfs[i].severity == pfs[j].severity {
			// Break ties using the name.
			return pfs[i].name < pfs[j].name
		}
		// Otherwise use the impact score.
		return severityImpact(pfs[i].severity) > severityImpact(pfs[j].severity)
	})

	factors = make([]*storage.Risk_Result_Factor, 0, len(pfs))
	for _, pf := range pfs {
		factors = append(factors,
			&storage.Risk_Result_Factor{Message: fmt.Sprintf("%s (severity: %s)", pf.name, severityString(pf.severity))})
	}
	return
}
