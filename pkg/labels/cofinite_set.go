package labels

import "github.com/stackrox/rox/pkg/set"

// cofiniteSet is a set that is either finite, or its complement is finite. It is realized as a set of values and
// a flag specifying whether this is the set of values that *are* or *are not* elements of the set.
type cofiniteSet struct {
	values set.StringSet
	invert bool
}

func makeCofiniteSet(invert bool, values ...string) cofiniteSet {
	return cofiniteSet{
		invert: invert,
		values: set.NewStringSet(values...),
	}
}

func (s *cofiniteSet) Contains(value string) bool {
	return s.values.Contains(value) != s.invert
}

func (s *cofiniteSet) Union(other cofiniteSet) cofiniteSet {
	if !s.invert && !other.invert {
		return cofiniteSet{
			values: s.values.Union(other.values),
		}
	}
	if s.invert && !other.invert {
		return cofiniteSet{
			values: s.values.Difference(other.values),
			invert: true,
		}
	}
	if !s.invert && other.invert {
		return cofiniteSet{
			values: other.values.Difference(s.values),
			invert: true,
		}
	}
	// s.invert && other.invert
	return cofiniteSet{
		values: s.values.Intersect(other.values),
		invert: true,
	}
}

func (s *cofiniteSet) Intersect(other cofiniteSet) cofiniteSet {
	if !s.invert && !other.invert {
		return cofiniteSet{
			values: s.values.Intersect(other.values),
		}
	}
	if s.invert && !other.invert {
		return cofiniteSet{
			values: other.values.Difference(s.values),
		}
	}
	if !s.invert && other.invert {
		return cofiniteSet{
			values: s.values.Difference(other.values),
		}
	}
	// s.invert && other.invert
	return cofiniteSet{
		values: s.values.Union(other.values),
		invert: true,
	}
}
