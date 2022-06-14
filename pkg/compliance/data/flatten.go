package data

import "github.com/stackrox/stackrox/generated/internalapi/compliance"

// FlattenFileMap takes a map of file paths to File objects and returns a map with all recursively nested Files in a single top level map of path to File.
func FlattenFileMap(toFlatten map[string]*compliance.File) map[string]*compliance.File {
	totalNodeFiles := make(map[string]*compliance.File)
	for path, file := range toFlatten {
		expanded := expandFile(file)
		for k, v := range expanded {
			totalNodeFiles[k] = v
		}
		totalNodeFiles[path] = file
	}
	return totalNodeFiles
}

func expandFile(parent *compliance.File) map[string]*compliance.File {
	expanded := make(map[string]*compliance.File)
	for _, child := range parent.GetChildren() {
		childExpanded := expandFile(child)
		for k, v := range childExpanded {
			expanded[k] = v
		}
	}
	expanded[parent.GetPath()] = parent
	return expanded
}
