package effectiveaccessscope

import (
	"reflect"
	"sort"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
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

// convertRulesToLabelSelectors:
//   - converts included_clusters rules to a single cluster label selector,
//   - converts included_namespaces rules to a single namespace label selector,
//   - converts all label selectors to standard ones with matching support.
func convertRulesToLabelSelectors(scopeRules *storage.SimpleAccessScope_Rules) (clusterSelectors, namespaceSelectors []labels.Selector, err error) {
	// Convert each selector to labels.Selector.
	clusterSelectors, err = convertEachSetBasedLabelSelectorToK8sLabelSelector(scopeRules.GetClusterLabelSelectors())
	if err != nil {
		return nil, nil, errors.Wrap(err, "bad cluster label selector")
	}

	// Add included cluster names as a special label.
	if clusterNames := scopeRules.GetIncludedClusters(); len(clusterNames) != 0 {
		selector := labels.NewSelector()
		req, err := labels.NewRequirement(clusterNameLabel, selection.In, clusterNames)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "label selector from cluster names %v", clusterNames)
		}
		clusterSelectors = append(clusterSelectors, selector.Add(*req))
	}

	// Convert each selector to labels.Selector.
	namespaceSelectors, err = convertEachSetBasedLabelSelectorToK8sLabelSelector(scopeRules.GetNamespaceLabelSelectors())
	if err != nil {
		return nil, nil, errors.Wrap(err, "bad namespace label selector")
	}

	// Add included namespace names as a special label. Note how validation of
	// label keys and values is bypassed when creating labels.Requirement.
	if namespaceNames := scopeRules.GetIncludedNamespaces(); len(namespaceNames) != 0 {
		selector := labels.NewSelector()
		req, err := newUnvalidatedRequirement(namespaceNameLabel, selection.In, convertEachRulesNamespaceToFQSN(namespaceNames))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "label selector from namespace names %v", namespaceNames)
		}
		namespaceSelectors = append(namespaceSelectors, selector.Add(*req))
	}

	return clusterSelectors, namespaceSelectors, nil
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

// convertEachRulesNamespaceToFQSN (fully qualified scope name) converts
// Namespace{cluster_name: "foo", namespace_name: "bar"} to "foo::bar".
func convertEachRulesNamespaceToFQSN(namespaces []*storage.SimpleAccessScope_Rules_Namespace) []string {
	result := make([]string, 0, len(namespaces))
	for _, elem := range namespaces {
		result = append(result, getNamespaceFQSN(elem.GetClusterName(), elem.GetNamespaceName()))
	}
	return result
}

// newUnvalidatedRequirement is like labels.NewRequirement() but without label
// key and values validation. Fully qualified scope names:
//   - contain a separator which must be forbidden in label values;
//   - might exceed 63 length limit.
//
// The hacks below enable us to create labels.Requirement for FQSN and hence
// embed the by-name inclusions into the general selector matching approach.
func newUnvalidatedRequirement(key string, op selection.Operator, values []string) (*labels.Requirement, error) {
	req := &labels.Requirement{}
	reqUnleashed := reflect.ValueOf(req).Elem()

	setValue := func(fieldName string, value interface{}) {
		field := reqUnleashed.FieldByName(fieldName)
		//#nosec G103
		field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
		field.Set(reflect.ValueOf(value).Elem())
	}

	setValue("key", &key)
	setValue("operator", &op)
	setValue("strValues", &values)

	return req, nil
}
