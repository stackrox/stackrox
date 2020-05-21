package pathutil

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/pkg/pointers"
)

// stepMapKey represents a "hash" of a step, which is comparable
// and can be used as a map key.
type stepMapKey interface{}

type step struct {
	field string
	index *int
}

func (s *step) mapKey() stepMapKey {
	if s.index != nil {
		return *s.index
	}
	return s.field
}

func stepFromMapKey(key stepMapKey) step {
	if asInt, ok := key.(int); ok {
		return step{index: pointers.Int(asInt)}
	}
	return step{field: key.(string)}
}

// A Path represents a list of steps taken to traverse an object.
// This includes struct field indirections and array indexing.
type Path struct {
	steps []step
}

// TraverseField adds a new step to Path that traverses a struct field.
func (p *Path) TraverseField(fieldName string) *Path {
	p.steps = append(p.steps, step{field: fieldName})
	return p
}

// IndexSlice adds a new step to Path that indexes into a slice.
func (p *Path) IndexSlice(index int) *Path {
	p.steps = append(p.steps, step{index: pointers.Int(index)})
	return p
}

func (p *Path) String() string {
	keys := make([]string, len(p.steps))
	for idx, step := range p.steps {
		keys[idx] = fmt.Sprint(step.mapKey())
	}
	return strings.Join(keys, ".")
}
