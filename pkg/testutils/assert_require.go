package testutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertRequire returns assert and require objects for the given test.
func AssertRequire(t *testing.T) (*assert.Assertions, *require.Assertions) {
	return assert.New(t), require.New(t)
}
