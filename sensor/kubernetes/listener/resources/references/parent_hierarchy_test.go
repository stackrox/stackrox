package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParentHierarchy(t *testing.T) {
	hierarchy := NewParentHierarchy()

	// Empty is false
	assert.False(t, hierarchy.IsValidChild("C", "D"))

	// Test single hop parent
	hierarchy.Add([]string{"C"}, "D")
	assert.True(t, hierarchy.IsValidChild("C", "D"))

	// Test multiple hops
	hierarchy.Add([]string{"B"}, "C")
	assert.True(t, hierarchy.IsValidChild("B", "D"))

	// Test multiple parents
	hierarchy.Add([]string{"C", "A"}, "D")
	assert.True(t, hierarchy.IsValidChild("B", "D"))
	assert.True(t, hierarchy.IsValidChild("A", "D"))

	// Remove a middle parent
	hierarchy.Remove("C")
	assert.False(t, hierarchy.IsValidChild("B", "D"))
	assert.True(t, hierarchy.IsValidChild("A", "D"))
}
