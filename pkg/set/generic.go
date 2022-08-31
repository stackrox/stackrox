package set

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mauricelam/genny/generic"
)

// If you want to add a set for your custom type, simply add another go generate line along with the
// existing ones. If you're creating a set for a primitive type, you can follow the example of "string"
// and create the generated file in this package.
// For non-primitive sets, please make the generated code files go outside this package.
// Sometimes, you might need to create it in the same package where it is defined to avoid import cycles.
// The permission set is an example of how to do that.
// You can also specify the -imp command to specify additional imports in your generated file, if required.

// KeyType represents a generic type that we want to have a set of.
//
//go:generate genny -in=$GOFILE -out=gen-string-$GOFILE gen "KeyType=string"
//go:generate genny -in=$GOFILE -out=gen-int-$GOFILE gen "KeyType=int"
//go:generate genny -in=$GOFILE -out=gen-uint32-$GOFILE gen "KeyType=uint32"
//go:generate genny -in=$GOFILE -out=gen-uint64-$GOFILE gen "KeyType=uint64"
type KeyType generic.Type

// KeyTypeSet will get translated to generic sets.
type KeyTypeSet map[KeyType]struct{}

// Add adds an element of type KeyType.
func (k *KeyTypeSet) Add(i KeyType) bool {
	if *k == nil {
		*k = make(map[KeyType]struct{})
	}

	oldLen := len(*k)
	(*k)[i] = struct{}{}
	return len(*k) > oldLen
}

// AddMatching is a utility function that adds all the elements that match the given function to the set.
func (k *KeyTypeSet) AddMatching(matchFunc func(KeyType) bool, elems ...KeyType) bool {
	oldLen := len(*k)
	for _, elem := range elems {
		if !matchFunc(elem) {
			continue
		}
		if *k == nil {
			*k = make(map[KeyType]struct{})
		}
		(*k)[elem] = struct{}{}
	}
	return len(*k) > oldLen
}

// AddAll adds all elements of type KeyType. The return value is true if any new element
// was added.
func (k *KeyTypeSet) AddAll(is ...KeyType) bool {
	if len(is) == 0 {
		return false
	}
	if *k == nil {
		*k = make(map[KeyType]struct{})
	}

	oldLen := len(*k)
	for _, i := range is {
		(*k)[i] = struct{}{}
	}
	return len(*k) > oldLen
}

// Remove removes an element of type KeyType.
func (k *KeyTypeSet) Remove(i KeyType) bool {
	if len(*k) == 0 {
		return false
	}

	oldLen := len(*k)
	delete(*k, i)
	return len(*k) < oldLen
}

// RemoveAll removes the given elements.
func (k *KeyTypeSet) RemoveAll(is ...KeyType) bool {
	if len(*k) == 0 {
		return false
	}

	oldLen := len(*k)
	for _, i := range is {
		delete(*k, i)
	}
	return len(*k) < oldLen
}

// RemoveMatching removes all elements that match a given predicate.
func (k *KeyTypeSet) RemoveMatching(pred func(KeyType) bool) bool {
	if len(*k) == 0 {
		return false
	}

	oldLen := len(*k)
	for elem := range *k {
		if pred(elem) {
			delete(*k, elem)
		}
	}
	return len(*k) < oldLen
}

// Contains returns whether the set contains an element of type KeyType.
func (k KeyTypeSet) Contains(i KeyType) bool {
	_, ok := k[i]
	return ok
}

// Cardinality returns the number of elements in the set.
func (k KeyTypeSet) Cardinality() int {
	return len(k)
}

// IsEmpty returns whether the underlying set is empty (includes uninitialized).
//
// Deprecated: use Cardinality() == 0 instead
func (k KeyTypeSet) IsEmpty() bool {
	return len(k) == 0
}

// Clone returns a copy of this set.
func (k KeyTypeSet) Clone() KeyTypeSet {
	if k == nil {
		return nil
	}
	cloned := make(map[KeyType]struct{}, len(k))
	for elem := range k {
		cloned[elem] = struct{}{}
	}
	return cloned
}

// Difference returns a new set with all elements of k not in other.
func (k KeyTypeSet) Difference(other KeyTypeSet) KeyTypeSet {
	if len(k) == 0 || len(other) == 0 {
		return k.Clone()
	}

	retained := make(map[KeyType]struct{}, len(k))
	for elem := range k {
		if !other.Contains(elem) {
			retained[elem] = struct{}{}
		}
	}
	return retained
}

// Helper function for intersections.
func (k KeyTypeSet) getSmallerLargerAndMaxIntLen(other KeyTypeSet) (smaller KeyTypeSet, larger KeyTypeSet, maxIntLen int) {
	maxIntLen = len(k)
	smaller, larger = k, other
	if l := len(other); l < maxIntLen {
		maxIntLen = l
		smaller, larger = larger, smaller
	}
	return smaller, larger, maxIntLen
}

// Intersects returns whether the set has a non-empty intersection with the other set.
func (k KeyTypeSet) Intersects(other KeyTypeSet) bool {
	smaller, larger, maxIntLen := k.getSmallerLargerAndMaxIntLen(other)
	if maxIntLen == 0 {
		return false
	}
	for elem := range smaller {
		if _, ok := larger[elem]; ok {
			return true
		}
	}
	return false
}

// Intersect returns a new set with the intersection of the members of both sets.
func (k KeyTypeSet) Intersect(other KeyTypeSet) KeyTypeSet {
	smaller, larger, maxIntLen := k.getSmallerLargerAndMaxIntLen(other)
	if maxIntLen == 0 {
		return nil
	}

	retained := make(map[KeyType]struct{}, maxIntLen)
	for elem := range smaller {
		if _, ok := larger[elem]; ok {
			retained[elem] = struct{}{}
		}
	}
	return retained
}

// Union returns a new set with the union of the members of both sets.
func (k KeyTypeSet) Union(other KeyTypeSet) KeyTypeSet {
	if len(k) == 0 {
		return other.Clone()
	} else if len(other) == 0 {
		return k.Clone()
	}

	underlying := make(map[KeyType]struct{}, len(k)+len(other))
	for elem := range k {
		underlying[elem] = struct{}{}
	}
	for elem := range other {
		underlying[elem] = struct{}{}
	}
	return underlying
}

// Equal returns a bool if the sets are equal
func (k KeyTypeSet) Equal(other KeyTypeSet) bool {
	thisL, otherL := len(k), len(other)
	if thisL == 0 && otherL == 0 {
		return true
	}
	if thisL != otherL {
		return false
	}
	for elem := range k {
		if _, ok := other[elem]; !ok {
			return false
		}
	}
	return true
}

// AsSlice returns a slice of the elements in the set. The order is unspecified.
func (k KeyTypeSet) AsSlice() []KeyType {
	if len(k) == 0 {
		return nil
	}
	elems := make([]KeyType, 0, len(k))
	for elem := range k {
		elems = append(elems, elem)
	}
	return elems
}

// GetArbitraryElem returns an arbitrary element from the set.
// This can be useful if, for example, you know the set has exactly one
// element, and you want to pull it out.
// If the set is empty, the zero value is returned.
func (k KeyTypeSet) GetArbitraryElem() (arbitraryElem KeyType) {
	for elem := range k {
		arbitraryElem = elem
		break
	}
	return arbitraryElem
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
	*k = nil
}

// Freeze returns a new, frozen version of the set.
func (k KeyTypeSet) Freeze() FrozenKeyTypeSet {
	return NewFrozenKeyTypeSetFromMap(k)
}

// ElementsString returns a string representation of all elements, with individual element strings separated by `sep`.
// The string representation of an individual element is obtained via `fmt.Fprint`.
func (k KeyTypeSet) ElementsString(sep string) string {
	if len(k) == 0 {
		return ""
	}
	var sb strings.Builder
	first := true
	for elem := range k {
		if !first {
			sb.WriteString(sep)
		}
		fmt.Fprint(&sb, elem)
		first = false
	}
	return sb.String()
}

// NewKeyTypeSet returns a new thread unsafe set with the given key type.
func NewKeyTypeSet(initial ...KeyType) KeyTypeSet {
	underlying := make(map[KeyType]struct{}, len(initial))
	for _, elem := range initial {
		underlying[elem] = struct{}{}
	}
	return underlying
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

// ElementsString returns a string representation of all elements, with individual element strings separated by `sep`.
// The string representation of an individual element is obtained via `fmt.Fprint`.
func (k FrozenKeyTypeSet) ElementsString(sep string) string {
	if len(k.underlying) == 0 {
		return ""
	}
	var sb strings.Builder
	first := true
	for elem := range k.underlying {
		if !first {
			sb.WriteString(sep)
		}
		fmt.Fprint(&sb, elem)
		first = false
	}
	return sb.String()
}

// The following functions make use of casting `k.underlying` into a mutable Set. This is safe, since we never leak
// references to these objects, and only invoke mutable set methods that are guaranteed to return a new copy.

// Union returns a frozen set that represents the union between this and other.
func (k FrozenKeyTypeSet) Union(other FrozenKeyTypeSet) FrozenKeyTypeSet {
	if len(k.underlying) == 0 {
		return other
	}
	if len(other.underlying) == 0 {
		return k
	}
	return FrozenKeyTypeSet{
		underlying: KeyTypeSet(k.underlying).Union(other.underlying),
	}
}

// Intersect returns a frozen set that represents the intersection between this and other.
func (k FrozenKeyTypeSet) Intersect(other FrozenKeyTypeSet) FrozenKeyTypeSet {
	return FrozenKeyTypeSet{
		underlying: KeyTypeSet(k.underlying).Intersect(other.underlying),
	}
}

// Difference returns a frozen set that represents the set difference between this and other.
func (k FrozenKeyTypeSet) Difference(other FrozenKeyTypeSet) FrozenKeyTypeSet {
	return FrozenKeyTypeSet{
		underlying: KeyTypeSet(k.underlying).Difference(other.underlying),
	}
}

// Unfreeze returns a mutable set with the same contents as this frozen set. This set will not be affected by any
// subsequent modifications to the returned set.
func (k FrozenKeyTypeSet) Unfreeze() KeyTypeSet {
	return KeyTypeSet(k.underlying).Clone()
}
