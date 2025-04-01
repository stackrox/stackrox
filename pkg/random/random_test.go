package random

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateString(t *testing.T) {
	assert.Empty(t, GenerateString(0, AlphanumericCharacters))
	assert.Empty(t, GenerateString(0, ""))
	assert.Empty(t, GenerateString(10, ""))
	assert.Equal(t, GenerateString(3, "0"), "000")
}
