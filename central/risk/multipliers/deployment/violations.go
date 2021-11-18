package deployment

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/stackrox/rox/central/alert/mappings"
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
	policyMaxValue   = 4
)

var (
	log = logging.LoggerForModule()

	policyNameField = mappings.OptionsMap.MustGet(search.PolicyName.String())
	severityField   = mappings.OptionsMap.MustGet(search.Severity.String())
)

// ViolationsMultiplier is a scorer for the violations on a deployment
type ViolationsMultiplier struct {
	getter getters.AlertGetter
}

type policyFactor struct {
	name     string
	severity storage.Severity
}

// NewViolations scores the data based on the number and severity of policy violations.
func NewViolations(getter getters.AlertGetter) *ViolationsMultiplier {
	return &ViolationsMultiplier{
		getter: getter,
	}
}

// Score takes a deployment and evaluates its risk based on policy violations.
func (v *ViolationsMultiplier) Score(ctx context.Context, deployment *storage.Deployment, _ map[string][]*storage.Risk_Result) *storage.Risk_Result {
	qb := search.NewQueryBuilder().
		AddExactMatches(search.DeploymentID, deployment.GetId()).
		AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).
		AddStringsHighlighted(search.PolicyName, search.WildcardString).
		AddStringsHighlighted(search.Severity, search.WildcardString)

	results, err := v.getter.Search(ctx, qb.ProtoQuery())
	if err != nil {
		log.Errorf("Couldn't get risk violations for %s: %s", deployment.GetId(), err)
		return nil
	}

	var severitySum float32
	var count int
	var factors []policyFactor
	for _, result := range results {
		count++

		severityStr, ok := result.Matches[severityField.FieldPath]
		if !ok {
			log.Error("UNEXPECTED: could not retrieve severity from alert")
			continue
		}
		if len(severityStr) != 1 {
			log.Errorf("UNEXPECTED: number of severities (%d) does not equal one", len(severityStr))
			continue
		}

		severityInt, err := strconv.Atoi(severityStr[0])
		if err != nil {
			log.Errorf("UNEXPECTED: could not convert severity %s to integer: %v", severityStr, err)
			continue
		}
		severity := storage.Severity(severityInt)

		policyName, ok := result.Matches[policyNameField.FieldPath]
		if !ok {
			log.Error("UNEXPECTED: could not retrieve policy name from alert")
			continue
		}
		if len(policyName) != 1 {
			log.Errorf("UNEXPECTED: number of policy names (%d) does not equal one", len(policyName))
			continue
		}

		severitySum += severityImpact(severity)
		factors = append(factors, policyFactor{
			name:     policyName[0],
			severity: severity,
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
