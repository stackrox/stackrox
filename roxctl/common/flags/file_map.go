package flags

import (
	"fmt"
	"path/filepath"
	"strings"
)

// FileMapVar can be used for a flag that takes multiple arguments (either comma-separated or in multiple flag usages)
// of form `[file-key=]file-path`, where `file-key`, if omitted, is assumed to be the basename of `file-path`.
type FileMapVar struct {
	FileMap *map[string]string
	changed bool
}

// Type implements the value interface.
func (FileMapVar) Type() string {
	return "fileMap"
}

// String implements the value interface
func (v FileMapVar) String() string {
	if len(*v.FileMap) == 0 {
		return ""
	}
	mappings := make([]string, 0, len(*v.FileMap))
	for k, v := range *v.FileMap {
		mappings = append(mappings, fmt.Sprintf("%s=%s", k, v))
	}

	return strings.Join(mappings, ",")
}

// Set implements the value interface.
func (v *FileMapVar) Set(val string) error {
	mappings := strings.Split(val, ",")

	for _, mapping := range mappings {
		mappingParts := strings.SplitN(mapping, "=", 2)
		if len(mappingParts) == 0 {
			continue
		}
		if len(mappingParts) == 1 {
			mappingParts = []string{filepath.Base(mappingParts[0]), mappingParts[0]}
		}
		for i, part := range mappingParts {
			mappingParts[i] = strings.TrimSpace(part)
		}

		if !v.changed {
			*v.FileMap = make(map[string]string)
			v.changed = true
		}
		(*v.FileMap)[mappingParts[0]] = mappingParts[1]
	}
	return nil
}
