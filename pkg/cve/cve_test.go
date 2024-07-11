package cve

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestCVETypesAreAccountedFor(t *testing.T) {
	// + 1 for unknown type
	assert.Equal(t, len(storage.CVE_CVEType_name), len(clusterCVETypes)+len(componentCVETypes)+1)
}
