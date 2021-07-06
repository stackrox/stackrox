package idmap

// NamespaceInfo contains information about namespaces - name, ID, cluster name, cluster ID
type NamespaceInfo struct {
	ID, Name               string
	ClusterID, ClusterName string
}

// IDMap is a read-only map allowing fast lookup of object properties by ID.
type IDMap struct {
	byNamespaceID map[string]*NamespaceInfo
}

// NewIDMap creates and returns a new IDMap.
func NewIDMap() *IDMap {
	return &IDMap{
		byNamespaceID: make(map[string]*NamespaceInfo),
	}
}

// ByNamespaceID looks up information about the namespace with the given ID.
func (m *IDMap) ByNamespaceID(namespaceID string) *NamespaceInfo {
	return m.byNamespaceID[namespaceID]
}

// Clone returns a copy of this ID map.
// Note: The individual entry pointers are not cloned, as they are never allowed to be modified anyway.
func (m *IDMap) Clone() *IDMap {
	byNamespaceID := make(map[string]*NamespaceInfo, len(m.byNamespaceID))
	for k, v := range m.byNamespaceID {
		byNamespaceID[k] = v
	}
	return &IDMap{
		byNamespaceID: byNamespaceID,
	}
}
