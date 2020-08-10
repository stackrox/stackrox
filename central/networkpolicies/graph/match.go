package graph

import "github.com/stackrox/rox/pkg/labels"

func matchDeployments(deployments []*node, podSel labels.CompiledSelector) []*node {
	if podSel.MatchesNone() {
		return nil
	}
	if podSel.MatchesAll() {
		return deployments
	}

	var result []*node
	for _, deployment := range deployments {
		if podSel.Matches(deployment.GetPodLabels()) {
			result = append(result, deployment)
		}
	}
	return result
}
