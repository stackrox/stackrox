package mathutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMod(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 3, Mod(7, 4))
	assert.Equal(t, 3, Mod(-5, 4))
	assert.Equal(t, 3, Mod(7, -4))
	assert.Equal(t, 3, Mod(-1, -4))
}
