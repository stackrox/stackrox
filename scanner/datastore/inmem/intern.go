package inmem

import (
	"fmt"
	"math"
)

// intern maps distinct comparable values to dense int32 IDs, retaining the
// insertion-ordered list for reconstruction.
type intern[T comparable] struct {
	m    map[T]int32
	list []T
}

func newIntern[T comparable]() *intern[T] {
	return &intern[T]{m: make(map[T]int32)}
}

// id returns the ID for v, allocating a new one if unseen. isNew reports whether
// this call allocated the ID (used by blob builders to compress each distinct
// value exactly once).
func (it *intern[T]) id(v T) (id int32, isNew bool) {
	if existing, ok := it.m[v]; ok {
		return existing, false
	}
	if len(it.list) >= math.MaxInt32 {
		panic(fmt.Errorf("inmem: intern dictionary overflow (%d)", len(it.list)))
	}
	id = int32(len(it.list))
	it.m[v] = id
	it.list = append(it.list, v)
	return id, true
}

// lookup returns the ID for v if present.
func (it *intern[T]) lookup(v T) (int32, bool) {
	id, ok := it.m[v]
	return id, ok
}
