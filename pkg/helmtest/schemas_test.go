package helmtest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchemas(t *testing.T) {
	requiredSchemas := []string{
		"kubernetes-1.20.2",
		"openshift-3.11.0",
		"openshift-4.1.0",
	}

	for _, schemaName := range requiredSchemas {
		_, err := getSchema(schemaName)
		assert.NoErrorf(t, err, "failed to load required schema %s", schemaName)
	}
}
