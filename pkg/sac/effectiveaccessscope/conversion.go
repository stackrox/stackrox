package effectiveaccessscope

import (
	"sort"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// ToEffectiveAccessScope converts effective access scope tree with enriched
// nodes to storage.EffectiveAccessScope.
func ToEffectiveAccessScope(tree *ScopeTree) (*storage.EffectiveAccessScope, error) {
	response := &storage.EffectiveAccessScope{}
	if len(tree.Clusters) != 0 {
		response.Clusters = make([]*storage.EffectiveAccessScope_Cluster, 0, len(tree.Clusters))
	}

	for _, clusterSubTree := range tree.Clusters {
		cluster := &storage.EffectiveAccessScope_Cluster{
			Id:     clusterSubTree.Attributes.ID,
			Name:   clusterSubTree.Attributes.Name,
			State:  convertScopeStateToEffectiveAccessScopeState(clusterSubTree.State),
			Labels: clusterSubTree.Attributes.Labels,
		}
		if len(clusterSubTree.Namespaces) != 0 {
			cluster.Namespaces = make([]*storage.EffectiveAccessScope_Namespace, 0, len(clusterSubTree.Namespaces))
		}

		for _, namespaceSubTree := range clusterSubTree.Namespaces {
			namespace := &storage.EffectiveAccessScope_Namespace{
				Id:     namespaceSubTree.Attributes.ID,
				Name:   namespaceSubTree.Attributes.Name,
				State:  convertScopeStateToEffectiveAccessScopeState(namespaceSubTree.State),
				Labels: namespaceSubTree.Attributes.Labels,
			}

			cluster.Namespaces = append(cluster.Namespaces, namespace)
		}

		response.Clusters = append(response.Clusters, cluster)
	}

	// Ensure order consistency across invocations.
	sortScopesInEffectiveAccessScope(response)

	return response, nil
}

// ConvertLabelSelectorOperatorToSelectionOperator translates storage selection operator into k8s type.
func ConvertLabelSelectorOperatorToSelectionOperator(op storage.SetBasedLabelSelector_Operator) selection.Operator {
	switch op {
	case storage.SetBasedLabelSelector_IN:
		return selection.In
	case storage.SetBasedLabelSelector_NOT_IN:
		return selection.NotIn
	case storage.SetBasedLabelSelector_EXISTS:
		return selection.Exists
	case storage.SetBasedLabelSelector_NOT_EXISTS:
		return selection.DoesNotExist
	default:
		return selection.Operator(op.String())
	}
}

func convertScopeStateToEffectiveAccessScopeState(scopeState scopeState) storage.EffectiveAccessScope_State {
	switch scopeState {
	case Excluded:
		return storage.EffectiveAccessScope_EXCLUDED
	case Partial:
		return storage.EffectiveAccessScope_PARTIAL
	case Included:
		return storage.EffectiveAccessScope_INCLUDED
	default:
		return storage.EffectiveAccessScope_UNKNOWN
	}
}

func sortScopesInEffectiveAccessScope(msg *storage.EffectiveAccessScope) {
	clusters := msg.GetClusters()
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].GetId() < clusters[j].GetId()
	})

	for _, cluster := range clusters {
		namespaces := cluster.GetNamespaces()
		sort.Slice(namespaces, func(i, j int) bool {
			return namespaces[i].GetId() < namespaces[j].GetId()
		})
	}
}

// convertRulesToSelectors:
//   - converts included_clusters rules to a cluster name matching map,
//   - converts included_namespaces rules to a namespace matching map (parent cluster is identified by name),
//   - converts all label selectors to standard ones with matching support.
func convertRulesToSelectors(scopeRules *storage.SimpleAccessScope_Rules) (*selectors, error) {
	output := &selectors{}
	if scopeRules == nil {
		return output, nil
	}

	// Convert each selector to labels.Selector.
	clusterSelectors, clusterSelectorErr := convertEachSetBasedLabelSelectorToK8sLabelSelector(scopeRules.GetClusterLabelSelectors())
	if clusterSelectorErr != nil {
		return nil, errors.Wrap(clusterSelectorErr, "bad cluster label selector")
	}
	output.clustersByLabel = clusterSelectors

	output.clustersByName = set.NewStringSet(scopeRules.GetIncludedClusters()...)

	// Convert each selector to labels.Selector.
	namespaceSelectors, namespaceSelectorErr := convertEachSetBasedLabelSelectorToK8sLabelSelector(scopeRules.GetNamespaceLabelSelectors())
	if namespaceSelectorErr != nil {
		return nil, errors.Wrap(namespaceSelectorErr, "bad namespace label selector")
	}
	output.namespacesByLabel = namespaceSelectors

	includedNamespaces := scopeRules.GetIncludedNamespaces()
	output.namespacesByClusterName = make(map[string]set.StringSet, len(includedNamespaces))
	for _, namespace := range includedNamespaces {
		clusterName := namespace.GetClusterName()
		namespaceName := namespace.GetNamespaceName()
		if clusterName == "" {
			continue
		}
		if namespaceName == "" {
			continue
		}
		addToNamespaceMap(output.namespacesByClusterName, clusterName, namespaceName)
	}

	return output, nil
}

func addToNamespaceMap(targetMap map[string]set.StringSet, clusterKey string, namespaceKey string) {
	if _, exists := targetMap[clusterKey]; !exists {
		targetMap[clusterKey] = set.NewStringSet()
	}
	clusterNamespaces := targetMap[clusterKey]
	clusterNamespaces.Add(namespaceKey)
}

func convertEachSetBasedLabelSelectorToK8sLabelSelector(selectors []*storage.SetBasedLabelSelector) ([]labels.Selector, error) {
	converted := make([]labels.Selector, 0, len(selectors))
	for _, elem := range selectors {
		compiled, err := convertSetBasedLabelSelectorToK8sLabelSelector(elem)
		if err != nil {
			return nil, err
		}
		converted = append(converted, compiled)
	}
	return converted, nil
}

// convertSetBasedLabelSelectorToK8sLabelSelector converts SetBasedLabelSelector
// protobuf to the standard labels.Selector type that supports matching.
func convertSetBasedLabelSelectorToK8sLabelSelector(selector *storage.SetBasedLabelSelector) (labels.Selector, error) {
	// We want empty requirements map to nothing and not every label.
	if len(selector.GetRequirements()) == 0 {
		return labels.Nothing(), nil
	}

	compiled := labels.NewSelector()
	for _, elem := range selector.GetRequirements() {
		req, err := labels.NewRequirement(elem.GetKey(), ConvertLabelSelectorOperatorToSelectionOperator(elem.GetOp()), elem.GetValues())
		if err != nil {
			return nil, err
		}
		compiled = compiled.Add(*req)
	}

	return compiled, nil
}
