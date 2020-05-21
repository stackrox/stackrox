package pathutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// PathFromSteps returns a path based on a list of steps.
// It's a convenience function for testing.
func PathFromSteps(t *testing.T, steps ...interface{}) *Path {
	p := &Path{}
	for _, step := range steps {
		switch typedStep := step.(type) {
		case int:
			p = p.IndexSlice(typedStep)
		case string:
			p = p.TraverseField(typedStep)
		default:
			require.FailNow(t, "invalid type of component", step)
		}
	}
	return p
}
