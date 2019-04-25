package search

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stackrox/rox/central/deployment/index"
	"github.com/stackrox/rox/central/deployment/index/mappings"
	"github.com/stackrox/rox/central/globalindex"
	imageIndexer "github.com/stackrox/rox/central/image/index"
	processIndicatorDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	processIndicatorIndex "github.com/stackrox/rox/central/processindicator/index"
	processIndicatorSearch "github.com/stackrox/rox/central/processindicator/search"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/central/searchbasedpolicies/fields"
	"github.com/stackrox/rox/central/searchbasedpolicies/matcher"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/image/policies"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/defaults"
	"github.com/stackrox/rox/pkg/fixtures"
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

func setup(b require.TestingT) (processIndicatorDataStore.DataStore, imageIndexer.Indexer, index.Indexer) {
	db, err := bolthelper.NewTemp("bench_test.db")
	require.NoError(b, err)

	bleveIndex, err := globalindex.TempInitializeIndices("")
	require.NoError(b, err)

	processStore := processIndicatorStore.New(db)
	processIndexer := processIndicatorIndex.New(bleveIndex)
	processSearcher, err := processIndicatorSearch.New(processStore, processIndexer)
	require.NoError(b, err)

	deploymentIndexer := index.New(bleveIndex)
	imageIdx := imageIndexer.New(bleveIndex)

	processDataStore := processIndicatorDataStore.New(processStore, processIndexer, processSearcher, nil)
	return processDataStore, imageIdx, deploymentIndexer
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

func BenchmarkPolicies(b *testing.B) {
	policies := getPolicies(b)

	numDeployments := []int{10000}
	numProcessIndicators := []int{100000}

	for _, dNum := range numDeployments {
		for _, pNum := range numProcessIndicators {
			processDatastore, _, indexer := setup(b)
			require.NoError(b, indexer.AddDeployments(getDeployments(dNum)))
			require.NoError(b, processDatastore.AddProcessIndicators(getProcesses(dNum, pNum)...))
			matcherBuilder := matcher.NewBuilder(fields.NewRegistry(processDatastore), mappings.OptionsMap)
			for _, p := range policies {
				b.Run(fmt.Sprintf("%s %dd %dp", p.GetName(), dNum, pNum), func(b *testing.B) {
					mr, err := matcherBuilder.ForPolicy(p)
					require.NoError(b, err)
					for i := 0; i < b.N; i++ {
						_, err = mr.Match(indexer)
						require.NoError(b, err)
					}
				})
			}
		}
	}
}
