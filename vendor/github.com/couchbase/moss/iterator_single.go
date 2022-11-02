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
	"io"
)

// An iteratorSingle implements the Iterator interface, and is an edge
// case optimization when there's only a single segment to iterate and
// there's no lower-level iterator.  In contrast to the main iterator
// implementation, iteratorSingle doesn't have any heap operations.
type iteratorSingle struct {
	s  *segment
	sc SegmentCursor

	op uint64
	k  []byte
	v  []byte

	closer io.Closer

	options *CollectionOptions

	iteratorOptions IteratorOptions
}

// Close must be invoked to release resources.
func (iter *iteratorSingle) Close() error {
	if iter.closer != nil {
		iter.closer.Close()
		iter.closer = nil
	}

	return nil
}

func (iter *iteratorSingle) InitCloser(closer io.Closer) error {
	if iter.closer != nil {
		return ErrAlreadyInitialized
	}
	iter.closer = closer
	return nil
}

// Next returns ErrIteratorDone if the iterator is done.
func (iter *iteratorSingle) Next() error {
	err := iter.sc.Next()
	if err != nil {
		iter.op = 0
		iter.k = nil
		iter.v = nil

		// we DO want to return ErrIteratorDone here
		return err
	}

	iter.op, iter.k, iter.v = iter.sc.Current()
	if iter.op != OperationDel ||
		iter.iteratorOptions.IncludeDeletions {
		return nil
	}

	return iter.Next()
}

func (iter *iteratorSingle) SeekTo(seekToKey []byte) error {
	key, _, err := iter.Current()
	if err != nil && err != ErrIteratorDone {
		return err
	}

	if key != nil {
		cmp := bytes.Compare(seekToKey, key)
		if cmp == 0 {
			return nil
		}

		if cmp > 0 {
			// Try a loop of naive Next()'s for several attempts.
			err = naiveSeekTo(iter, seekToKey, DefaultNaiveSeekToMaxTries)
			if err != ErrMaxTries {
				return err
			}
		}
	}

	iter.op = 0
	iter.k = nil
	iter.v = nil

	err = iter.sc.Seek(seekToKey)
	if err != nil {
		// we DO want to return ErrIteratorDone here
		return err
	}

	iter.op, iter.k, iter.v = iter.sc.Current()
	if !iter.iteratorOptions.IncludeDeletions &&
		iter.op == OperationDel {
		return iter.Next()
	}

	return nil
}

// Current returns ErrIteratorDone if the iterator is done.
// Otherwise, Current() returns the current key and val, which should
// be treated as immutable or read-only.  The key and val bytes will
// remain available until the next call to Next() or Close().
func (iter *iteratorSingle) Current() ([]byte, []byte, error) {
	if iter.op == 0 {
		return nil, nil, ErrIteratorDone
	}

	if iter.op == OperationDel {
		return nil, nil, nil
	}

	if iter.op == OperationMerge {
		var mo MergeOperator
		if iter.options != nil {
			mo = iter.options.MergeOperator
		}
		if mo == nil {
			return iter.k, nil, ErrMergeOperatorNil
		}

		vMerged, ok := mo.FullMerge(iter.k, nil, [][]byte{iter.v})
		if !ok {
			return iter.k, nil, ErrMergeOperatorFullMergeFailed
		}

		return iter.k, vMerged, nil
	}

	return iter.k, iter.v, nil
}

// CurrentEx is a more advanced form of Current() that returns more
// metadata.  It returns ErrIteratorDone if the iterator is done.
// Otherwise, the current operation, key, val are returned.
func (iter *iteratorSingle) CurrentEx() (
	entryEx EntryEx, key, val []byte, err error) {
	if iter.op == 0 {
		return EntryEx{}, nil, nil, ErrIteratorDone
	}

	return EntryEx{Operation: iter.op}, iter.k, iter.v, nil
}
