package datastore

import (
	"context"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
)

// Gather deployment risk metrics.
// Current properties we gather:
// "Total Deployment Risks"
// "Deployment Risk Score <bucket>"
// "Risk Factor <name>"
var Gather phonehome.GatherFunc = func(ctx context.Context) (map[string]any, error) {
	ctx = sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.DeploymentExtension)))
	props := make(map[string]any)

	q := search.NewQueryBuilder().
		AddExactMatches(search.RiskSubjectType, storage.RiskSubjectType_DEPLOYMENT.String()).
		ProtoQuery()

	risks, err := Singleton().SearchRawRisks(ctx, q)
	if err != nil {
		return nil, err
	}

	_ = phonehome.AddTotal(ctx, props, "Deployment Risks", phonehome.Len(risks))

	scoreBuckets := map[string]int{
		"<1":    0,
		"1-2":   0,
		"2-3":   0,
		"3-4":   0,
		"4-5":   0,
		"5-10":  0,
		"10-20": 0,
		"20+":   0,
	}

	factorCounts := map[string]int{
		"Image Vulnerabilities":           0,
		"Policy Violations":               0,
		"Service Configuration":           0,
		"Service Reachability":            0,
		"Image Freshness":                 0,
		"Components Useful for Attackers": 0,
		"Number of Components in Image":   0,
		"Suspicious Process Executions":   0,
	}

	for _, risk := range risks {
		scoreBuckets[bucketRiskScore(risk.GetScore())]++
		for _, result := range risk.GetResults() {
			if _, tracked := factorCounts[result.GetName()]; tracked {
				factorCounts[result.GetName()]++
			}
		}
	}

	for bucket, count := range scoreBuckets {
		props[fmt.Sprintf("Deployment Risk Score %s", bucket)] = count
	}

	for factor, count := range factorCounts {
		props[fmt.Sprintf("Risk Factor %s", factor)] = count
	}

	return props, nil
}

func bucketRiskScore(score float32) string {
	switch {
	case score < 1:
		return "<1"
	case score < 2:
		return "1-2"
	case score < 3:
		return "2-3"
	case score < 4:
		return "3-4"
	case score < 5:
		return "4-5"
	case score < 10:
		return "5-10"
	case score < 20:
		return "10-20"
	default:
		return "20+"
	}
}
