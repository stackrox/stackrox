package set

// TODO: remove these shortcuts to comonly used types, made to avoid huge diff.

// StringSet is a set of strings.
type StringSet = Set[string]

// NewStringSet creates an initialized set of strings.
func NewStringSet(initial ...string) Set[string] {
	return NewSet(initial...)
}

// FrozenStringSet is a frozen set of strings.
type FrozenStringSet = FrozenSet[string]

// NewFrozenStringSet creates an initialized frozen set of strings.
func NewFrozenStringSet(initial ...string) FrozenSet[string] {
	return NewFrozenSet(initial...)
}

// IntSet is a set of ints.
type IntSet = Set[int]

// NewIntSet creates an initialized set of ints.
func NewIntSet(initial ...int) Set[int] {
	return NewSet(initial...)
}

// FrozenIntSet is a frozen set of ints.
type FrozenIntSet = FrozenSet[int]

// NewFrozenIntSet creates an initialized frozen set of ints.
func NewFrozenIntSet(initial ...int) FrozenSet[int] {
	return NewFrozenSet(initial...)
}
