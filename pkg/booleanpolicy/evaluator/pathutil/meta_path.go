package pathutil

import (
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// A MetaStep represents a step in a MetaPath. It is either a struct field
// traversal (through StructFieldIndex), or a leap through an augmented object.
type MetaStep struct {
	Type             reflect.Type
	FieldName        string
	StructFieldIndex []int // This is reflect.StructField.Index, which is an efficient way to index into a struct.
}

// A MetaPath represents a path on types
// (ie, a MetaPath can be thought of the abstract version of a Path).
// Whereas a Path operates on an _instance_ of an object,
// a MetaPath operates on the type.
type MetaPath []MetaStep

type metaPathAndMetadata struct {
	metaPath     MetaPath
	preferParent bool
}

// FieldToMetaPathMap helps store and retrieve meta paths given a field tag.
type FieldToMetaPathMap struct {
	underlying map[string]metaPathAndMetadata
}

func (m *FieldToMetaPathMap) add(tag string, metaPath MetaPath, shouldPreferParent bool) error {
	lowerTag := strings.ToLower(tag)
	if existingPath, exists := m.underlying[lowerTag]; exists {
		// Neither of these tells you to prefer a parent!
		if !shouldPreferParent && !existingPath.preferParent {
			return errors.Errorf("duplicate search tag detected: %s (clashing paths: %v/%v)", tag, existingPath, metaPath)
		}
		// Defer to the other one, don't overwrite.
		if shouldPreferParent {
			return nil
		}
	}
	m.underlying[lowerTag] = metaPathAndMetadata{preferParent: shouldPreferParent, metaPath: metaPath}
	return nil
}

// Get returns the MetaPath for the given tag, and a bool indicates whether it exists.
func (m *FieldToMetaPathMap) Get(tag string) (MetaPath, bool) {
	metaPath, found := m.underlying[strings.ToLower(tag)]
	if found {
		return metaPath.metaPath, true
	}
	return nil, false
}
