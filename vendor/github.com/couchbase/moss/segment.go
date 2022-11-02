//  Copyright (c) 2016 Couchbase, Inc.
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the
//  License. You may obtain a copy of the License at
//    http://www.apache.org/licenses/LICENSE-2.0
//  Unless required by applicable law or agreed to in writing,
//  software distributed under the License is distributed on an "AS
//  IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
//  express or implied. See the License for the specific language
//  governing permissions and limitations under the License.

package moss

import (
	"bytes"
	"fmt"
	"sort"
)

// SegmentKindBasic is the code for a basic, persistable segment
// implementation, which represents a segment as two arrays: an array
// of contiguous key-val bytes [key0, val0, key1, val1, ... keyN,
// valN], and an array of offsets plus lengths into the first array.
var SegmentKindBasic = "a"

func init() {
	SegmentLoaders[SegmentKindBasic] = loadBasicSegment
	SegmentPersisters[SegmentKindBasic] = persistBasicSegment
}

// A SegmentCursor represents a handle for iterating through consecutive
// op/key/value tuples.
type SegmentCursor interface {
	// Current returns the operation/key/value pointed to by the cursor.
	Current() (operation uint64, key []byte, val []byte)

	// Seek advances current to point to specified key.
	// If the seek key is less than the original startKeyInclusive
	// used to create this cursor, it will seek to that startKeyInclusive
	// instead.
	// If the cursor is not pointing at a valid entry ErrIteratorDone
	// is returned.
	Seek(startKeyInclusive []byte) error

	// Next moves the cursor to the next entry.  If there is no Next
	// entry, ErrIteratorDone is returned.
	Next() error
}

// A Segment represents the read-oriented interface for a segment.
type Segment interface {
	// Returns the kind of segment, used for persistence.
	Kind() string

	// Len returns the number of ops in the segment.
	Len() int

	// NumKeyValBytes returns the number of bytes used for key-val data.
	NumKeyValBytes() (uint64, uint64)

	// Get returns the operation and value associated with the given key.
	// If the key does not exist, the operation is 0, and the val is nil.
	// If an error occurs it is returned instead of the operation and value.
	Get(key []byte) (operation uint64, val []byte, err error)

	// Cursor returns an SegmentCursor that will iterate over entries
	// from the given (inclusive) start key, through the given (exclusive)
	// end key.
	Cursor(startKeyInclusive []byte, endKeyExclusive []byte) (SegmentCursor,
		error)

	// Returns true if the segment is already sorted, and returns
	// false if the sorting is only asynchronously scheduled.
	RequestSort(synchronous bool) bool
}

// SegmentValidater is an optional interface that can be implemented by
// any Segment to allow additional validation in test cases.  The
// method of this interface is NOT invoked during the normal
// runtime usage of a Segment.
type SegmentValidater interface {

	// Valid examines the state of the segment, any problem is returned
	// as an error.
	Valid() error
}

// A SegmentMutator represents the mutation methods of a segment.
type SegmentMutator interface {
	Mutate(operation uint64, key, val []byte) error
}

// A SegmentPersister represents a segment that can be persisted.
type SegmentPersister interface {
	Persist(file File, options *StoreOptions) (SegmentLoc, error)
}

// A segment is a basic implementation of the segment related
// interfaces and represents a sequence of key-val entries or
// operations.  A segment's kvs will be sorted by key when the segment
// is pushed into the collection.  A segment implements the Batch
// interface.
type segment struct {
	// Each key-val operation is encoded as 2 uint64's...
	// - operation (see: maskOperation) |
	//       key length (see: maskKeyLength) |
	//       val length (see: maskValLength).
	// - start index into buf for key-val bytes.
	kvs []uint64

	// Contiguous backing memory for the keys and vals of the segment.
	buf []byte

	// If this segment needs sorting, then needSorterCh will be
	// non-nil and also the first goroutine that reads successfully
	// from needSorterCh becomes the sorter of this segment.  All
	// other goroutines must instead wait on the waitSortedCh.
	needSorterCh chan bool

	// Once the sorter of this segment is done sorting the kvs, it
	// close()'s the waitSortedCh, treating waitSortedCh like a
	// one-way latch.  The needSorterCh and waitSortedCh will either
	// be nil or non-nil together.  A segment that was "born
	// sorted" will have needSorterCh and waitSortedCh as both nil.
	waitSortedCh chan struct{}

	totOperationSet   uint64
	totOperationDel   uint64
	totOperationMerge uint64
	totKeyByte        uint64
	totValByte        uint64

	rootCollection *collection // Non-nil when segment is from a batch.

	// In-memory index, immutable after segment initialization.
	index *segmentKeysIndex
}

// See the OperationXxx consts.
const maskOperation = uint64(0x0F00000000000000)

// Max key length is 2^24, from 24 bits key length.
const maskKeyLength = uint64(0x00FFFFFF00000000)

const maxKeyLength = 1<<24 - 1

// Max val length is 2^28, from 28 bits val length.
const maskValLength = uint64(0x000000000FFFFFFF)

const maxValLength = 1<<28 - 1

const maskRESERVED = uint64(0xF0000000F0000000)

// newSegment() allocates a segment with hinted amount of resources.
func newSegment(totalOps, totalKeyValBytes int) (*segment, error) {
	return &segment{
		kvs: make([]uint64, 0, totalOps*2),
		buf: make([]byte, 0, totalKeyValBytes),
	}, nil
}

func (a *segment) Kind() string { return SegmentKindBasic }

// Close releases resources associated with the segment.
func (a *segment) Close() error {
	return nil
}

// Set copies the key and val bytes into the segment as a "set"
// mutation.  The key must be unique (not repeated) within the
// segment.
func (a *segment) Set(key, val []byte) error {
	return a.mutate(OperationSet, key, val)
}

// Del copies the key bytes into the segment as a "deletion" mutation.
// The key must be unique (not repeated) within the segment.
func (a *segment) Del(key []byte) error {
	return a.mutate(OperationDel, key, nil)
}

// Merge creates or updates a key-val entry in the Collection via the
// MergeOperator defined in the CollectionOptions.  The key must be
// unique (not repeated) within the segment.
func (a *segment) Merge(key, val []byte) error {
	return a.mutate(OperationMerge, key, val)
}

// ------------------------------------------------------

// Alloc provides a slice of bytes "owned" by the segment, to reduce
// extra copying of memory.  See the Collection.NewBatch() method.
func (a *segment) Alloc(numBytes int) ([]byte, error) {
	bufLen := len(a.buf)
	bufCap := cap(a.buf)

	if numBytes > bufCap-bufLen {
		return nil, ErrAllocTooLarge
	}

	rv := a.buf[bufLen : bufLen+numBytes]

	a.buf = a.buf[0 : bufLen+numBytes]

	return rv, nil
}

// AllocSet is like Set(), but the caller must provide []byte
// parameters that came from Alloc(), for less buffer copying.
func (a *segment) AllocSet(keyFromAlloc, valFromAlloc []byte) error {
	bufCap := cap(a.buf)

	keyStart := bufCap - cap(keyFromAlloc)

	return a.mutateEx(OperationSet,
		keyStart, len(keyFromAlloc), len(valFromAlloc))
}

// AllocDel is like Del(), but the caller must provide []byte
// parameters that came from Alloc(), for less buffer copying.
func (a *segment) AllocDel(keyFromAlloc []byte) error {
	bufCap := cap(a.buf)

	keyStart := bufCap - cap(keyFromAlloc)

	return a.mutateEx(OperationDel,
		keyStart, len(keyFromAlloc), 0)
}

// AllocMerge is like Merge(), but the caller must provide []byte
// parameters that came from Alloc(), for less buffer copying.
func (a *segment) AllocMerge(keyFromAlloc, valFromAlloc []byte) error {
	bufCap := cap(a.buf)

	keyStart := bufCap - cap(keyFromAlloc)

	return a.mutateEx(OperationMerge,
		keyStart, len(keyFromAlloc), len(valFromAlloc))
}

// ------------------------------------------------------

func (a *segment) Mutate(operation uint64, key, val []byte) error {
	return a.mutate(operation, key, val)
}

func (a *segment) mutate(operation uint64, key, val []byte) error {
	keyStart := len(a.buf)
	a.buf = append(a.buf, key...)
	keyLength := len(a.buf) - keyStart

	valStart := len(a.buf)
	a.buf = append(a.buf, val...)
	valLength := len(a.buf) - valStart

	return a.mutateEx(operation, keyStart, keyLength, valLength)
}

func (a *segment) mutateEx(operation uint64,
	keyStart, keyLength, valLength int) error {
	if keyLength > maxKeyLength {
		return ErrKeyTooLarge
	}
	if valLength > maxValLength {
		return ErrValueTooLarge
	}

	if keyLength <= 0 && valLength <= 0 {
		keyStart = 0
	}

	opKlVl := encodeOpKeyLenValLen(operation, keyLength, valLength)

	a.kvs = append(a.kvs, opKlVl, uint64(keyStart))

	switch operation {
	case OperationSet:
		a.totOperationSet++
	case OperationDel:
		a.totOperationDel++
	case OperationMerge:
		a.totOperationMerge++
	default:
	}

	a.totKeyByte += uint64(keyLength)
	a.totValByte += uint64(valLength)

	return nil
}

// ------------------------------------------------------

// NumKeyValBytes returns the number of bytes used for key-val data.
func (a *segment) NumKeyValBytes() (uint64, uint64) {
	return a.totKeyByte, a.totValByte
}

// ------------------------------------------------------

// Len returns the number of ops in the segment.
func (a *segment) Len() int {
	return len(a.kvs) / 2
}

func (a *segment) Swap(i, j int) {
	x := i * 2
	y := j * 2

	// Operation + key length + val length.
	a.kvs[x], a.kvs[y] = a.kvs[y], a.kvs[x]

	x++
	y++

	a.kvs[x], a.kvs[y] = a.kvs[y], a.kvs[x] // Buf index.
}

func (a *segment) Less(i, j int) bool {
	x := i * 2
	y := j * 2

	kxLength := int((maskKeyLength & a.kvs[x]) >> 32)
	kxStart := int(a.kvs[x+1])
	kx := a.buf[kxStart : kxStart+kxLength]

	kyLength := int((maskKeyLength & a.kvs[y]) >> 32)
	kyStart := int(a.kvs[y+1])
	ky := a.buf[kyStart : kyStart+kyLength]

	return bytes.Compare(kx, ky) < 0
}

// ------------------------------------------------------

type segmentCursor struct {
	s     *segment
	start int
	end   int
	curr  int
}

func (c *segmentCursor) Current() (operation uint64, key []byte, val []byte) {
	if c.curr >= c.start && c.curr < c.end {
		operation, key, val = c.s.getOperationKeyVal(c.curr)
	}
	return
}

func (c *segmentCursor) Seek(startKeyInclusive []byte) error {
	c.curr = c.s.findStartKeyInclusivePos(startKeyInclusive)
	if c.curr < c.start {
		c.curr = c.start
	}
	if c.curr >= c.end {
		return ErrIteratorDone
	}
	return nil
}

func (c *segmentCursor) Next() error {
	c.curr++
	if c.curr >= c.end {
		return ErrIteratorDone
	}
	return nil
}

// nextDelta advances the cursor position by 'delta' steps.
func (c *segmentCursor) nextDelta(delta int) error {
	c.curr += delta
	if c.curr >= c.end {
		return ErrIteratorDone
	}
	return nil
}

// currentKey returns the array position and the key pointed to by the cursor.
func (c *segmentCursor) currentKey() (idx int, key []byte) {
	if c.curr >= c.start && c.curr < c.end {
		idx = c.curr
		_, key, _ = c.s.getOperationKeyVal(c.curr)
	}
	return
}

func (a *segment) Cursor(startKeyInclusive []byte, endKeyExclusive []byte) (
	SegmentCursor, error) {
	rv := &segmentCursor{
		s:   a,
		end: a.Len(),
	}
	rv.start = a.findStartKeyInclusivePos(startKeyInclusive)
	if endKeyExclusive != nil {
		rv.end = a.findStartKeyInclusivePos(endKeyExclusive)
	}
	rv.curr = rv.start
	return rv, nil
}

func (a *segment) Get(key []byte) (operation uint64, val []byte, err error) {
	var pos int
	pos, err = a.findKeyPos(key)
	if err != nil {
		return
	}

	if pos >= 0 {
		operation, _, val = a.getOperationKeyVal(pos)
	}
	return
}

// Searches for the key within the in-memory index of the segment
// if available. Returns left and right positions between which
// the key likely exists.
func (a *segment) searchIndex(key []byte) (int, int) {
	if a.index != nil {
		// Check the in-memory index for a more accurate window.
		return a.index.lookup(key)
	}

	return 0, a.Len()
}

func (a *segment) findKeyPos(key []byte) (int, error) {
	kvs := a.kvs
	buf := a.buf

	if len(kvs) < 2 {
		return -1, nil
	}

	startKeyLen := int((maskKeyLength & kvs[0]) >> 32)
	startKeyBeg := int(kvs[1])
	if startKeyBeg+startKeyLen > len(buf) {
		return -1, ErrSegmentCorrupted
	}
	// If key smaller than smallest key, return early.
	startCmp := bytes.Compare(key, buf[startKeyBeg:startKeyBeg+startKeyLen])
	if startCmp < 0 {
		return -1, nil
	}

	i, j := a.searchIndex(key)
	if i == j {
		return -1, nil
	}

	// additional best effort guard against mmap buf beyond eof
	x := 2 * (j - 1)
	if x+1 > len(kvs) {
		return -1, ErrSegmentCorrupted
	}
	endKeyLen := int((maskKeyLength & kvs[x]) >> 32)
	endKeyBeg := int(kvs[x+1])
	if endKeyBeg+endKeyLen > len(buf) {
		return -1, ErrSegmentCorrupted
	}

	for i < j {
		h := i + (j-i)/2 // Keep i <= h < j.
		x := h * 2
		klen := int((maskKeyLength & kvs[x]) >> 32)
		kbeg := int(kvs[x+1])
		cmp := bytes.Compare(buf[kbeg:kbeg+klen], key)
		if cmp == 0 {
			return h, nil
		} else if cmp < 0 {
			i = h + 1
		} else {
			j = h
		}
	}

	return -1, nil
}

// FindStartKeyInclusivePos() returns the logical entry position for
// the given (inclusive) start key.  With segment keys of [b, d, f],
// looking for 'c' will return 1.  Looking for 'd' will return 1.
// Looking for 'g' will return 3.  Looking for 'a' will return 0.
func (a *segment) findStartKeyInclusivePos(startKeyInclusive []byte) int {
	kvs := a.kvs
	buf := a.buf

	i, j := a.searchIndex(startKeyInclusive)
	if i == j {
		return i
	}

	startKeyLen := int((maskKeyLength & kvs[0]) >> 32)
	startKeyBeg := int(kvs[1])
	startCmp := bytes.Compare(startKeyInclusive,
		buf[startKeyBeg:startKeyBeg+startKeyLen])
	if startCmp < 0 { // If key smaller than smallest key, return early.
		return i
	}

	for i < j {
		h := i + (j-i)/2 // Keep i <= h < j.
		x := h * 2
		klen := int((maskKeyLength & kvs[x]) >> 32)
		kbeg := int(kvs[x+1])
		cmp := bytes.Compare(buf[kbeg:kbeg+klen], startKeyInclusive)
		if cmp == 0 {
			return h
		} else if cmp < 0 {
			i = h + 1
		} else {
			j = h
		}
	}

	return i
}

// getOperationKeyVal() returns the operation, key, val for a given
// logical entry position in the segment.
func (a *segment) getOperationKeyVal(pos int) (uint64, []byte, []byte) {
	x := pos * 2
	if x < len(a.kvs) {
		opklvl := a.kvs[x]
		kstart := int(a.kvs[x+1])
		operation, keyLen, valLen := decodeOpKeyLenValLen(opklvl)
		vstart := kstart + keyLen

		return operation, a.buf[kstart:vstart], a.buf[vstart : vstart+valLen]
	}

	return 0, nil, nil
}

// ------------------------------------------------------

func encodeOpKeyLenValLen(operation uint64, keyLen, valLen int) uint64 {
	return (maskOperation & operation) |
		(maskKeyLength & (uint64(keyLen) << 32)) |
		(maskValLength & (uint64(valLen)))
}

func decodeOpKeyLenValLen(opklvl uint64) (uint64, int, int) {
	operation := maskOperation & opklvl
	keyLen := int((maskKeyLength & opklvl) >> 32)
	valLen := int(maskValLength & opklvl)
	return operation, keyLen, valLen
}

// ------------------------------------------------------
// readyDeferredSort() will create a ticket for the future sorter and
// a channel to wait for its completion
func (a *segment) readyDeferredSort() {
	a.needSorterCh = make(chan bool, 1)
	a.needSorterCh <- true // A ticket for the future sorter.
	close(a.needSorterCh)

	a.waitSortedCh = make(chan struct{})
}

// RequestSort() will either perform the previously deferred sorting,
// if the goroutine can acquire the 1 ticket from the needSorterCh.
// Or, requestSort() will ensure that a sorter is working on this
// segment.  Returns true if the segment is sorted, and returns false
// if the sorting is only asynchronously scheduled.
func (a *segment) RequestSort(synchronous bool) bool {
	if a.needSorterCh == nil {
		return true
	}

	iAmTheSorter := <-a.needSorterCh
	if iAmTheSorter {
		a.doSort()
		close(a.waitSortedCh) // Signal any waiters.
		return true
	}

	if synchronous {
		<-a.waitSortedCh // Wait for the sorter to be done.
		return true
	}

	return false
}

// doSort() will immediately sort this segment.
func (a *segment) doSort() {
	// After sorting, the segment is immutable and then safe for
	// concurrent reads.
	sort.Sort(a)

	if !SkipStats {
		go a.rootCollection.updateStats(a)
	}
}

// SkipStats allows advanced applications that don't care about
// correct stats to avoid some stats maintenance overhead.  Defaults
// to false (stats are correctly maintained).
var SkipStats bool

// ------------------------------------------------------

// Persist persists a basic segment, and allows a segment to meet the
// SegmentPersister interface.
func (a *segment) Persist(file File, options *StoreOptions) (rv SegmentLoc, err error) {
	finfo, err := file.Stat()
	if err != nil {
		return rv, err
	}

	persistKind := DefaultPersistKind
	if options.PersistKind != "" {
		persistKind = options.PersistKind
	}

	segmentPersister, exists := SegmentPersisters[persistKind]
	if !exists || segmentPersister == nil {
		return rv, fmt.Errorf("store: unknown PersistKind: %+v", persistKind)
	}

	return segmentPersister(a, file, finfo.Size(), nil)
}

// ------------------------------------------------------

// loadBasicSegment loads a basic segment.
func loadBasicSegment(sloc *SegmentLoc) (Segment, error) {
	var kvs []uint64
	var buf []byte
	var err error

	if sloc.KvsBytes > 0 {
		if sloc.KvsBytes > uint64(len(sloc.mref.buf)) {
			return nil, fmt.Errorf("store: load basic segment KvsOffset/KvsBytes too big,"+
				" len(mref.buf): %d, sloc: %+v", len(sloc.mref.buf), sloc)
		}

		kvsBytes := sloc.mref.buf[0:sloc.KvsBytes]
		kvs, err = ByteSliceToUint64Slice(kvsBytes)
		if err != nil {
			return nil, err
		}
	}

	if sloc.BufBytes > 0 {
		bufStart := sloc.BufOffset - sloc.KvsOffset
		if bufStart+sloc.BufBytes > uint64(len(sloc.mref.buf)) {
			return nil, fmt.Errorf("store: load basic segment BufOffset/BufBytes too big,"+
				" len(mref.buf): %d, sloc: %+v", len(sloc.mref.buf), sloc)
		}

		buf = sloc.mref.buf[bufStart : bufStart+sloc.BufBytes]
	}

	return &segment{
		kvs:             kvs,
		buf:             buf,
		totOperationSet: sloc.TotOpsSet,
		totOperationDel: sloc.TotOpsDel,
		totKeyByte:      sloc.TotKeyByte,
		totValByte:      sloc.TotValByte,
	}, nil
}

// ------------------------------------------------------

func persistBasicSegment(
	s Segment, file File, pos int64, options *StoreOptions) (rv SegmentLoc, err error) {

	seg, ok := s.(*segment)
	if !ok {
		return rv, fmt.Errorf("wrong segment type")
	}

	kvsBuf, err := Uint64SliceToByteSlice(seg.kvs)
	if err != nil {
		return rv, err
	}

	kvsPos := pageAlignCeil(pos)
	bufPos := pageAlignCeil(kvsPos + int64(len(kvsBuf)))

	ioCh := make(chan ioResult)

	go func() {
		kvsWritten, err := file.WriteAt(kvsBuf, kvsPos)
		ioCh <- ioResult{kind: "kvs", want: len(kvsBuf), got: kvsWritten, err: err}
	}()

	go func() {
		bufWritten, err := file.WriteAt(seg.buf, bufPos)
		ioCh <- ioResult{kind: "buf", want: len(seg.buf), got: bufWritten, err: err}
	}()

	resMap := map[string]ioResult{}
	for len(resMap) < 2 {
		res := <-ioCh
		if res.err != nil {
			return rv, res.err
		}
		if res.want != res.got {
			return rv, fmt.Errorf("store: persistSegment error writing,"+
				" res: %+v, err: %v", res, res.err)
		}
		resMap[res.kind] = res
	}

	close(ioCh)

	return SegmentLoc{
		Kind:       seg.Kind(),
		KvsOffset:  uint64(kvsPos),
		KvsBytes:   uint64(resMap["kvs"].got),
		BufOffset:  uint64(bufPos),
		BufBytes:   uint64(resMap["buf"].got),
		TotOpsSet:  seg.totOperationSet,
		TotOpsDel:  seg.totOperationDel,
		TotKeyByte: seg.totKeyByte,
		TotValByte: seg.totValByte,
	}, nil
}

func (a *segment) Valid() error {
	if a.kvs == nil || len(a.kvs) <= 0 {
		return fmt.Errorf("expected kvs")
	}
	if a.buf == nil || len(a.buf) <= 0 {
		return fmt.Errorf("expected buf")
	}
	for pos := 0; pos < a.Len(); pos++ {
		x := pos * 2
		if x < 0 || x >= len(a.kvs) {
			return fmt.Errorf("pos to x error")
		}

		opklvl := a.kvs[x]

		operation, keyLen, valLen := decodeOpKeyLenValLen(opklvl)
		if operation == 0 {
			return fmt.Errorf("should have some nonzero op")
		}

		kstart := int(a.kvs[x+1])
		vstart := kstart + keyLen

		if kstart+keyLen > len(a.buf) {
			return fmt.Errorf("key larger than buf, pos: %d, kstart: %d, keyLen: %d, len(buf): %d, op: %x",
				pos, kstart, keyLen, len(a.buf), operation)
		}
		if vstart+valLen > len(a.buf) {
			return fmt.Errorf("val larger than buf, pos: %d, vstart: %d, valLen: %d, len(buf): %d, op: %x",
				pos, vstart, valLen, len(a.buf), operation)
		}
	}

	return nil
}

// ------------------------------------------------------

// Builds and initializes the in-memory index for the segment.
func (a *segment) buildIndex(quota int, minKeyBytes int) {
	if int(a.totKeyByte) < minKeyBytes {
		// Build the index only if the total key bytes is greater
		// than or equal to the SegmentKeysIndexMinKeyBytes.
		return
	}

	keyCount := a.Len()
	if keyCount == 0 {
		return // No keys to index.
	}

	keyAvgSize := int(a.totKeyByte) / keyCount

	sindex := newSegmentKeysIndex(quota, keyCount, keyAvgSize)
	if sindex == nil {
		return
	}

	scursor := &segmentCursor{
		s:   a,
		end: a.Len(),
	}

	for {
		keyIdx, key := scursor.currentKey()
		if key == nil {
			break
		}

		if !sindex.add(keyIdx, key) {
			break // Out of space.
		}

		err := scursor.nextDelta(sindex.hop)
		if err != nil {
			break
		}
	}

	a.index = sindex
}

// ------------------------------------------------------

type batch struct {
	// A batch is a type of segment with childCollections.
	*segment

	// childBatches track the segments of child collections indexed by their
	// unique collection names.
	childBatches map[string]*batch
}

// deletedChildBatchMarker conveys a delete request from
// DelChildCollection() to ExecuteBatch().
var deletedChildBatchMarker = &batch{}

// newBatch() allocates a segment with hinted amount of resources.
func newBatch(rootCollection *collection, options BatchOptions) (
	*batch, error) {
	return &batch{
		segment: &segment{
			kvs:            make([]uint64, 0, options.TotalOps*2),
			buf:            make([]byte, 0, options.TotalKeyValBytes),
			rootCollection: rootCollection,
		},
		childBatches: nil, // Created later on demand.
	}, nil
}

func (b *batch) NewChildCollectionBatch(collectionName string,
	options BatchOptions) (Batch, error) {
	if len(collectionName) == 0 {
		return nil, ErrBadCollectionName
	}

	childBatch, err := newBatch(b.rootCollection, options)

	if b.childBatches == nil { // First creation of child batch.
		b.childBatches = make(map[string]*batch)
	}
	b.childBatches[collectionName] = childBatch

	return childBatch, err
}

func (b *batch) DelChildCollection(collectionName string) error {
	if len(collectionName) == 0 {
		return ErrNoSuchCollection
	}

	if b.childBatches == nil { // No previous child batches seen.
		b.childBatches = make(map[string]*batch)
	}

	// The parent batch remembers this batch with deletion sentinel.
	b.childBatches[collectionName] = deletedChildBatchMarker

	return nil
}

func (b *batch) readyDeferredSort() {
	if b == deletedChildBatchMarker {
		return
	}

	for _, childBatch := range b.childBatches {
		childBatch.readyDeferredSort()
	}

	b.segment.readyDeferredSort()
}

// RequestSort() returns true if all child batches are sorted and
// false if sorting has been asynchronously scheduled.
func (b *batch) RequestSort() bool {
	if b == deletedChildBatchMarker {
		return true
	}

	// false because we must never wait for sorter else it can deadlock.
	sorted := b.segment.RequestSort(false)

	for _, childBatch := range b.childBatches {
		sorted = childBatch.RequestSort() && sorted
	}

	return sorted
}

func (b *batch) doSort() {
	if b == deletedChildBatchMarker {
		return
	}

	b.segment.doSort()

	for _, childBatch := range b.childBatches {
		childBatch.doSort()
	}
}

func (b *batch) isEmpty() bool {
	if len(b.childBatches) != 0 {
		// Presence of child batches indicates a non-empty batch even
		// if the child batches themselves are empty. This is so that
		// collection creation/deletions will work.
		return false
	}

	return b.Len() <= 0
}
