package sac

import (
	"reflect"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

// ScopeState represents possible states of a scope.
type ScopeState int

const (
	// Excluded indicates that neither the scope nor its subtree are included.
	Excluded ScopeState = iota
	// Partial indicates that the scope is not included in its entirety but some
	// of its children are included.
	Partial
	// Included indicates that the scope is included with its subtree.
	Included
)

var (
	log = logging.LoggerForModule()
)

// EffectiveAccessScopeTree is a tree of scopes with their states.
type EffectiveAccessScopeTree struct {
	state    ScopeState
	clusters map[string]*ClustersScopeSubTree
}

// ClustersScopeSubTree is a subtree of cluster scopes with their states.
type ClustersScopeSubTree struct {
	state      ScopeState
	namespaces map[string]*NamespacesScopeSubTree
}

// NamespacesScopeSubTree is a subtree of namespace scopes with their states.
type NamespacesScopeSubTree struct {
	state ScopeState
}

func newEffectiveAccessScopeTree(state ScopeState) *EffectiveAccessScopeTree {
	return &EffectiveAccessScopeTree{
		state:    state,
		clusters: make(map[string]*ClustersScopeSubTree),
	}
}

func newClusterScopeSubTree(state ScopeState) *ClustersScopeSubTree {
	return &ClustersScopeSubTree{
		state:      state,
		namespaces: make(map[string]*NamespacesScopeSubTree),
	}
}

func newNamespacesScopeSubTree(state ScopeState) *NamespacesScopeSubTree {
	return &NamespacesScopeSubTree{
		state: state,
	}
}

const (
	// FQSN stands for "fully qualified scope name".
	clusterNameLabel   = "stackrox.io/authz.metadata.cluster.fqsn"
	namespaceNameLabel = "stackrox.io/authz.metadata.namespace.fqsn"
	scopeSeparator     = "::"
)

// ComputeEffectiveAccessScope applies a simple access scope to provided
// clusters and namespaces and yields EffectiveAccessScopeTree. Note that:
//   * no (nil) scope => everything is included,
//   * empty scope => nothing is included.
func ComputeEffectiveAccessScope(scope *storage.SimpleAccessScope, clusters []*storage.Cluster, namespaces []*storage.NamespaceMetadata) (*EffectiveAccessScopeTree, error) {
	var root *EffectiveAccessScopeTree
	if scope == nil {
		root = newEffectiveAccessScopeTree(Included)
	} else {
		root = newEffectiveAccessScopeTree(Excluded)
	}

	// Compile scope into cluster and namespace selectors.
	clusterSelectors, namespaceSelectors, err := getAugmentedSelectors(scope)
	if err != nil {
		return nil, err
	}

	// Check every cluster against corresponding access scope rules represented
	// by clusterSelectors (note cluster name to label conversion). Partial
	// state is not possible here yet.
	for _, cluster := range clusters {
		populateStateForCluster(cluster, root, clusterSelectors)
	}

	// Check every namespace not indirectly included by its parent cluster
	// against corresponding access scope rules represented by
	// namespaceSelectors (note namespace name to label conversion).
	for _, namespace := range namespaces {
		clusterName := namespace.GetClusterName()
		namespaceFQSN := getNamespaceFQSN(clusterName, namespace.GetName())

		// If parent cluster is unknown, log and add cluster as Excluded.
		parentCluster := root.clusters[clusterName]
		if parentCluster == nil {
			log.Warnf("namespace %q belongs to unknown cluster %q", namespaceFQSN, clusterName)
			parentCluster = newClusterScopeSubTree(Excluded)
			root.clusters[clusterName] = parentCluster
		}

		populateStateForNamespace(namespace, parentCluster, namespaceSelectors)
	}

	// Recursively update parent nodes.
	bubbleUpStates(root)

	return root, nil
}

// populateStateForCluster adds given cluster as Included or Excluded to root.
// Only the last observed cluster is considered if multiple ones with the same
// name exist.
func populateStateForCluster(cluster *storage.Cluster, root *EffectiveAccessScopeTree, clusterSelectors []labels.Selector) {
	clusterName := cluster.GetName()

	// If root is Included, include the cluster and short-circuit:
	// no need to match if parent is included.
	if root.state == Included {
		root.clusters[clusterName] = newClusterScopeSubTree(Included)
		return
	}

	// Augment cluster labels with cluster's name.
	clusterLabels := augmentLabels(cluster.GetLabels(), clusterNameLabel, clusterName)

	// Match and update the tree.
	matched := matchLabels(clusterSelectors, clusterLabels)
	root.clusters[clusterName] = newClusterScopeSubTree(matched)
}

// populateStateForNamespace adds given namespace as Included or Excluded to
// parent cluster. Only the last observed namespace is considered if multiple
// ones with the same <cluster name, namespace name> exist.
func populateStateForNamespace(namespace *storage.NamespaceMetadata, parentCluster *ClustersScopeSubTree, namespaceSelectors []labels.Selector) {
	clusterName := namespace.GetClusterName()
	namespaceName := namespace.GetName()
	namespaceFQSN := getNamespaceFQSN(clusterName, namespaceName)

	// If parent is Included, include the namespace and short-circuit:
	// no need to match if parent is included.
	if parentCluster.state == Included {
		parentCluster.namespaces[namespaceName] = newNamespacesScopeSubTree(Included)
		return
	}

	// Augment namespace labels with namespace's FQSN.
	namespaceLabels := augmentLabels(namespace.GetLabels(), namespaceNameLabel, namespaceFQSN)

	// Match and update the tree.
	matched := matchLabels(namespaceSelectors, namespaceLabels)
	parentCluster.namespaces[namespaceName] = newNamespacesScopeSubTree(matched)
}

// getAugmentedSelectors:
//   * converts included_clusters rules to a single cluster label selector,
//   * converts included_namespaces rules to a single namespace label selector,
//   * converts all label selectors to standard ones with matching support.
func getAugmentedSelectors(scope *storage.SimpleAccessScope) ([]labels.Selector, []labels.Selector, error) {
	// No scope means no restrictions.
	if scope == nil {
		return []labels.Selector{labels.Everything()}, []labels.Selector{labels.Everything()}, nil
	}

	// Convert each selector to labels.Selector.
	clusterSelectors, err := convertEachSetBasedLabelSelectorToK8sLabelSelector(scope.GetRules().GetClusterLabelSelectors())
	if err != nil {
		return nil, nil, errors.Wrap(err, "bad cluster label selector")
	}

	// Add included cluster names as a special label.
	if clusterNames := scope.GetRules().GetIncludedClusters(); len(clusterNames) != 0 {
		selector := labels.NewSelector()
		req, err := labels.NewRequirement(clusterNameLabel, selection.In, clusterNames)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "label selector from cluster names %v", clusterNames)
		}
		clusterSelectors = append(clusterSelectors, selector.Add(*req))
	}

	// Convert each selector to labels.Selector.
	namespaceSelectors, err := convertEachSetBasedLabelSelectorToK8sLabelSelector(scope.GetRules().GetNamespaceLabelSelectors())
	if err != nil {
		return nil, nil, errors.Wrap(err, "bad namespace label selector")
	}

	// Add included namespace names as a special label. Note how validation of
	// label keys and values is bypassed when creating labels.Requirement.
	if namespaceNames := scope.GetRules().GetIncludedNamespaces(); len(namespaceNames) != 0 {
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
	compiled := labels.NewSelector()
	for _, elem := range selector.GetRequirements() {
		req, err := labels.NewRequirement(elem.GetKey(), convertLabelSelectorOperatorToSelectionOperator(elem.GetOp()), elem.GetValues())
		if err != nil {
			return nil, err
		}
		compiled = compiled.Add(*req)
	}

	return compiled, nil
}

func convertLabelSelectorOperatorToSelectionOperator(op storage.SetBasedLabelSelector_Operator) selection.Operator {
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

// newUnvalidatedRequirement is like labels.NewRequirement() but without label
// key and values validation. Fully qualified scope names:
//   * contain a separator which must be forbidden in label values;
//   * might exceed 63 length limit.
// The hacks below enable us to create labels.Requirement for FQSN and hence
// embed the by-name inclusions into the general selector matching approach.
func newUnvalidatedRequirement(key string, op selection.Operator, values []string) (*labels.Requirement, error) {
	req := &labels.Requirement{}
	reqUnleashed := reflect.ValueOf(req).Elem()

	setValue := func(fieldName string, value interface{}) {
		field := reqUnleashed.FieldByName(fieldName)
		field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
		field.Set(reflect.ValueOf(value).Elem())
	}

	setValue("key", &key)
	setValue("operator", &op)
	setValue("strValues", &values)

	return req, nil
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

func getNamespaceFQSN(cluster string, namespace string) string {
	return cluster + scopeSeparator + namespace
}

func augmentLabels(labels map[string]string, key string, value string) map[string]string {
	result := make(map[string]string)
	for k, v := range labels {
		result[k] = v
	}
	result[key] = value

	return result
}

// matchLabels checks if any of the given selectors matches the given label map.
func matchLabels(selectors []labels.Selector, lbls map[string]string) ScopeState {
	for _, s := range selectors {
		if s.Matches(labels.Set(lbls)) {
			return Included
		}
	}
	return Excluded
}

// bubbleUpStates updates the state of parent nodes based on the state of their
// children. If any child is Included or Partial, its parent becomes at least
// Partial. If all children are Included, the parent will still be Partial
// unless it has been included directly.
func bubbleUpStates(root *EffectiveAccessScopeTree) {
	for _, cluster := range root.clusters {
		for _, namespace := range cluster.namespaces {
			// Namespaces are currently tree leaves hence we can short-circuit.
			if cluster.state != Excluded {
				break
			}
			if namespace.state == Included || namespace.state == Partial {
				cluster.state = Partial
				break
			}
		}
		if root.state == Excluded && (cluster.state == Included || cluster.state == Partial) {
			root.state = Partial
		}
	}
}
