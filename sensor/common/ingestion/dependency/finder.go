package dependency

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/ingestion"
	"github.com/stackrox/rox/sensor/common/selector"
)

var (
	FinderForKind = map[string]Finder{
		"Deployment": &deploymentFinder{},
		"NetworkPolicy": &networkPolicyFinder{},
	}
)

type Identifier struct {
	Kind, Namespace, Id string
}

type Finder interface {
	FindDependencies(interface{}, *ingestion.ResourceStore) []Identifier
	FindDependants(interface{}, *ingestion.ResourceStore) []Identifier
}

type deploymentFinder struct {}

func (f *deploymentFinder) FindDependencies(raw interface{}, stores *ingestion.ResourceStore) []Identifier {
	deployment, ok := raw.(*storage.Deployment)
	if !ok {
		// TODO: don't panic but return an error
		panic("raw object should be a deployment")
	}

	var result []Identifier
	policies := stores.NetworkPolicy.Find(deployment.Namespace, deployment.Labels)
	for _, pol := range policies {
		result = append(result, Identifier{
			Kind:      "NetworkPolicy",
			Namespace: pol.Namespace,
			Id:        pol.Id,
		})
	}

	// TODO:
	// Find Pods
	// Find Bindings
	// Find Services

	return result
}

func (f *deploymentFinder) FindDependants(deployment interface{}, stores *ingestion.ResourceStore) []Identifier {
	// There are no resources that depend on deployments as they are the top-most resource in
	// the dependency graph
	return []Identifier{}
}

type networkPolicyFinder struct {}

func (f *networkPolicyFinder) FindDependencies(raw interface{}, stores *ingestion.ResourceStore) []Identifier {
	// Network policies don't depend on anything
	return []Identifier{}
}

func (f *networkPolicyFinder) FindDependants(raw interface{}, stores *ingestion.ResourceStore) []Identifier {
	networkPolicy, ok := raw.(*storage.NetworkPolicy)
	if !ok {
		// TODO: don't panic but return an error
		panic("raw object should be a network policy")
	}

	// Just deployments can be dependencies here
	sel := selector.CreateSelector(networkPolicy.GetSpec().GetPodSelector().
		GetMatchLabels(), selector.EmptyMatchesEverything())
	deployments := stores.Deployments.GetMatchingDeployments(networkPolicy.Namespace, sel)
	log.Infof("found %d deployments from selector %+v", len(deployments), sel)
	var result []Identifier
	for _, dep := range deployments {
		result = append(result, Identifier{
			Kind:      "Deployment",
			Namespace: dep.Namespace,
			Id:        dep.Id,
		})
	}
	return result
}
