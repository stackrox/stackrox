package graph

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
)

func getMockDeployment(id string) *storage.Deployment {
	return &storage.Deployment{
		Id:          id,
		Namespace:   "default",
		NamespaceId: "default",
		Labels:      deploymentLabels("app", "web"),
		PodLabels:   map[string]string{},
	}
}

func getMockNetworkPolicy(name string) *storage.NetworkPolicy {
	return &storage.NetworkPolicy{
		Name:      name,
		Id:        name,
		Namespace: "default",
		Spec: &storage.NetworkPolicySpec{
			PodSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{name: name},
			},
		},
	}
}

func matchIngress(np *storage.NetworkPolicy, dep *storage.Deployment) {
	spec := np.GetSpec()
	newRule := &storage.NetworkPolicyIngressRule{From: getPeer(dep.GetId())}
	spec.Ingress = append(spec.Ingress, newRule)
}

func matchEgress(np *storage.NetworkPolicy, dep *storage.Deployment) {
	spec := np.GetSpec()
	newRule := &storage.NetworkPolicyEgressRule{To: getPeer(dep.Id)}
	spec.Egress = append(spec.Egress, newRule)
}

func applyPolicy(np *storage.NetworkPolicy, deployment *storage.Deployment) {
	deployment.GetPodLabels()[np.GetName()] = np.GetName()
}

func getPeer(podSelector string) []*storage.NetworkPolicyPeer {
	return []*storage.NetworkPolicyPeer{
		{
			PodSelector: &storage.LabelSelector{
				MatchLabels: map[string]string{podSelector: podSelector},
			},
		},
	}
}

// shuffle and take the first N is equivalent to choosing without replacement.  Has non-impactful side-effects.
func chooseN(n int, deployments []*storage.Deployment) []*storage.Deployment {
	shuffle(deployments)
	return deployments[:n]
}

func shuffle(deployments []*storage.Deployment) {
	rand.Shuffle(len(deployments), func(i, j int) { deployments[i], deployments[j] = deployments[j], deployments[i] })
}

func applyPolicies(networkPolicies []*storage.NetworkPolicy, deployments []*storage.Deployment, numPoliciesToApplyTo int) {
	matchPolicyToRandomDeps(networkPolicies, deployments, numPoliciesToApplyTo, applyPolicy)
}

func matchIngressRules(networkPolicies []*storage.NetworkPolicy, deployments []*storage.Deployment, numIngressRules int) {
	matchPolicyToRandomDeps(networkPolicies, deployments, numIngressRules, matchIngress)
}

func matchEgressRules(networkPolicies []*storage.NetworkPolicy, deployments []*storage.Deployment, numEgressRules int) {
	matchPolicyToRandomDeps(networkPolicies, deployments, numEgressRules, matchEgress)
}

func matchPolicyToRandomDeps(networkPolicies []*storage.NetworkPolicy, deployments []*storage.Deployment, numPoliciesToApplyTo int, applyFunc func(*storage.NetworkPolicy, *storage.Deployment)) {
	for _, policy := range networkPolicies {
		deps := chooseN(numPoliciesToApplyTo, deployments)
		for _, dep := range deps {
			applyFunc(policy, dep)
		}
	}
}

func benchmarkEvaluateCluster(b *testing.B, numDeployments, numNetworkPolicies, numPoliciesApplyTo, ingressMatches, egressMatches int) {
	m := newMockGraphEvaluator()
	deployments := make([]*storage.Deployment, 0, numDeployments)
	for i := 0; i < numDeployments; i++ {
		deployments = append(deployments, getMockDeployment(strconv.Itoa(i)))
	}
	networkPolicies := make([]*storage.NetworkPolicy, 0, numDeployments)
	for i := 0; i < numNetworkPolicies; i++ {
		networkPolicies = append(networkPolicies, getMockNetworkPolicy(fmt.Sprintf("%d", i)))
	}
	applyPolicies(networkPolicies, deployments, numPoliciesApplyTo)
	matchIngressRules(networkPolicies, deployments, ingressMatches)
	matchEgressRules(networkPolicies, deployments, egressMatches)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.GetGraph("", nil, deployments, nil, networkPolicies, false)
	}
}

func BenchmarkEvaluateCluster(b *testing.B) {
	for numDeployments := 1; numDeployments <= 1000; numDeployments *= 10 {
		for numPolicies := 1; numPolicies <= 100; numPolicies *= 10 {
			b.Run(fmt.Sprintf(" %s - %d deployments; %d policies", b.Name(), numDeployments, numPolicies), func(subB *testing.B) {
				benchmarkEvaluateCluster(subB, numDeployments, numPolicies, 0, 0, 0)
			})
		}
	}
}

func BenchmarkDensePolicies(b *testing.B) {
	connectivityPercents := []float32{.1, .5, 1}
	numDeployments := 2000
	numPolicies := 1
	for _, ingressPercent := range connectivityPercents {
		for _, egressPercent := range connectivityPercents {
			numIngress := int(ingressPercent * float32(numDeployments))
			numEgress := int(egressPercent * float32(numDeployments))
			b.Run(fmt.Sprintf(" %s - %d deployments %d policies %d ingress allowed per policy %d egress allowed per policy", b.Name(), numDeployments, numPolicies, numIngress, numEgress), func(dubB *testing.B) {
				benchmarkEvaluateCluster(dubB, numDeployments, numPolicies, numDeployments, numIngress, numEgress)
			})
		}
	}
}

func BenchmarkSparsePolicies(b *testing.B) {
	numDeployments := 10000
	for numPolicies := 1; numPolicies <= 100; numPolicies *= 10 {
		for numApplied := 1; numApplied <= 100; numApplied *= 10 {
			b.Run(fmt.Sprintf(" %s - %d deployments; %d policies each applied to %d nodes", b.Name(), numDeployments, numPolicies, numApplied), func(subB *testing.B) {
				benchmarkEvaluateCluster(subB, numDeployments, numPolicies, numApplied, 5, 5)
			})
		}
	}
}
