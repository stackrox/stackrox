package graph

import "github.com/stackrox/stackrox/pkg/labels"

func matchDeployments(nodes []*node, podSel labels.CompiledSelector) []*node {
	if podSel.MatchesNone() {
		return nil
	}
	var result []*node
	if podSel.MatchesAll() {
		for _, node := range nodes {
			if node.deployment != nil {
				result = append(result, node)
			}
		}
		return result
	}

	for _, node := range nodes {
		if node.deployment == nil {
			continue
		}

		if podSel.Matches(node.deployment.GetPodLabels()) {
			result = append(result, node)
		}
	}
	return result
}
