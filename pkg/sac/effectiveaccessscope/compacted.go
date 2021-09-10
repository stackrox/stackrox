package effectiveaccessscope

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

// ScopeTreeCompacted is a compacted, JSON-ready representation of a ScopeTree.
// Cluster name -> sorted list of included namespace names.
type ScopeTreeCompacted map[string][]string

// String converts ScopeTreeCompacted to a one-line string.
func (c ScopeTreeCompacted) String() string {
	clusterStrs := make([]string, 0, len(c))

	for clusterName, namespaces := range c {
		var result strings.Builder
		result.WriteString(clusterName)
		result.WriteString(scopeSeparator)

		switch len(namespaces) {
		case 0:
			continue
		case 1:
			result.WriteString(namespaces[0])
		default:
			result.WriteString("{")
			result.WriteString(strings.Join(namespaces, ", "))
			result.WriteString("}")
		}

		clusterStrs = append(clusterStrs, result.String())
	}

	// Ensure order consistency across invocations.
	sort.Slice(clusterStrs, func(i, j int) bool {
		return clusterStrs[i] < clusterStrs[j]
	})

	return strings.Join(clusterStrs, ", ")
}

// ToJSON converts ScopeTreeCompacted to JSON string.
func (c ScopeTreeCompacted) ToJSON() (string, error) {
	jsonified, err := json.Marshal(c)
	if err != nil {
		return "", errors.Wrap(err, "converting compacted scope tree to JSON")
	}
	return string(jsonified), nil
}
