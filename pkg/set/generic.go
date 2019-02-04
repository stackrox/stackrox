package set

import (
	"sort"

	"github.com/deckarep/golang-set"
	"github.com/mauricelam/genny/generic"
)

// If you want to add a set for your custom type, simply add another go generate line along with the
// existing ones. If you're creating a set for a primitive type, you can follow the example of "string"
// and create the generated file in this package.
// Sometimes, you might need to create it in the same package where it is defined to avoid import cycles.
// The permission set is an example of how to do that.
// You can also specify the -imp command to specify additional imports in your generated file, if required.

// KeyType represents a generic type that we want to have a set of.
//go:generate genny -in=$GOFILE -out=gen-string-$GOFILE gen "KeyType=string"
//go:generate genny -in=$GOFILE -out=gen-int-$GOFILE gen "KeyType=int"
//go:generate genny -in=$GOFILE -out=gen-uint32-$GOFILE gen "KeyType=uint32"
//go:generate genny -in=$GOFILE -out=$GOPATH/src/github.com/stackrox/rox/pkg/auth/permissions/set.go -pkg permissions gen "KeyType=Resource"
type KeyType generic.Type

// KeyTypeSet will get translated to generic sets.
// It uses mapset.Set as the underlying implementation, so it comes with a bunch
// of utility methods, and is thread-safe.
type KeyTypeSet struct {
	underlying mapset.Set
}

// Add adds an element of type KeyType.
func (k KeyTypeSet) Add(i KeyType) bool {
	if k.underlying == nil {
		k.underlying = mapset.NewSet()
	}

	return k.underlying.Add(i)
}

// Remove removes an element of type KeyType.
func (k KeyTypeSet) Remove(i KeyType) {
	if k.underlying != nil {
		k.underlying.Remove(i)
	}
}

// Contains returns whether the set contains an element of type KeyType.
func (k KeyTypeSet) Contains(i KeyType) bool {
	if k.underlying != nil {
		return k.underlying.Contains(i)
	}
	return false
}

// Cardinality returns the number of elements in the set.
func (k KeyTypeSet) Cardinality() int {
	if k.underlying != nil {
		return k.underlying.Cardinality()
	}
	return 0
}

// Difference returns a new set with all elements of k not in other.
func (k KeyTypeSet) Difference(other KeyTypeSet) KeyTypeSet {
	if k.underlying == nil {
		return KeyTypeSet{underlying: other.underlying}
	} else if other.underlying == nil {
		return KeyTypeSet{underlying: k.underlying}
	}

	return KeyTypeSet{underlying: k.underlying.Difference(other.underlying)}
}

// Intersect returns a new set with the intersection of the members of both sets.
func (k KeyTypeSet) Intersect(other KeyTypeSet) KeyTypeSet {
	if k.underlying != nil && other.underlying != nil {
		return KeyTypeSet{underlying: k.underlying.Intersect(other.underlying)}
	}
	return KeyTypeSet{}
}

// Union returns a new set with the union of the members of both sets.
func (k KeyTypeSet) Union(other KeyTypeSet) KeyTypeSet {
	if k.underlying == nil {
		return KeyTypeSet{underlying: other.underlying}
	} else if other.underlying == nil {
		return KeyTypeSet{underlying: k.underlying}
	}

	return KeyTypeSet{underlying: k.underlying.Union(other.underlying)}
}

// AsSlice returns a slice of the elements in the set. The order is unspecified.
func (k KeyTypeSet) AsSlice() []KeyType {
	if k.underlying == nil {
		return nil
	}
	elems := make([]KeyType, 0, k.Cardinality())
	for elem := range k.underlying.Iter() {
		elems = append(elems, elem.(KeyType))
	}
	return elems
}

// AsSortedSlice returns a slice of the elements in the set, sorted using the passed less function.
func (k KeyTypeSet) AsSortedSlice(less func(i, j KeyType) bool) []KeyType {
	slice := k.AsSlice()
	if len(slice) < 2 {
		return slice
	}
	// Since we're generating the code, we might as well use sort.Sort
	// and avoid paying the reflection penalty of sort.Slice.
	sortable := &sortableKeyTypeSlice{slice: slice, less: less}
	sort.Sort(sortable)
	return sortable.slice
}

// IsInitialized returns whether the set has been initialized
func (k KeyTypeSet) IsInitialized() bool {
	return k.underlying != nil
}

// NewKeyTypeSet returns a new set with the given key type.
func NewKeyTypeSet(initial ...KeyType) KeyTypeSet {
	k := KeyTypeSet{underlying: mapset.NewSet()}
	for _, elem := range initial {
		k.Add(elem)
	}
	return k
}

type sortableKeyTypeSlice struct {
	slice []KeyType
	less  func(i, j KeyType) bool
}

func (s *sortableKeyTypeSlice) Len() int {
	return len(s.slice)
}

func (s *sortableKeyTypeSlice) Less(i, j int) bool {
	return s.less(s.slice[i], s.slice[j])
}

func (s *sortableKeyTypeSlice) Swap(i, j int) {
	s.slice[j], s.slice[i] = s.slice[i], s.slice[j]
}
