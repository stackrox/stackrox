//  Copyright (c) 2017 Couchbase, Inc.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing,
//  software distributed under the License is distributed on an "AS
//  IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
//  express or implied. See the License for the specific language
//  governing permissions and limitations under the License.

package moss

import (
	"bytes"
)

type segmentKeysIndex struct {
	// Number of keys that can be indexed.
	numIndexableKeys int

	// Keys that have been added so far.
	numKeys int

	// Size in bytes of all the indexed keys.
	numKeyBytes int

	// In-memory byte array of keys.
	data []byte

	// Start offsets of keys in the data array.
	offsets []uint32

	// Number of skips over keys in the segment kvs to arrive at the
	// next adjacent key in the data array.
	hop int

	// Total number of keys in the source segment.
	srcKeyCount int
}

// newSegmentKeysIndex preallocates the data/offsets arrays
// based on a calculated hop.
func newSegmentKeysIndex(quota int, srcKeyCount int,
	keyAvgSize int) *segmentKeysIndex {
	numIndexableKeys := quota / (keyAvgSize + 4 /* 4 for the offset */)
	if numIndexableKeys == 0 {
		return nil
	}

	hop := (srcKeyCount / numIndexableKeys) + 1

	data := make([]byte, numIndexableKeys*keyAvgSize)
	offsets := make([]uint32, numIndexableKeys)

	return &segmentKeysIndex{
		numIndexableKeys: numIndexableKeys,
		numKeys:          0,
		numKeyBytes:      0,
		data:             data,
		offsets:          offsets,
		hop:              hop,
		srcKeyCount:      srcKeyCount,
	}
}

// Adds a qualified entry to the index. Returns true if space
// still available, false otherwise.
func (s *segmentKeysIndex) add(keyIdx int, key []byte) bool {
	if s.numKeys >= s.numIndexableKeys {
		// All keys that can be indexed already have been,
		// return false indicating that there's no room for
		// anymore.
		return false
	}

	if len(key) > (len(s.data) - s.numKeyBytes) {
		// No room for any more keys.
		return false
	}

	if keyIdx%(s.hop) != 0 {
		// Key does not satisfy the hop condition.
		return true
	}

	s.offsets[s.numKeys] = uint32(s.numKeyBytes)
	copy(s.data[s.numKeyBytes:], key)
	s.numKeys++
	s.numKeyBytes += len(key)

	return true
}

// Fetches the range of offsets between which the key exists,
// if present at all. The returned leftPos and rightPos can
// directly be used as the left and right extreme cursors
// while binary searching over the source segment.
func (s *segmentKeysIndex) lookup(key []byte) (leftPos int, rightPos int) {
	i, j := 0, s.numKeys

	if i == j || s.numKeys < 2 {
		// The index either wasn't used or isn't of any use.
		rightPos = s.srcKeyCount
		return
	}

	// If key smaller than the first key, return early.
	keyStart := s.offsets[0]
	keyEnd := s.offsets[1]
	cmp := bytes.Compare(key, s.data[keyStart:keyEnd])
	if cmp < 0 {
		return
	}

	indexOfLastKey := s.numKeys - 1

	// If key larger than last key, return early.
	keyStart = s.offsets[indexOfLastKey]
	keyEnd = uint32(s.numKeyBytes)
	cmp = bytes.Compare(s.data[keyStart:keyEnd], key)
	if cmp < 0 {
		leftPos = (indexOfLastKey) * s.hop
		rightPos = s.srcKeyCount
		return
	}

	for i < j {
		h := i + (j-i)/2

		keyStart = s.offsets[h]
		if h < indexOfLastKey {
			keyEnd = s.offsets[h+1]
		} else {
			keyEnd = uint32(s.numKeyBytes)
		}

		cmp = bytes.Compare(s.data[keyStart:keyEnd], key)
		if cmp == 0 {
			leftPos = h * s.hop
			rightPos = leftPos + 1
			return // Direct hit.
		} else if cmp < 0 {
			if i == h {
				break
			}
			i = h
		} else {
			j = h
		}
	}

	leftPos = i * s.hop
	rightPos = j * s.hop

	return
}
