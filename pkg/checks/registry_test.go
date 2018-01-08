package checks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistrySize(t *testing.T) {
	assert.Equal(t, len(checkCreators), len(Registry))
}
