package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFixDescriptorOrder(t *testing.T) {
	descriptors := []map[string]interface{}{
		{"path": "central.db.enabled"},
		{"path": "central"},
		{"path": "central.db"},
		{"path": "scanner.enabled"},
		{"path": "scanner"},
	}

	fixDescriptorOrder(descriptors)

	// Verify order: parents before children
	paths := make([]string, len(descriptors))
	for i, d := range descriptors {
		paths[i] = d["path"].(string)
	}

	// Expected order with parent-path sorting (matches Python behavior):
	// - Items with same parent path maintain their original relative order
	// - "central" and "scanner" both have empty parent, so maintain original order: central, scanner
	// - "central.db" has parent ".central", comes after top-level items
	// - "central.db.enabled" has parent ".central.db", comes after "central.db"
	// - "scanner.enabled" has parent ".scanner", comes after "central.db.*" (lexicographic)
	assert.Equal(t, []string{
		"central",
		"scanner",
		"central.db",
		"central.db.enabled",
		"scanner.enabled",
	}, paths)
}

func TestFixDescriptorOrderPreservesSiblingOrder(t *testing.T) {
	// Test that siblings (items with same parent) maintain their original relative order
	descriptors := []map[string]interface{}{
		{"path": "scanner.resources"},        // Parent: ".scanner"
		{"path": "scanner.db"},               // Parent: ".scanner"
		{"path": "scanner.db.enabled"},       // Parent: ".scanner.db"
		{"path": "scanner.resources.limits"}, // Parent: ".scanner.resources"
	}

	fixDescriptorOrder(descriptors)

	paths := make([]string, len(descriptors))
	for i, d := range descriptors {
		paths[i] = d["path"].(string)
	}

	// With lexicographic sort, "scanner.db" would come before "scanner.resources"
	// With parent-path sort (Python behavior), siblings maintain original order
	assert.Equal(t, []string{
		"scanner.resources",        // First sibling (parent: ".scanner")
		"scanner.db",               // Second sibling (parent: ".scanner")
		"scanner.db.enabled",       // Child of "scanner.db"
		"scanner.resources.limits", // Child of "scanner.resources"
	}, paths, "Siblings with same parent should maintain original relative order")
}

func TestAllowRelativeFieldDependencies(t *testing.T) {
	descriptors := []map[string]interface{}{
		{
			"path": "central.db.passwordSecret",
			"x-descriptors": []interface{}{
				"urn:alm:descriptor:com.tectonic.ui:fieldDependency:.enabled:true",
			},
		},
		{
			"path": "central.db.enabled",
		},
	}

	allowRelativeFieldDependencies(descriptors)

	xDescs := descriptors[0]["x-descriptors"].([]interface{})
	assert.Equal(t,
		"urn:alm:descriptor:com.tectonic.ui:fieldDependency:central.db.enabled:true",
		xDescs[0])
}
