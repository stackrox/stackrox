package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func metaObj(uid string, parents ...string) metav1.Object {
	parentRefs := make([]metav1.OwnerReference, 0, len(parents))
	for _, p := range parents {
		parentRefs = append(parentRefs, metav1.OwnerReference{
			UID:        types.UID(p),
			Kind:       "Deployment",
			APIVersion: "v1",
		})
	}
	return &metav1.ObjectMeta{
		UID:             types.UID(uid),
		OwnerReferences: parentRefs,
	}
}

func TestParentHierarchy(t *testing.T) {
	hierarchy := NewParentHierarchy()

	// Empty is false
	assert.False(t, hierarchy.IsValidChild("C", metaObj("D")))

	// Test single hop parent
	dMetaObj := metaObj("D", "C")
	hierarchy.Add(metaObj("D", "C"))
	assert.True(t, hierarchy.IsValidChild("C", dMetaObj))
	assert.ElementsMatch(t, hierarchy.TopLevelParents("D").AsSlice(), []string{"C"})

	// Test multiple hops
	hierarchy.Add(metaObj("C", "B"))
	assert.ElementsMatch(t, hierarchy.TopLevelParents("D").AsSlice(), []string{"B"})
	assert.True(t, hierarchy.IsValidChild("B", dMetaObj))

	// Test multiple parents
	hierarchy.Add(metaObj("D", "C", "A"))
	dMetaObj = metaObj("D", "C", "A")
	assert.True(t, hierarchy.IsValidChild("B", dMetaObj))
	assert.True(t, hierarchy.IsValidChild("A", dMetaObj))
	assert.ElementsMatch(t, hierarchy.TopLevelParents("D").AsSlice(), []string{"A", "B"})

	// Remove a middle parent
	hierarchy.Remove("C")
	assert.False(t, hierarchy.IsValidChild("B", dMetaObj))
	assert.True(t, hierarchy.IsValidChild("A", dMetaObj))
	assert.ElementsMatch(t, hierarchy.TopLevelParents("D").AsSlice(), []string{"A", "C"})
}
