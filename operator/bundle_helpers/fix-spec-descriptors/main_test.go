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

	assert.Equal(t, []string{
		"central",
		"central.db",
		"central.db.enabled",
		"scanner",
		"scanner.enabled",
	}, paths)
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
