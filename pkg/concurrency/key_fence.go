package concurrency

import (
	"bytes"

	"github.com/stackrox/rox/pkg/concurrency/sortedkeys"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stackrox/rox/pkg/sync"
)

// KeyFence provides a way of blocking set of keys from being operated on simultaneously by different processes.
type KeyFence interface {
	Lock(KeySet)
	Unlock(KeySet)
	DoWithLock(toLock KeySet, fn func())
	DoStatusWithLock(toLock KeySet, fn func() error) error
}

// NewKeyFence returns a new instance of a KeyFence with no keys currently locked.
func NewKeyFence() KeyFence {
	return &keyFenceImpl{}
}

type keyFenceImpl struct {
	lock sync.Mutex
	grp  sync.WaitGroup

	active []KeySet
}

func (b *keyFenceImpl) Lock(toLock KeySet) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if b.collidesNoLock(toLock) {
		b.grp.Wait()
	}
	b.active = append(b.active, toLock)
	b.grp.Add(1)
}

func (b *keyFenceImpl) Unlock(toUnlock KeySet) {
	b.grp.Done()

	b.lock.Lock()
	defer b.lock.Unlock()
	b.removeNoLock(toUnlock)
}

func (b *keyFenceImpl) DoWithLock(toLock KeySet, fn func()) {
	b.Lock(toLock)
	defer b.Unlock(toLock)

	fn()
}

func (b *keyFenceImpl) DoStatusWithLock(toLock KeySet, fn func() error) error {
	b.Lock(toLock)
	defer b.Unlock(toLock)

	return fn()
}

// collidesNoLock returns if any active KeySets collide with the input KeySet.
func (b *keyFenceImpl) collidesNoLock(in KeySet) bool {
	for _, ks := range b.active {
		if ks.Collides(in) {
			return true
		}
	}
	return false
}

// removeNoLock removes one KeySet from the list of active KeySets if one is equal to the input.
func (b *keyFenceImpl) removeNoLock(in KeySet) {
	filtered := b.active[:0]
	var removed bool
	for _, ks := range b.active {
		if !ks.Equals(in) || removed {
			filtered = append(filtered, ks)
		} else {
			removed = true
		}
	}
	b.active = filtered
}

// Available KeySet definitions.
////////////////////////////////

// KeySet provides ways of specifying which keys collide.
type KeySet interface {
	Collides(KeySet) bool
	Equals(KeySet) bool
	Clone() KeySet
}

// A Range of keys
//////////////////

// RangedKeySet returns a new KeySet representing a range of keys, from lower bound to upper bound.
func RangedKeySet(lower, upper []byte) KeySet {
	return &rangeKeySetImpl{
		lower: lower,
		upper: upper,
	}
}

type rangeKeySetImpl struct {
	lower, upper []byte
}

func (rks *rangeKeySetImpl) Collides(in KeySet) bool {
	if rangeSet, isRangeSet := in.(*rangeKeySetImpl); isRangeSet {
		// Target lower bound falls within this range.
		if bytes.Compare(rks.lower, rangeSet.lower) <= 0 && bytes.Compare(rks.upper, rangeSet.lower) >= 0 {
			return true
		}
		// Target upper bound falls within this range.
		if bytes.Compare(rks.lower, rangeSet.upper) <= 0 && bytes.Compare(rks.upper, rangeSet.upper) >= 0 {
			return true
		}
		// This lower bound falls within target range.
		if bytes.Compare(rangeSet.lower, rks.lower) <= 0 && bytes.Compare(rangeSet.upper, rks.lower) >= 0 {
			return true
		}
		// This upper bound falls within target range.
		if bytes.Compare(rangeSet.lower, rks.upper) <= 0 && bytes.Compare(rangeSet.upper, rks.upper) >= 0 {
			return true
		}
		return false
	} else if prefixSet, isPrefix := in.(*prefixKeySetImpl); isPrefix {
		return bytes.HasPrefix(rks.lower, prefixSet.prefix) || bytes.HasPrefix(rks.upper, prefixSet.prefix)
	} else if discrete, isDiscrete := in.(*discreteKeySetImpl); isDiscrete {
		if len(discrete.sorted) > 0 {
			return bytes.Compare(rks.lower, discrete.sorted[0]) >= 1 && bytes.Compare(rks.upper, discrete.sorted[len(discrete.sorted)-1]) <= 1
		}
		return false
	}
	return in.Collides(rks)
}

func (rks *rangeKeySetImpl) Equals(in KeySet) bool {
	if rangeSet, isRangeSet := in.(*rangeKeySetImpl); isRangeSet {
		return bytes.Equal(rks.lower, rangeSet.lower) && bytes.Equal(rks.upper, rangeSet.upper)
	}
	return false
}

func (rks *rangeKeySetImpl) Clone() KeySet {
	return &rangeKeySetImpl{
		lower: sliceutils.ShallowClone(rks.lower),
		upper: sliceutils.ShallowClone(rks.upper),
	}
}

// All Keys with a Prefix.
//////////////////////////

// PrefixKeySet returns a new KeySet representing all keys with a given prefix.
func PrefixKeySet(prefix []byte) KeySet {
	return &prefixKeySetImpl{
		prefix: prefix,
	}
}

type prefixKeySetImpl struct {
	prefix []byte
}

func (pkr *prefixKeySetImpl) Collides(in KeySet) bool {
	if prefixSet, isPrefixSet := in.(*prefixKeySetImpl); isPrefixSet {
		return bytes.HasPrefix(pkr.prefix, prefixSet.prefix) || bytes.HasPrefix(prefixSet.prefix, pkr.prefix)
	} else if discrete, isDiscrete := in.(*discreteKeySetImpl); isDiscrete {
		for _, key := range discrete.sorted {
			if dbhelper.HasPrefix(pkr.prefix, key) {
				return true
			}
		}
		return false
	}
	return in.Collides(pkr)
}

func (pkr *prefixKeySetImpl) Equals(in KeySet) bool {
	if prefixSet, isPrefixSet := in.(*prefixKeySetImpl); isPrefixSet {
		return bytes.Equal(pkr.prefix, prefixSet.prefix)
	}
	return false
}

func (pkr *prefixKeySetImpl) Clone() KeySet {
	return &prefixKeySetImpl{
		prefix: sliceutils.ShallowClone(pkr.prefix),
	}
}

// A specific set of keys.
//////////////////////////

// DiscreteKeySet returns a new KeySet representing a set of discrete key values.
func DiscreteKeySet(keys ...[]byte) KeySet {
	return &discreteKeySetImpl{
		sorted: sortedkeys.Sort(keys),
	}
}

type discreteKeySetImpl struct {
	sorted sortedkeys.SortedKeys
}

func (dks *discreteKeySetImpl) Collides(in KeySet) bool {
	if discreteSet, isDiscreteSet := in.(*discreteKeySetImpl); isDiscreteSet {
		for _, key := range discreteSet.sorted {
			if dks.sorted.Find(key) != -1 {
				return true
			}
		}
		return false
	}
	return in.Collides(dks)
}

func (dks *discreteKeySetImpl) Equals(in KeySet) bool {
	if discreteSet, isDiscreteSet := in.(*discreteKeySetImpl); isDiscreteSet {
		if len(dks.sorted) != len(discreteSet.sorted) {
			return false
		}
		for i := 0; i < len(dks.sorted); i++ {
			if !bytes.Equal(dks.sorted[i], discreteSet.sorted[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (dks *discreteKeySetImpl) Clone() KeySet {
	return &discreteKeySetImpl{
		sorted: sliceutils.ShallowClone2DSlice(dks.sorted),
	}
}

// All keys.
////////////

// EntireKeySet returns a KeySet representing all possible key values.
func EntireKeySet() KeySet {
	return &entireKeySetImpl{}
}

type entireKeySetImpl struct{}

func (dks *entireKeySetImpl) Collides(in KeySet) bool {
	_, isEmpty := in.(*emptyKeySetImpl)
	return !isEmpty
}

func (dks *entireKeySetImpl) Equals(in KeySet) bool {
	_, isEntireKeySet := in.(*entireKeySetImpl)
	return isEntireKeySet
}

func (dks *entireKeySetImpl) Clone() KeySet {
	return &entireKeySetImpl{}
}

// No keys.
////////////

// EmptyKeySet returns a KeySet representing no key values, so it will not collide with any keys.
func EmptyKeySet() KeySet {
	return &emptyKeySetImpl{}
}

type emptyKeySetImpl struct{}

func (ekr *emptyKeySetImpl) Collides(_ KeySet) bool {
	return false
}

func (ekr *emptyKeySetImpl) Equals(in KeySet) bool {
	_, isEmpty := in.(*emptyKeySetImpl)
	return isEmpty
}

func (ekr *emptyKeySetImpl) Clone() KeySet {
	return &emptyKeySetImpl{}
}
