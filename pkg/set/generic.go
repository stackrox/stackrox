package set

import (
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
//go:generate genny -in=$GOFILE -out=$GOPATH/src/github.com/stackrox/rox/pkg/auth/permissions/set.go -pkg permissions gen "KeyType=Permission"
type KeyType generic.Type

// KeyTypeSet will get translated to generic sets.
// It uses mapset.Set as the underlying implementation, so it comes with a bunch
// of utility methods, and is thread-safe.
type KeyTypeSet struct {
	underlying mapset.Set
}

// Add adds an element of type KeyType.
func (k KeyTypeSet) Add(i KeyType) bool {
	return k.underlying.Add(i)
}

// Remove removes an element of type KeyType.
func (k KeyTypeSet) Remove(i KeyType) {
	k.underlying.Remove(i)
}

// Contains returns whether the set contains an element of type KeyType.
func (k KeyTypeSet) Contains(i KeyType) bool {
	return k.underlying.Contains(i)
}

// Cardinality returns the number of elements in the set.
func (k KeyTypeSet) Cardinality() int {
	return k.underlying.Cardinality()
}

// Intersect returns a new set with the intersection of the members of both sets.
func (k KeyTypeSet) Intersect(other KeyTypeSet) KeyTypeSet {
	return KeyTypeSet{underlying: k.underlying.Intersect(other.underlying)}
}

// Union returns a new set with the union of the members of both sets.
func (k KeyTypeSet) Union(other KeyTypeSet) KeyTypeSet {
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

// NewKeyTypeSet returns a new set with the given key type.
func NewKeyTypeSet(initial ...KeyType) KeyTypeSet {
	k := KeyTypeSet{underlying: mapset.NewSet()}
	for _, elem := range initial {
		k.Add(elem)
	}
	return k
}
