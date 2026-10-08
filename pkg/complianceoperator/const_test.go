package complianceoperator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultNodeRoles(t *testing.T) {
	assert.Equal(t, []string{"master", "worker"}, DefaultNodeRoles())
}
