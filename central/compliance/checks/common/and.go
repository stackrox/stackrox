package common

// Andable is a function that returns true or false.
type Andable func() bool

// Ander allows the execution of multiple Andables and returns the result of anding the individual results.
type Ander interface {
	Execute() bool
}

// NewAnder returns an Ander
func NewAnder(andables ...Andable) Ander {
	return &and{
		andables: andables,
	}
}

type and struct {
	andables []Andable
}

func (a *and) Execute() bool {
	for _, toAnd := range a.andables {
		if !toAnd() {
			return false
		}
	}
	return true
}
