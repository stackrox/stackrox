package pathutil

import (
	"strconv"
	"strings"
)

// A Step represents a step in a path.
// It is either a field traversal (if Index() returns < 0),
// or a slice index (if Index() returns >= 0).
type Step interface {
	Field() string
	Index() int
}

// FieldStep returns a step that represents a field traversal.
type FieldStep string

// Field implements Step.
func (f FieldStep) Field() string {
	return string(f)
}

// Index implements Step.
func (f FieldStep) Index() int {
	return -1
}

// IndexStep returns a step that indexes into a slice with the given index.
type IndexStep int

// Field implements Step.
func (i IndexStep) Field() string {
	return ""
}

// Index implements Step.
func (i IndexStep) Index() int {
	return int(i)
}

// A Path represents a list of steps taken to traverse an object.
// This includes struct field indirections and array indexing.
type Path struct {
	steps []Step
}

// NewPath creates a path with the given steps.
func NewPath(steps ...Step) *Path {
	return &Path{steps: steps}
}

func (p *Path) String() string {
	keys := make([]string, len(p.steps))
	for idx, step := range p.steps {
		if stepIndex := step.Index(); stepIndex >= 0 {
			keys[idx] = strconv.Itoa(stepIndex)
		} else {
			keys[idx] = step.Field()
		}
	}
	return strings.Join(keys, ".")
}
