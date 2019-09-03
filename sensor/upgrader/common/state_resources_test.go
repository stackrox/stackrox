package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// TestEnsureStateResourceTypesAreCorrect exists to make accidental modifications of StateResourceTypes less likely.
// There is of course no protection against somebody intentionally modifying both lists.
func TestEnsureStateResourceTypesAreCorrect(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(t, StateResourceTypes, []schema.GroupVersionKind{
		{Version: "v1", Kind: "Secret"},
	})
}
