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
			p.steps = append(p.steps, IndexStep(typedStep))
		case string:
			p.steps = append(p.steps, FieldStep(typedStep))
		default:
			require.FailNow(t, "invalid type of component", step)
		}
	}
	return p
}
