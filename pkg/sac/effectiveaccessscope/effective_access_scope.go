package effectiveaccessscope

import (
	"sort"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/apimachinery/pkg/labels"
)

// scopeState represents possible states of a scope.
type scopeState int

const (
	// Excluded indicates that neither the scope nor its subtree are included.
	Excluded scopeState = iota
	// Partial indicates that the scope is not included in its entirety but some
	// of its children are included.
	Partial
	// Included indicates that the scope is included with its subtree.
	Included

	// FQSN stands for "fully qualified scope name".
	clusterNameLabel   = "stackrox.io/authz.metadata.cluster.fqsn"
	namespaceNameLabel = "stackrox.io/authz.metadata.namespace.fqsn"
	scopeSeparator     = "::"
)

var (
	log = logging.LoggerForModule()
)

// ScopeTree is a tree of scopes with their states.
type ScopeTree struct {
	State           scopeState
	Clusters        map[string]*clustersScopeSubTree // keyed by cluster name
	clusterIDToName map[string]string
}

// clustersScopeSubTree is a subtree of cluster scopes with their states.
// Attributes field optionally stores additional node info, e.g., cluster id,
// cluster name, labels.
type clustersScopeSubTree struct {
	State      scopeState
	Namespaces map[string]*namespacesScopeSubTree // keyed by namespace name
	Attributes treeNodeAttributes
}

// namespacesScopeSubTree is a subtree of namespace scopes with their states.
// Attributes field optionally stores additional node info, e.g., namespace id,
// namespace name, labels.
type namespacesScopeSubTree struct {
	State      scopeState
	Attributes treeNodeAttributes
}

func (n *namespacesScopeSubTree) copy() *namespacesScopeSubTree {
	return &namespacesScopeSubTree{
		State:      n.State,
		Attributes: *n.Attributes.copy(),
	}
}

// UnrestrictedEffectiveAccessScope returns ScopeTree allowing everything
// implicitly via marking the root Included.
func UnrestrictedEffectiveAccessScope() *ScopeTree {
	return newEffectiveAccessScopeTree(Included)
}

// DenyAllEffectiveAccessScope returns ScopeTree denying everything
// implicitly via marking the root Excluded.
func DenyAllEffectiveAccessScope() *ScopeTree {
	return newEffectiveAccessScopeTree(Excluded)
}

// ComputeEffectiveAccessScope applies a simple access scope to provided
// clusters and namespaces and yields ScopeTree. Empty access scope rules
// mean nothing is included.
func ComputeEffectiveAccessScope(scopeRules *storage.SimpleAccessScope_Rules, clusters []ClusterForSAC, namespaces []NamespaceForSAC, detail v1.ComputeEffectiveAccessScopeRequest_Detail) (*ScopeTree, error) {
	root := newEffectiveAccessScopeTree(Excluded)

	// Compile scope into cluster and namespace selectors.
	clusterSelectors, namespaceSelectors, err := convertRulesToLabelSelectors(scopeRules)
	if err != nil {
		return nil, err
	}

	// Check every cluster against corresponding access scope rules represented
	// by clusterSelectors (note cluster name to label conversion). Partial
	// state is not possible here yet.
	for _, cluster := range clusters {
		root.populateStateForCluster(cluster, clusterSelectors, detail)
	}

	// Check every namespace not indirectly included by its parent cluster
	// against corresponding access scope rules represented by
	// namespaceSelectors (note namespace name to label conversion).
	for _, namespace := range namespaces {
		clusterName := namespace.GetClusterName()
		namespaceFQSN := getNamespaceFQSN(clusterName, namespace.GetName())

		// If parent cluster is unknown, log and add cluster as Excluded.
		parentCluster := root.Clusters[clusterName]
		if parentCluster == nil {
			log.Warnf("namespace %q belongs to unknown cluster %q", namespaceFQSN, clusterName)
			parentCluster = newClusterScopeSubTree(Excluded, treeNodeAttributes{Name: clusterName})
			root.Clusters[clusterName] = parentCluster
		}

		parentCluster.populateStateForNamespace(namespace, namespaceSelectors, detail)
	}

	root.bubbleUpStatesAndCompactify(detail)

	return root, nil
}

// Compactify yields a compact representation of the scope tree.
func (root *ScopeTree) Compactify() ScopeTreeCompacted {
	compacted := make(ScopeTreeCompacted)

	switch root.State {
	case Excluded:
		return compacted
	case Included:
		compacted["*"] = []string{"*"}
		return compacted
	}

	// `root.State` is Partial.
	for clusterName, clusterSubTree := range root.Clusters {
		switch clusterSubTree.State {
		case Excluded:
			continue
		case Included:
			compacted[clusterName] = []string{"*"}
			continue
		}

		// `clusterSubTree.State` is Partial.
		namespaces := make([]string, 0)
		for namespaceName, namespaceSubTree := range clusterSubTree.Namespaces {
			switch namespaceSubTree.State {
			case Excluded:
				// Skip to the next one.
			case Included:
				namespaces = append(namespaces, namespaceName)
			}
		}
		// Ensure order consistency across invocations.
		sort.Slice(namespaces, func(i, j int) bool {
			return namespaces[i] < namespaces[j]
		})
		compacted[clusterName] = namespaces
	}

	return compacted
}

// FromClustersAndNamespacesMap will build and return a ScopeTree that allows access to all clusters and
// (cluster, namespace) pairs provided in input.
func FromClustersAndNamespacesMap(includedClusters []string, includedNamespaces map[string][]string) *ScopeTree {
	if len(includedClusters) == 0 && len(includedNamespaces) == 0 {
		return DenyAllEffectiveAccessScope()
	}
	root := &ScopeTree{
		State:           Partial,
		clusterIDToName: make(map[string]string, 0),
		Clusters:        make(map[string]*clustersScopeSubTree, 0),
	}
	includedClusterSubTree := &clustersScopeSubTree{
		State: Included,
	}
	includedNamespaceSubTree := &namespacesScopeSubTree{
		State: Included,
	}
	for _, clusterID := range includedClusters {
		root.clusterIDToName[clusterID] = clusterID
		root.Clusters[clusterID] = includedClusterSubTree
	}
	for clusterID, namespaces := range includedNamespaces {
		root.clusterIDToName[clusterID] = clusterID
		clusterTree := root.Clusters[clusterID]
		if clusterTree == nil && len(namespaces) == 0 {
			continue
		}
		if clusterTree == nil {
			clusterTree = &clustersScopeSubTree{
				State:      Partial,
				Namespaces: make(map[string]*namespacesScopeSubTree, 0),
			}
		}
		for _, namespace := range namespaces {
			clusterTree.Namespaces[namespace] = includedNamespaceSubTree
		}
		root.Clusters[clusterID] = clusterTree
	}
	return root
}

// String yields a compacted one-line string representation.
func (root *ScopeTree) String() string {
	return root.Compactify().String()
}

// ToJSON yields a compacted JSON representation.
func (root *ScopeTree) ToJSON() (string, error) {
	if root == nil {
		return "{}", nil
	}
	return root.Compactify().ToJSON()
}

// GetClusterIDs returns the list of cluster IDs known to the current scope tree
func (root *ScopeTree) GetClusterIDs() []string {
	clusterIDs := make([]string, 0, len(root.clusterIDToName))
	for k := range root.clusterIDToName {
		clusterIDs = append(clusterIDs, k)
	}
	return clusterIDs
}

// GetClusterByID returns ClusterScopeSubTree for given cluster ID.
// Returns nil when clusterID is not known.
func (root *ScopeTree) GetClusterByID(clusterID string) *clustersScopeSubTree {
	return root.Clusters[root.clusterIDToName[clusterID]]
}

// populateStateForCluster adds given cluster as Included or Excluded to root.
// Only the last observed cluster is considered if multiple ones with the same
// name exist.
func (root *ScopeTree) populateStateForCluster(cluster ClusterForSAC, clusterSelectors []labels.Selector, detail v1.ComputeEffectiveAccessScopeRequest_Detail) {
	clusterName := cluster.GetName()

	// There is no need to check if root is Included as we start with Excluded root.
	// If it will be Included then we can include the cluster and short-circuit:
	// no need to match if parent is included.

	// Augment cluster labels with cluster's name.
	clusterLabels := augmentLabels(cluster.GetLabels(), clusterNameLabel, clusterName)

	// Match and update the tree.
	matched := matchLabels(clusterSelectors, clusterLabels)
	root.Clusters[clusterName] = newClusterScopeSubTree(matched, nodeAttributesForCluster(cluster, detail))
	root.clusterIDToName[cluster.GetID()] = clusterName
}

// bubbleUpStatesAndCompactify updates the state of parent nodes based on the
// state of their children and compactifies the tree iff the requested level of
// detail is MINIMAL.
//
// If any child is Included or Partial, its parent becomes at least Partial. If
// all children are Included, the parent is still Partial unless it has been
// included directly.
//
// For MINIMAL level of detail, delete from the tree:
//   - subtrees *with roots* in the Excluded state,
//   - subtrees *of nodes* in the Included state.
func (root *ScopeTree) bubbleUpStatesAndCompactify(detail v1.ComputeEffectiveAccessScopeRequest_Detail) {
	deleteUnnecessaryNodes := detail == v1.ComputeEffectiveAccessScopeRequest_MINIMAL
	for clusterName, cluster := range root.Clusters {
		for namespaceName, namespace := range cluster.Namespaces {
			// Update the cluster's state from Excluded to Partial
			// if any of its namespaces is included.
			if cluster.State == Excluded &&
				(namespace.State == Included || namespace.State == Partial) {
				cluster.State = Partial

				// If we don't need to delete nodes, we can short-circuit.
				if !deleteUnnecessaryNodes {
					break
				}
			}
			// Delete Excluded namespaces if desired.
			if deleteUnnecessaryNodes && namespace.State == Excluded {
				delete(cluster.Namespaces, namespaceName)
			}
		}

		// Delete all namespaces for Included clusters and Excluded clusters
		// if desired.
		if deleteUnnecessaryNodes {
			if cluster.State == Included {
				cluster.Namespaces = nil
			} else if cluster.State == Excluded {
				delete(root.Clusters, clusterName)
			}
		}

		// Update the root's state from to Partial if any cluster is included.
		if root.State == Excluded && (cluster.State == Included || cluster.State == Partial) {
			root.State = Partial
		}
	}
}

// Merge adds scope tree to the current root so result tree includes nodes that are included at least in one of them.
// As we don't know in which form we get tree to merge with result will be in MINIMAL form
func (root *ScopeTree) Merge(tree *ScopeTree) {
	if tree == nil || tree == DenyAllEffectiveAccessScope() {
		return
	}
	if tree.State == Included || root.State == Included {
		root.State = Included
		return
	}
	if root.Clusters == nil {
		root.Clusters = map[string]*clustersScopeSubTree{}
	}
	if len(tree.clusterIDToName) > 0 && root.clusterIDToName == nil {
		root.clusterIDToName = make(map[string]string)
	}
	for clusterID, clusterName := range tree.clusterIDToName {
		root.clusterIDToName[clusterID] = clusterName
	}
	for key, cluster := range tree.Clusters {
		rootCluster := root.Clusters[key]
		if rootCluster == nil || cluster.State == Included {
			root.Clusters[key] = cluster.copy()
			continue
		}
		if rootCluster.State == Included || cluster.State == Excluded {
			continue
		}
		// partial
		for nsName, namespace := range cluster.Namespaces {
			if namespace.State == Included {
				if rootCluster.Namespaces == nil {
					rootCluster.Namespaces = map[string]*namespacesScopeSubTree{}
				}
				rootCluster.Namespaces[nsName] = namespace.copy()
			}
		}
	}
	root.bubbleUpStatesAndCompactify(v1.ComputeEffectiveAccessScopeRequest_MINIMAL)
}

func (cluster *clustersScopeSubTree) copy() *clustersScopeSubTree {
	namespaces := make(map[string]*namespacesScopeSubTree, len(cluster.Namespaces))
	for k, v := range cluster.Namespaces {
		namespaces[k] = v.copy()
	}
	return &clustersScopeSubTree{
		State:      cluster.State,
		Namespaces: namespaces,
		Attributes: *cluster.Attributes.copy(),
	}
}

// populateStateForNamespace adds given namespace as Included or Excluded to
// parent cluster. Only the last observed namespace is considered if multiple
// ones with the same <cluster name, namespace name> exist.
func (cluster *clustersScopeSubTree) populateStateForNamespace(namespace NamespaceForSAC, namespaceSelectors []labels.Selector, detail v1.ComputeEffectiveAccessScopeRequest_Detail) {
	clusterName := namespace.GetClusterName()
	namespaceName := namespace.GetName()
	namespaceFQSN := getNamespaceFQSN(clusterName, namespaceName)

	// If parent is Included, include the namespace and short-circuit:
	// no need to match if parent is included.
	if cluster.State == Included {
		cluster.Namespaces[namespaceName] = newNamespacesScopeSubTree(Included, nodeAttributesForNamespace(namespace, detail))
		return
	}

	// Augment namespace labels with namespace's FQSN.
	namespaceLabels := augmentLabels(namespace.GetLabels(), namespaceNameLabel, namespaceFQSN)

	// Match and update the tree.
	matched := matchLabels(namespaceSelectors, namespaceLabels)
	cluster.Namespaces[namespaceName] = newNamespacesScopeSubTree(matched, nodeAttributesForNamespace(namespace, detail))
}

func newEffectiveAccessScopeTree(state scopeState) *ScopeTree {
	return &ScopeTree{
		State:           state,
		Clusters:        make(map[string]*clustersScopeSubTree),
		clusterIDToName: make(map[string]string),
	}
}

func newClusterScopeSubTree(state scopeState, attributes treeNodeAttributes) *clustersScopeSubTree {
	return &clustersScopeSubTree{
		State:      state,
		Namespaces: make(map[string]*namespacesScopeSubTree),
		Attributes: attributes,
	}
}

func newNamespacesScopeSubTree(state scopeState, attributes treeNodeAttributes) *namespacesScopeSubTree {
	return &namespacesScopeSubTree{
		State:      state,
		Attributes: attributes,
	}
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
func matchLabels(selectors []labels.Selector, lbls map[string]string) scopeState {
	for _, s := range selectors {
		if s.Matches(labels.Set(lbls)) {
			return Included
		}
	}
	return Excluded
}
