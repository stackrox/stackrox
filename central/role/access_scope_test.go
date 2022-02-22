package role

import (
	"testing"

	"gotest.tools/assert"
)

func TestGenerateAccessScopeID(t *testing.T) {
	id := GenerateAccessScopeID()
	validID := EnsureValidAccessScopeID(id)
	assert.Equal(t, id, validID)
}
