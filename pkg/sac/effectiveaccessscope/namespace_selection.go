package effectiveaccessscope

import (
	"iter"
	"maps"
	"slices"
)

// NamespaceAllowed checks a name against the namespace grants in this cluster.
// Exclusion maps are bounded by the inventory captured when the scope was built.
func (cluster *clustersScopeSubTree) NamespaceAllowed(name string) bool {
	if cluster.State == Included {
		return true
	}
	if namespace := cluster.Namespaces[name]; namespace != nil {
		return namespace.State == Included
	}
	if cluster.namespaceDefault != Included {
		return false
	}
	_, known := cluster.namespaceUniverse[name]
	return known
}

// GetNamespaceByName returns namespace metadata, including implicit grants in
// an exclusion map. It returns nil for unknown names.
func (cluster *clustersScopeSubTree) GetNamespaceByName(name string) *namespacesScopeSubTree {
	if namespace := cluster.Namespaces[name]; namespace != nil {
		return namespace
	}
	if cluster.namespaceDefault == Included {
		if id, known := cluster.namespaceUniverse[name]; known {
			return newNamespacesScopeSubTree(Included, treeNodeAttributes{ID: id})
		}
	}
	return nil
}

// AllowedNamespaceNames iterates over the namespace names granted explicitly or
// implicitly by a partial cluster. A full cluster grant has no namespace list.
func (cluster *clustersScopeSubTree) AllowedNamespaceNames() iter.Seq[string] {
	return func(yield func(string) bool) {
		if cluster.namespaceDefault == Included {
			for name := range cluster.namespaceUniverse {
				if cluster.NamespaceAllowed(name) && !yield(name) {
					return
				}
			}
			return
		}
		for name, namespace := range cluster.Namespaces {
			if namespace.State == Included && !yield(name) {
				return
			}
		}
	}
}

// NamespaceFilterNames returns the names to include and the exceptions to
// exclude in a search filter. The inclusion list bounds exclusion grants to
// this request's inventory, so searches and point checks agree on unknown names.
func (cluster *clustersScopeSubTree) NamespaceFilterNames() (included, excluded []string) {
	if cluster.namespaceDefault != Included {
		return slices.Collect(cluster.AllowedNamespaceNames()), nil
	}
	included = slices.Collect(maps.Keys(cluster.namespaceUniverse))
	for name, namespace := range cluster.Namespaces {
		if namespace.State == Excluded {
			excluded = append(excluded, name)
		}
	}
	return included, excluded
}

// HasNamespaceAccess reports whether any namespace is accessible in this cluster.
func (cluster *clustersScopeSubTree) HasNamespaceAccess() bool {
	if cluster.State == Included {
		return true
	}
	for range cluster.AllowedNamespaceNames() {
		return true
	}
	return false
}

func namespaceSelection(group MatchedNamespaceGroup) *clustersScopeSubTree {
	cluster := &clustersScopeSubTree{}
	if !group.Complete {
		cluster.Namespaces = make(map[string]*namespacesScopeSubTree, len(group.Namespaces))
		for _, namespace := range group.Namespaces {
			if existing := cluster.Namespaces[namespace.Name]; existing != nil {
				// Choose the smallest matching ID for duplicate recorded names.
				existing.Attributes.ID = min(existing.Attributes.ID, namespace.ID)
			} else {
				cluster.Namespaces[namespace.Name] = newNamespacesScopeSubTree(Included, treeNodeAttributes{ID: namespace.ID})
			}
		}
		return cluster
	}

	cluster.namespaceDefault = Included
	cluster.namespaceUniverse = make(map[string]string, len(group.Namespaces)+len(group.ExcludedNamespaces))
	cluster.Namespaces = make(map[string]*namespacesScopeSubTree, len(group.ExcludedNamespaces))
	// Process matches first: any matching row grants a recorded name, even if
	// another row with that name has excluded labels.
	for _, namespace := range group.Namespaces {
		id, known := cluster.namespaceUniverse[namespace.Name]
		if !known || namespace.ID < id {
			cluster.namespaceUniverse[namespace.Name] = namespace.ID
		}
	}
	for _, namespace := range group.ExcludedNamespaces {
		if existing := cluster.Namespaces[namespace.Name]; existing != nil {
			existing.Attributes.ID = min(existing.Attributes.ID, namespace.ID)
			cluster.namespaceUniverse[namespace.Name] = existing.Attributes.ID
		} else if _, known := cluster.namespaceUniverse[namespace.Name]; !known {
			cluster.namespaceUniverse[namespace.Name] = namespace.ID
			cluster.Namespaces[namespace.Name] = newNamespacesScopeSubTree(Excluded, treeNodeAttributes{ID: namespace.ID})
		}
	}
	// Duplicate names can change the density after SQL has counted rows.
	if len(cluster.Namespaces)*2 >= len(cluster.namespaceUniverse) {
		included := make(map[string]*namespacesScopeSubTree, len(cluster.namespaceUniverse)-len(cluster.Namespaces))
		for name, id := range cluster.namespaceUniverse {
			if cluster.Namespaces[name] == nil {
				included[name] = newNamespacesScopeSubTree(Included, treeNodeAttributes{ID: id})
			}
		}
		cluster.Namespaces = included
		cluster.namespaceDefault = Excluded
		cluster.namespaceUniverse = nil
	}
	return cluster
}

// mergeNamespaces unions grants. An exclusion in one role never overrides an
// inclusion in another, and inventories can differ between scope computations.
func (cluster *clustersScopeSubTree) mergeNamespaces(other *clustersScopeSubTree) {
	if cluster.namespaceDefault != Included && other.namespaceDefault != Included {
		for name, namespace := range other.Namespaces {
			if namespace.State == Included {
				if cluster.Namespaces == nil {
					cluster.Namespaces = make(map[string]*namespacesScopeSubTree)
				}
				cluster.Namespaces[name] = namespace.copy()
			}
		}
		return
	}
	universe := maps.Clone(cluster.namespaceUniverse)
	if universe == nil {
		universe = make(map[string]string, len(cluster.Namespaces)+len(other.namespaceUniverse))
	}
	for name, namespace := range cluster.Namespaces {
		if namespace.State == Included {
			universe[name] = namespace.Attributes.ID
		}
	}
	for name, id := range other.namespaceUniverse {
		if _, known := universe[name]; !known || other.NamespaceAllowed(name) {
			universe[name] = id
		}
	}
	for name, namespace := range other.Namespaces {
		if namespace.State == Included {
			universe[name] = namespace.Attributes.ID
		}
	}
	namespaces := make(map[string]*namespacesScopeSubTree)
	for name, id := range universe {
		left, right := cluster.NamespaceAllowed(name), other.NamespaceAllowed(name)
		if !left && !right {
			namespaces[name] = newNamespacesScopeSubTree(Excluded, treeNodeAttributes{ID: id})
			continue
		}
		// Preserve enriched attributes if a caller merges a detailed tree.
		selected := cluster.Namespaces[name]
		if right {
			selected = other.Namespaces[name]
		}
		if selected != nil && (selected.Attributes.Name != "" || len(selected.Attributes.Labels) > 0) {
			namespaces[name] = selected.copy()
		}
	}
	cluster.namespaceDefault = Included
	cluster.namespaceUniverse = universe
	cluster.Namespaces = namespaces
}
