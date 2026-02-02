package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

func main() {
	// Read CSV from stdin
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	// Parse YAML
	var doc map[string]interface{}
	if err := yaml.Unmarshal(input, &doc); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML: %v\n", err)
		os.Exit(1)
	}

	// Fix descriptors in all owned CRDs
	spec := doc["spec"].(map[string]interface{})
	crds := spec["customresourcedefinitions"].(map[string]interface{})
	owned := crds["owned"].([]interface{})

	for _, crd := range owned {
		crdMap := crd.(map[string]interface{})
		if specDescriptors, ok := crdMap["specDescriptors"].([]interface{}); ok {
			// Convert to []map[string]interface{}
			descriptors := make([]map[string]interface{}, len(specDescriptors))
			for i, d := range specDescriptors {
				descriptors[i] = d.(map[string]interface{})
			}

			fixDescriptorOrder(descriptors)
			allowRelativeFieldDependencies(descriptors)

			// No need to reassign, we modified in place
		}
	}

	// Marshal back to YAML
	output, err := yaml.Marshal(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling YAML: %v\n", err)
		os.Exit(1)
	}

	// Write to stdout
	fmt.Print(string(output))
}

// fixDescriptorOrder performs stable sort so parents appear before children
func fixDescriptorOrder(descriptors []map[string]interface{}) {
	sort.SliceStable(descriptors, func(i, j int) bool {
		pathI := descriptors[i]["path"].(string)
		pathJ := descriptors[j]["path"].(string)

		// Sort lexicographically - this ensures parents come before children
		// because "central" < "central.db" < "central.db.enabled"
		return pathI < pathJ
	})
}

// allowRelativeFieldDependencies converts relative field dependencies to absolute
func allowRelativeFieldDependencies(descriptors []map[string]interface{}) {
	for _, d := range descriptors {
		xDescriptors, ok := d["x-descriptors"].([]interface{})
		if !ok {
			continue
		}

		for i, xDesc := range xDescriptors {
			xDescStr, ok := xDesc.(string)
			if !ok {
				continue
			}

			// Check if it's a fieldDependency descriptor
			if !strings.Contains(xDescStr, "urn:alm:descriptor:com.tectonic.ui:fieldDependency:") {
				continue
			}

			// Split to extract field and value
			parts := strings.Split(xDescStr, ":")
			if len(parts) < 7 {
				continue
			}

			field := parts[5]
			value := parts[6]

			// If field starts with '.', it's relative
			if !strings.HasPrefix(field, ".") {
				continue
			}

			// Resolve relative to current path
			currentPath := "." + d["path"].(string)
			parentPath := currentPath[:strings.LastIndex(currentPath, ".")]
			absoluteField := strings.TrimPrefix(parentPath, ".") + field

			// Reconstruct descriptor with absolute path
			xDescriptors[i] = fmt.Sprintf("urn:alm:descriptor:com.tectonic.ui:fieldDependency:%s:%s",
				absoluteField, value)
		}
	}
}
