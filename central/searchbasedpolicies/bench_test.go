package searchbasedpolicies

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/policies"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/matcher"
	"github.com/stretchr/testify/require"
)

func getPolicies(b require.TestingT) []*storage.Policy {
	defaults.PoliciesPath = policies.Directory()
	defaultPolicies, err := defaults.Policies()
	require.NoError(b, err)

	deployAndRuntimePolicies := defaultPolicies[:0]

policyLoop:
	for _, p := range defaultPolicies {
		for _, ls := range p.GetLifecycleStages() {
			if ls != storage.LifecycleStage_BUILD {
				deployAndRuntimePolicies = append(deployAndRuntimePolicies, p)
				continue policyLoop
			}
		}
	}
	sort.SliceStable(deployAndRuntimePolicies, func(i, j int) bool {
		return deployAndRuntimePolicies[i].GetName() < deployAndRuntimePolicies[j].GetName()
	})
	return deployAndRuntimePolicies
}

func getDeployments(num int) (deployments []*storage.Deployment) {
	deployments = make([]*storage.Deployment, 0, num)
	for i := 0; i < num; i++ {
		deployment := fixtures.GetDeployment()
		deployment.Id = fmt.Sprintf("%d", i)
		deployments = append(deployments, deployment)
	}
	return
}

func getProcesses(dNum, pNum int) (processes []*storage.ProcessIndicator) {
	processes = make([]*storage.ProcessIndicator, 0, pNum)
	for i := 0; i < pNum; i++ {
		indicator := fixtures.GetProcessIndicator()
		indicator.Id = fmt.Sprintf("%d", i)
		indicator.DeploymentId = fmt.Sprintf("%d", i%dNum)
		processes = append(processes, indicator)
	}
	return
}

func BenchmarkPoliciesMatchOne(b *testing.B) {
	policies := getPolicies(b)

	numDeployments := []int{10}
	numProcessIndicators := []int{10}

	matchCtx := sac.WithGlobalAccessScopeChecker(context.Background(), sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS)))
	for _, dNum := range numDeployments {
		for _, pNum := range numProcessIndicators {
			mockDeployments := getDeployments(dNum)
			mockProcesses := getProcesses(dNum, pNum)
			matcherBuilder := matcher.NewBuilder(matcher.NewRegistry(nil), deployments.OptionsMap)

			for _, p := range policies {
				b.Run(fmt.Sprintf("%s %dd %dp", p.GetName(), dNum, pNum), func(b *testing.B) {
					mr, err := matcherBuilder.ForPolicy(p)
					require.NoError(b, err)
					for i := 0; i < b.N; i++ {
						for _, deployment := range mockDeployments {
							for _, indicator := range mockProcesses {
								_, err = mr.MatchOne(matchCtx, deployment, nil, indicator)
								require.NoError(b, err)
							}
						}
					}
				})
			}
		}
	}
}
