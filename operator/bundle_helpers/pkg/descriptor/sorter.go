package descriptor

import (
	"sort"
	"strings"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chartutil"
)

// FixCSVDescriptorsMap processes all CRDs in a CSV and fixes their specDescriptors.
func FixCSVDescriptorsMap(csvDoc chartutil.Values) error {
	// Navigate to spec.customresourcedefinitions.owned
	ownedRaw, err := csvDoc.PathValue("spec.customresourcedefinitions.owned")
	if err != nil {
		return errors.Wrap(err, "spec.customresourcedefinitions.owned")
	}

	owned, ok := ownedRaw.([]any)
	if !ok {
		return errors.New("spec.customresourcedefinitions.owned is not a list")
	}

	// Process each CRD
	for _, crdItem := range owned {
		crd, ok := crdItem.(map[string]any)
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
func processSpecDescriptorsMap(crd map[string]any) error {
	descs, ok := crd["specDescriptors"]
	if !ok {
		// No specDescriptors, that's OK
		return nil
	}

	descriptors, ok := descs.([]any)
	if !ok {
		return errors.New("specDescriptors is not a list")
	}

	fixDescriptorOrderMap(descriptors)
	allowRelativeFieldDependenciesMap(descriptors)

	return nil
}

// fixDescriptorOrderMap performs a stable sort based on the parent path.
// This ensures children always come after their parents.
// Mimics Python: descriptors.sort(key=lambda d: f'.{d["path"]}'.rsplit('.', 1)[0])
func fixDescriptorOrderMap(descriptors []any) {
	sort.SliceStable(descriptors, func(i, j int) bool {
		pathI := getDescriptorPathMap(descriptors[i])
		pathJ := getDescriptorPathMap(descriptors[j])
		parentI := getParentPath(pathI)
		parentJ := getParentPath(pathJ)
		return parentI < parentJ
	})
}

// getParentPath extracts the parent path from a descriptor path.
// Returns everything before the last '.', or empty string if no '.' exists.
func getParentPath(path string) string {
	lastDot := strings.LastIndex(path, ".")
	if lastDot == -1 {
		return ""
	}
	return path[:lastDot]
}

// getDescriptorPathMap extracts the 'path' field from a descriptor map.
func getDescriptorPathMap(desc any) string {
	descMap, ok := desc.(map[string]any)
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
func allowRelativeFieldDependenciesMap(descriptors []any) {
	for _, desc := range descriptors {
		descMap, ok := desc.(map[string]any)
		if !ok {
			continue
		}

		path, _ := descMap["path"].(string)
		xDescsRaw, ok := descMap["x-descriptors"]
		if !ok {
			continue
		}

		xDescs, ok := xDescsRaw.([]any)
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
			// Get parent path and concatenate with relative field
			parentPath := getParentPath(path)
			absoluteField := parentPath + field

			// Reconstruct the x-descriptor
			prefix := "urn:alm:descriptor:com.tectonic.ui:fieldDependency:"
			xDescs[i] = prefix + absoluteField + ":" + val
		}
	}
}
