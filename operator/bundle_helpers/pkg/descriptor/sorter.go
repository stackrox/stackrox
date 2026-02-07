package descriptor

import (
	"fmt"
	"sort"
	"strings"
)

// FixCSVDescriptorsMap processes all CRDs in a CSV and fixes their specDescriptors.
// This function works with map[string]interface{} to match Python's behavior.
func FixCSVDescriptorsMap(csvDoc map[string]interface{}) error {
	// Navigate to spec.customresourcedefinitions.owned
	spec, ok := csvDoc["spec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("spec not found or not a map")
	}

	crds, ok := spec["customresourcedefinitions"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("customresourcedefinitions not found or not a map")
	}

	owned, ok := crds["owned"].([]interface{})
	if !ok {
		return fmt.Errorf("owned not found or not a list")
	}

	// Process each CRD
	for _, crdItem := range owned {
		crd, ok := crdItem.(map[string]interface{})
		if !ok {
			continue
		}

		if err := processSpecDescriptorsMap(crd); err != nil {
			return err
		}
	}

	return nil
}

// processSpecDescriptorsMap processes specDescriptors for a single CRD.
func processSpecDescriptorsMap(crd map[string]interface{}) error {
	descs, ok := crd["specDescriptors"]
	if !ok {
		// No specDescriptors, that's OK
		return nil
	}

	descriptors, ok := descs.([]interface{})
	if !ok {
		return fmt.Errorf("specDescriptors is not a list")
	}

	// Fix descriptor order
	fixDescriptorOrderMap(descriptors)

	// Allow relative field dependencies
	allowRelativeFieldDependenciesMap(descriptors)

	return nil
}

// fixDescriptorOrderMap performs a stable sort based on the parent path.
// This ensures children always come after their parents.
// Mimics Python: descriptors.sort(key=lambda d: f'.{d["path"]}'.rsplit('.', 1)[0])
func fixDescriptorOrderMap(descriptors []interface{}) {
	sort.SliceStable(descriptors, func(i, j int) bool {
		pathI := getDescriptorPathMap(descriptors[i])
		pathJ := getDescriptorPathMap(descriptors[j])
		parentI := getParentPath(pathI)
		parentJ := getParentPath(pathJ)
		return parentI < parentJ
	})
}

// getParentPath extracts the parent path from a descriptor path.
// Mimics Python: f'.{d["path"]}'.rsplit('.', 1)[0]
func getParentPath(path string) string {
	// Add a '.' in front for simplicity
	fullPath := "." + path
	// Split by last '.' and take the first part
	lastDot := strings.LastIndex(fullPath, ".")
	if lastDot == -1 {
		return ""
	}
	return fullPath[:lastDot]
}

// getDescriptorPathMap extracts the 'path' field from a descriptor map.
func getDescriptorPathMap(desc interface{}) string {
	descMap, ok := desc.(map[string]interface{})
	if !ok {
		return ""
	}

	path, ok := descMap["path"].(string)
	if !ok {
		return ""
	}

	return path
}

// allowRelativeFieldDependenciesMap converts relative field dependency paths to absolute.
func allowRelativeFieldDependenciesMap(descriptors []interface{}) {
	for _, desc := range descriptors {
		descMap, ok := desc.(map[string]interface{})
		if !ok {
			continue
		}

		path, _ := descMap["path"].(string)
		xDescsRaw, ok := descMap["x-descriptors"]
		if !ok {
			continue
		}

		xDescs, ok := xDescsRaw.([]interface{})
		if !ok {
			continue
		}

		// Process each x-descriptor
		for i, xDescRaw := range xDescs {
			xDesc, ok := xDescRaw.(string)
			if !ok {
				continue
			}

			if !strings.HasPrefix(xDesc, "urn:alm:descriptor:com.tectonic.ui:fieldDependency:") {
				continue
			}

			// Split by ':' and get the last two parts (field and value)
			parts := strings.Split(xDesc, ":")
			if len(parts) < 2 {
				continue
			}

			field := parts[len(parts)-2]
			val := parts[len(parts)-1]

			// Check if field starts with '.' (relative path)
			if !strings.HasPrefix(field, ".") {
				continue
			}

			// Convert relative to absolute
			// Mimics Python: f'.{d["path"]}'.rsplit('.', 1)[0][1:] + field
			parentPath := getParentPath(path)
			if len(parentPath) > 0 {
				parentPath = parentPath[1:] // Remove leading '.'
			}
			absoluteField := parentPath + field

			// Reconstruct the x-descriptor
			prefix := "urn:alm:descriptor:com.tectonic.ui:fieldDependency:"
			xDescs[i] = prefix + absoluteField + ":" + val
		}
	}
}
