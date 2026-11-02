package indexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForceOption(t *testing.T) {
	assert.False(t, makeOptions().force, "force defaults to false")
	assert.True(t, makeOptions(Force(true)).force)
	assert.False(t, makeOptions(Force(true), Force(false)).force, "last Force option wins")
}
