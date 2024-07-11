package fixtures

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReferences(t *testing.T) {
	refs, err := References()
	assert.NoError(t, err)
	assert.Equal(t, 2176, len(refs))
}
