package graph

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
)

func getMockDeployment(id string) *storage.Deployment {
	return &storage.Deployment{
		Id:        id,
		Namespace: "default",
		Labels:    deploymentLabels("app", "web"),
	}
}

func getMockNetworkPolicy(name string) *storage.NetworkPolicy {
	return &storage.NetworkPolicy{
		Name: name,
	}
}

func benchmarkEvaluateCluster(b *testing.B, numDeployments, numNetworkPolicies int) {
	m := newMockGraphEvaluator()
	deployments := make([]*storage.Deployment, 0, numDeployments)
	for i := 0; i < numDeployments; i++ {
		deployments = append(deployments, getMockDeployment(fmt.Sprintf("%d", i)))
	}
	networkPolicies := make([]*storage.NetworkPolicy, 0, numDeployments)
	for i := 0; i < numNetworkPolicies; i++ {
		networkPolicies = append(networkPolicies, getMockNetworkPolicy(fmt.Sprintf("%d", i)))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.evaluate(deployments, networkPolicies)
	}
}

func BenchmarkEvaluateCluster(b *testing.B) {
	for numDeployments := 1; numDeployments <= 1000; numDeployments *= 10 {
		for numPolicies := 1; numPolicies <= 100; numPolicies *= 10 {
			b.Run(fmt.Sprintf(" %s - %d deployments; %d policies", b.Name(), numDeployments, numPolicies), func(subB *testing.B) {
				benchmarkEvaluateCluster(subB, numDeployments, numPolicies)
			})
		}
	}
}
