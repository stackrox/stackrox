package namespaces

const (
	// NamespaceNameLabel allows selecting a namespace by its name.
	NamespaceNameLabel = `namespace.metadata.stackrox.io/name`
)

var (
	validNamespaceNameLabelKeys = []string{
		"kubernetes.io/metadata.name",
		"name",
		NamespaceNameLabel,
	}
)

// GetFirstValidNamespaceNameLabelKey takes in namespace labels and the namespace name and returns the first valid key
func GetFirstValidNamespaceNameLabelKey(labels map[string]string, name string) string {
	for _, validKey := range validNamespaceNameLabelKeys {
		if labels[validKey] == name {
			return validKey
		}
	}
	return ""
}
