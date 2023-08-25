package set

import (
	"fmt"
	"sort"
	"strings"
)

// Set is a generic set type.
type Set[KeyType comparable] map[KeyType]struct{}

// Add adds an element of type KeyType.
func (k *Set[KeyType]) Add(i KeyType) bool {
	if *k == nil {
		*k = make(map[KeyType]struct{})
	}

	oldLen := len(*k)
	(*k)[i] = struct{}{}
	return len(*k) > oldLen
}

// AddMatching is a utility function that adds all the elements that match the given function to the set.
func (k *Set[KeyType]) AddMatching(matchFunc func(KeyType) bool, elems ...KeyType) bool {
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
func (k *Set[KeyType]) AddAll(is ...KeyType) bool {
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
func (k *Set[KeyType]) Remove(i KeyType) bool {
	if len(*k) == 0 {
		return false
	}

	oldLen := len(*k)
	delete(*k, i)
	return len(*k) < oldLen
}

// RemoveAll removes the given elements.
func (k *Set[KeyType]) RemoveAll(is ...KeyType) bool {
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
func (k *Set[KeyType]) RemoveMatching(pred func(KeyType) bool) bool {
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
func (k Set[KeyType]) Contains(i KeyType) bool {
	_, ok := k[i]
	return ok
}

// Cardinality returns the number of elements in the set.
func (k Set[KeyType]) Cardinality() int {
	return len(k)
}

// IsEmpty returns whether the underlying set is empty (includes uninitialized).
//
// Deprecated: use Cardinality() == 0 instead
func (k Set[KeyType]) IsEmpty() bool {
	return len(k) == 0
}

// Clone returns a copy of this set.
func (k Set[KeyType]) Clone() Set[KeyType] {
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
func (k Set[KeyType]) Difference(other Set[KeyType]) Set[KeyType] {
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
func (k Set[KeyType]) getSmallerLargerAndMaxIntLen(other Set[KeyType]) (smaller Set[KeyType], larger Set[KeyType], maxIntLen int) {
	maxIntLen = len(k)
	smaller, larger = k, other
	if l := len(other); l < maxIntLen {
		maxIntLen = l
		smaller, larger = larger, smaller
	}
	return smaller, larger, maxIntLen
}

// Intersects returns whether the set has a non-empty intersection with the other set.
func (k Set[KeyType]) Intersects(other Set[KeyType]) bool {
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
func (k Set[KeyType]) Intersect(other Set[KeyType]) Set[KeyType] {
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
func (k Set[KeyType]) Union(other Set[KeyType]) Set[KeyType] {
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
func (k Set[KeyType]) Equal(other Set[KeyType]) bool {
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
func (k Set[KeyType]) AsSlice() []KeyType {
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
func (k Set[KeyType]) GetArbitraryElem() (arbitraryElem KeyType) {
	for elem := range k {
		arbitraryElem = elem
		break
	}
	return arbitraryElem
}

// AsSortedSlice returns a slice of the elements in the set, sorted using the passed less function.
func (k Set[KeyType]) AsSortedSlice(less func(i, j KeyType) bool) []KeyType {
	slice := k.AsSlice()
	if len(slice) < 2 {
		return slice
	}
	// Since we're generating the code, we might as well use sort.Sort
	// and avoid paying the reflection penalty of sort.Slice.
	sortable := &sortableSlice[KeyType]{slice: slice, less: less}
	sort.Sort(sortable)
	return sortable.slice
}

// Clear empties the set
func (k *Set[KeyType]) Clear() {
	*k = nil
}

// Freeze returns a new, frozen version of the set.
func (k Set[KeyType]) Freeze() FrozenSet[KeyType] {
	return NewFrozenSetFromMap(k)
}

// ElementsString returns a string representation of all elements, with individual element strings separated by `sep`.
// The string representation of an individual element is obtained via `fmt.Fprint`.
func (k Set[KeyType]) ElementsString(sep string) string {
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

// NewSet returns a new thread unsafe set with the given key type.
func NewSet[KeyType comparable](initial ...KeyType) Set[KeyType] {
	underlying := make(map[KeyType]struct{}, len(initial))
	for _, elem := range initial {
		underlying[elem] = struct{}{}
	}
	return underlying
}

type sortableSlice[KeyType comparable] struct {
	slice []KeyType
	less  func(i, j KeyType) bool
}

func (k *sortableSlice[KeyType]) Len() int {
	return len(k.slice)
}

func (k *sortableSlice[KeyType]) Less(i, j int) bool {
	return k.less(k.slice[i], k.slice[j])
}

func (k *sortableSlice[KeyType]) Swap(i, j int) {
	k.slice[j], k.slice[i] = k.slice[i], k.slice[j]
}

// A FrozenSet is a frozen set of KeyType elements, which
// cannot be modified after creation. This allows users to use it as if it were
// a "const" data structure, and also makes it slightly more optimal since
// we don't have to lock accesses to it.
type FrozenSet[KeyType comparable] struct {
	underlying map[KeyType]struct{}
}

// NewFrozenSetFromMap returns a new frozen set from the set-style map.
func NewFrozenSetFromMap[KeyType comparable](m map[KeyType]struct{}) FrozenSet[KeyType] {
	if len(m) == 0 {
		return FrozenSet[KeyType]{}
	}
	underlying := make(map[KeyType]struct{}, len(m))
	for elem := range m {
		underlying[elem] = struct{}{}
	}
	return FrozenSet[KeyType]{
		underlying: underlying,
	}
}

// NewFrozenSet returns a new frozen set with the provided elements.
func NewFrozenSet[KeyType comparable](elements ...KeyType) FrozenSet[KeyType] {
	underlying := make(map[KeyType]struct{}, len(elements))
	for _, elem := range elements {
		underlying[elem] = struct{}{}
	}
	return FrozenSet[KeyType]{
		underlying: underlying,
	}
}

// Contains returns whether the set contains the element.
func (k FrozenSet[KeyType]) Contains(elem KeyType) bool {
	_, ok := k.underlying[elem]
	return ok
}

// Cardinality returns the cardinality of the set.
func (k FrozenSet[KeyType]) Cardinality() int {
	return len(k.underlying)
}

// IsEmpty returns whether the underlying set is empty (includes uninitialized).
func (k FrozenSet[KeyType]) IsEmpty() bool {
	return len(k.underlying) == 0
}

// AsSlice returns the elements of the set. The order is unspecified.
func (k FrozenSet[KeyType]) AsSlice() []KeyType {
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
func (k FrozenSet[KeyType]) AsSortedSlice(less func(i, j KeyType) bool) []KeyType {
	slice := k.AsSlice()
	if len(slice) < 2 {
		return slice
	}
	// Since we're generating the code, we might as well use sort.Sort
	// and avoid paying the reflection penalty of sort.Slice.
	sortable := &sortableSlice[KeyType]{slice: slice, less: less}
	sort.Sort(sortable)
	return sortable.slice
}

// ElementsString returns a string representation of all elements, with individual element strings separated by `sep`.
// The string representation of an individual element is obtained via `fmt.Fprint`.
func (k FrozenSet[KeyType]) ElementsString(sep string) string {
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
func (k FrozenSet[KeyType]) Union(other FrozenSet[KeyType]) FrozenSet[KeyType] {
	if len(k.underlying) == 0 {
		return other
	}
	if len(other.underlying) == 0 {
		return k
	}
	return FrozenSet[KeyType]{
		underlying: Set[KeyType](k.underlying).Union(other.underlying),
	}
}

// Intersect returns a frozen set that represents the intersection between this and other.
func (k FrozenSet[KeyType]) Intersect(other FrozenSet[KeyType]) FrozenSet[KeyType] {
	return FrozenSet[KeyType]{
		underlying: Set[KeyType](k.underlying).Intersect(other.underlying),
	}
}

// Difference returns a frozen set that represents the set difference between this and other.
func (k FrozenSet[KeyType]) Difference(other FrozenSet[KeyType]) FrozenSet[KeyType] {
	return FrozenSet[KeyType]{
		underlying: Set[KeyType](k.underlying).Difference(other.underlying),
	}
}

// Unfreeze returns a mutable set with the same contents as this frozen set. This set will not be affected by any
// subsequent modifications to the returned set.
func (k FrozenSet[KeyType]) Unfreeze() Set[KeyType] {
	return Set[KeyType](k.underlying).Clone()
}
