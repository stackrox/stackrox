package set

import (
	"sort"

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
//go:generate genny -in=$GOFILE -out=gen-v1-search-cats-$GOFILE -imp=github.com/stackrox/rox/generated/api/v1 gen "KeyType=v1.SearchCategory"
//go:generate genny -in=$GOFILE -out=$GOPATH/src/github.com/stackrox/rox/pkg/auth/permissions/set.go -pkg permissions gen "KeyType=Resource"
//go:generate genny -in=$GOFILE -out=gen-upgrade-progress-state-$GOFILE -imp=github.com/stackrox/rox/generated/storage gen "KeyType=storage.UpgradeProgress_UpgradeState"
//go:generate genny -in=$GOFILE -out=$GOPATH/src/github.com/stackrox/rox/pkg/sensorupgrader/stage_set.go -pkg sensorupgrader gen "KeyType=Stage"
type KeyType generic.Type

// KeyTypeSet will get translated to generic sets.
type KeyTypeSet struct {
	underlying map[KeyType]struct{}
}

// Add adds an element of type KeyType.
func (k *KeyTypeSet) Add(i KeyType) bool {
	if k.underlying == nil {
		k.underlying = make(map[KeyType]struct{})
	}

	oldLen := len(k.underlying)
	k.underlying[i] = struct{}{}
	return len(k.underlying) > oldLen
}

// AddAll adds all elements of type KeyType. The return value is true if any new element
// was added.
func (k *KeyTypeSet) AddAll(is ...KeyType) bool {
	if len(is) == 0 {
		return false
	}
	if k.underlying == nil {
		k.underlying = make(map[KeyType]struct{})
	}

	oldLen := len(k.underlying)
	for _, i := range is {
		k.underlying[i] = struct{}{}
	}
	return len(k.underlying) > oldLen
}

// Remove removes an element of type KeyType.
func (k *KeyTypeSet) Remove(i KeyType) bool {
	if len(k.underlying) == 0 {
		return false
	}

	oldLen := len(k.underlying)
	delete(k.underlying, i)
	return len(k.underlying) < oldLen
}

// RemoveAll removes the given elements.
func (k *KeyTypeSet) RemoveAll(is ...KeyType) bool {
	if len(k.underlying) == 0 {
		return false
	}

	oldLen := len(k.underlying)
	for _, i := range is {
		delete(k.underlying, i)
	}
	return len(k.underlying) < oldLen
}

// RemoveMatching removes all elements that match a given predicate.
func (k *KeyTypeSet) RemoveMatching(pred func(KeyType) bool) bool {
	if len(k.underlying) == 0 {
		return false
	}

	oldLen := len(k.underlying)
	for elem := range k.underlying {
		if pred(elem) {
			delete(k.underlying, elem)
		}
	}
	return len(k.underlying) < oldLen
}

// Contains returns whether the set contains an element of type KeyType.
func (k KeyTypeSet) Contains(i KeyType) bool {
	_, ok := k.underlying[i]
	return ok
}

// Cardinality returns the number of elements in the set.
func (k KeyTypeSet) Cardinality() int {
	return len(k.underlying)
}

// IsEmpty returns whether the underlying set is empty (includes uninitialized).
func (k KeyTypeSet) IsEmpty() bool {
	return len(k.underlying) == 0
}

// Clone returns a copy of this set.
func (k KeyTypeSet) Clone() KeyTypeSet {
	if k.underlying == nil {
		return KeyTypeSet{}
	}
	cloned := make(map[KeyType]struct{}, len(k.underlying))
	for elem := range k.underlying {
		cloned[elem] = struct{}{}
	}
	return KeyTypeSet{underlying: cloned}
}

// Difference returns a new set with all elements of k not in other.
func (k KeyTypeSet) Difference(other KeyTypeSet) KeyTypeSet {
	if len(k.underlying) == 0 || len(other.underlying) == 0 {
		return k.Clone()
	}

	retained := make(map[KeyType]struct{}, len(k.underlying))
	for elem := range k.underlying {
		if !other.Contains(elem) {
			retained[elem] = struct{}{}
		}
	}
	return KeyTypeSet{underlying: retained}
}

// Intersect returns a new set with the intersection of the members of both sets.
func (k KeyTypeSet) Intersect(other KeyTypeSet) KeyTypeSet {
	maxIntLen := len(k.underlying)
	smaller, larger := k.underlying, other.underlying
	if l := len(other.underlying); l < maxIntLen {
		maxIntLen = l
		smaller, larger = larger, smaller
	}
	if maxIntLen == 0 {
		return KeyTypeSet{}
	}

	retained := make(map[KeyType]struct{}, maxIntLen)
	for elem := range smaller {
		if _, ok := larger[elem]; ok {
			retained[elem] = struct{}{}
		}
	}
	return KeyTypeSet{underlying: retained}
}

// Union returns a new set with the union of the members of both sets.
func (k KeyTypeSet) Union(other KeyTypeSet) KeyTypeSet {
	if len(k.underlying) == 0 {
		return other.Clone()
	} else if len(other.underlying) == 0 {
		return k.Clone()
	}

	underlying := make(map[KeyType]struct{}, len(k.underlying)+len(other.underlying))
	for elem := range k.underlying {
		underlying[elem] = struct{}{}
	}
	for elem := range other.underlying {
		underlying[elem] = struct{}{}
	}
	return KeyTypeSet{underlying: underlying}
}

// Equal returns a bool if the sets are equal
func (k KeyTypeSet) Equal(other KeyTypeSet) bool {
	thisL, otherL := len(k.underlying), len(other.underlying)
	if thisL == 0 && otherL == 0 {
		return true
	}
	if thisL != otherL {
		return false
	}
	for elem := range k.underlying {
		if _, ok := other.underlying[elem]; !ok {
			return false
		}
	}
	return true
}

// AsSlice returns a slice of the elements in the set. The order is unspecified.
func (k KeyTypeSet) AsSlice() []KeyType {
	if len(k.underlying) == 0 {
		return nil
	}
	elems := make([]KeyType, 0, len(k.underlying))
	for elem := range k.underlying {
		elems = append(elems, elem)
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

// Clear empties the set
func (k *KeyTypeSet) Clear() {
	k.underlying = nil
}

// Freeze returns a new, frozen version of the set.
func (k KeyTypeSet) Freeze() FrozenKeyTypeSet {
	return NewFrozenKeyTypeSetFromMap(k.underlying)
}

// NewKeyTypeSet returns a new thread unsafe set with the given key type.
func NewKeyTypeSet(initial ...KeyType) KeyTypeSet {
	underlying := make(map[KeyType]struct{}, len(initial))
	for _, elem := range initial {
		underlying[elem] = struct{}{}
	}
	return KeyTypeSet{underlying: underlying}
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

// A FrozenKeyTypeSet is a frozen set of KeyType elements, which
// cannot be modified after creation. This allows users to use it as if it were
// a "const" data structure, and also makes it slightly more optimal since
// we don't have to lock accesses to it.
type FrozenKeyTypeSet struct {
	underlying map[KeyType]struct{}
}

// NewFrozenKeyTypeSetFromMap returns a new frozen set from the set-style map.
func NewFrozenKeyTypeSetFromMap(m map[KeyType]struct{}) FrozenKeyTypeSet {
	if len(m) == 0 {
		return FrozenKeyTypeSet{}
	}
	underlying := make(map[KeyType]struct{}, len(m))
	for elem := range m {
		underlying[elem] = struct{}{}
	}
	return FrozenKeyTypeSet{
		underlying: underlying,
	}
}

// NewFrozenKeyTypeSet returns a new frozen set with the provided elements.
func NewFrozenKeyTypeSet(elements ...KeyType) FrozenKeyTypeSet {
	underlying := make(map[KeyType]struct{}, len(elements))
	for _, elem := range elements {
		underlying[elem] = struct{}{}
	}
	return FrozenKeyTypeSet{
		underlying: underlying,
	}
}

// Contains returns whether the set contains the element.
func (k FrozenKeyTypeSet) Contains(elem KeyType) bool {
	_, ok := k.underlying[elem]
	return ok
}

// Cardinality returns the cardinality of the set.
func (k FrozenKeyTypeSet) Cardinality() int {
	return len(k.underlying)
}

// IsEmpty returns whether the underlying set is empty (includes uninitialized).
func (k FrozenKeyTypeSet) IsEmpty() bool {
	return len(k.underlying) == 0
}

// AsSlice returns the elements of the set. The order is unspecified.
func (k FrozenKeyTypeSet) AsSlice() []KeyType {
	if len(k.underlying) == 0 {
		return nil
	}
	slice := make([]KeyType, 0, len(k.underlying))
	for elem := range k.underlying {
		slice = append(slice, elem)
	}
	return slice
}

// AsSortedSlice returns the elements of the set as a sorted slice.
func (k FrozenKeyTypeSet) AsSortedSlice(less func(i, j KeyType) bool) []KeyType {
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
