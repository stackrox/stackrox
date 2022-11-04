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

// calcTargetTopLevel() heuristically computes a new top level that
// the segmentStack should be merged to.
func (ss *segmentStack) calcTargetTopLevel() int {
	var minMergePercentage float64
	if ss.options != nil {
		minMergePercentage = ss.options.MinMergePercentage
	}
	if minMergePercentage <= 0 {
		minMergePercentage = DefaultCollectionOptions.MinMergePercentage
	}

	newTopLevel := 0
	maxTopLevel := len(ss.a) - 2

	for newTopLevel < maxTopLevel {
		numX0 := ss.a[newTopLevel].Len()
		numX1 := ss.a[newTopLevel+1].Len()
		if (float64(numX1) / float64(numX0)) > minMergePercentage {
			break
		}

		newTopLevel++
	}

	return newTopLevel
}

// ------------------------------------------------------

// merge() returns a new segmentStack, merging all the segments that
// are at the given newTopLevel and higher.
func (ss *segmentStack) merge(mergeAll bool, base *segmentStack) (
	*segmentStack, uint64, error) {
	newTopLevel := 0
	var numFullMerges uint64
	if !mergeAll {
		// If we have not been asked to merge all segments,
		// then heuristically calc a newTopLevel.
		newTopLevel = ss.calcTargetTopLevel()
	}
	if newTopLevel <= 0 {
		numFullMerges++
	}

	// ----------------------------------------------------
	// First, rough estimate the bytes neeeded.

	var totOps int
	var totKeyBytes, totValBytes uint64
	for i := newTopLevel; i < len(ss.a); i++ {
		totOps += ss.a[i].Len()
		nk, nv := ss.a[i].NumKeyValBytes()
		totKeyBytes += nk
		totValBytes += nv
	}

	// ----------------------------------------------------
	// Next, use an iterator for the actual merge.

	mergedSegment, err := newSegment(totOps, int(totKeyBytes+totValBytes))
	if err != nil {
		return nil, 0, err
	}

	err = ss.mergeInto(newTopLevel, len(ss.a), mergedSegment, base, true,
		true, nil)
	if err != nil {
		return nil, 0, err
	}

	a := make([]Segment, 0, newTopLevel+1)
	a = append(a, ss.a[0:newTopLevel]...)
	a = append(a, mergedSegment)

	rv := &segmentStack{
		options:            ss.options,
		stats:              ss.stats,
		a:                  a,
		refs:               1,
		lowerLevelSnapshot: ss.lowerLevelSnapshot.addRef(),
		incarNum:           ss.incarNum,
	}

	// ---------------------------------------------------
	// Recursively merge all the child segmentStacks with the base
	// stack, dropping any deleted collections present in base but not
	// in me.

	for cName, childSegStack := range ss.childSegStacks {
		var baseSegStack *segmentStack
		if base != nil {
			var exists bool
			if base.childSegStacks != nil {
				baseSegStack, exists = base.childSegStacks[cName]
			}
			if exists {
				if baseSegStack.incarNum != childSegStack.incarNum {
					// The base segment stack carries a child collection
					// which was subsequently recreated.
					baseSegStack = nil
					// The dirtyBase's old segmentStacks will be closed by
					// the collection merger after successful merge.
				}
			}
		}

		mergedChildStack, fullMerges, err :=
			childSegStack.merge(mergeAll, baseSegStack)
		if err != nil {
			rv.Close()
			return nil, numFullMerges, err
		}

		numFullMerges += fullMerges

		if len(rv.childSegStacks) == 0 {
			rv.childSegStacks = make(map[string]*segmentStack)
		}
		rv.childSegStacks[cName] = mergedChildStack
	}

	return rv, numFullMerges, nil
}

func (ss *segmentStack) mergeInto(minSegmentLevel, maxSegmentHeight int,
	dest SegmentMutator, base *segmentStack, includeDeletions, optimizeTail bool,
	cancelCh chan struct{}) error {
	cancelCheckEvery := ss.options.MergerCancelCheckEvery
	if cancelCheckEvery <= 0 {
		cancelCheckEvery = DefaultCollectionOptions.MergerCancelCheckEvery
	}

	iter, err := ss.startIterator(nil, nil, IteratorOptions{
		IncludeDeletions: includeDeletions,
		SkipLowerLevel:   true,
		MinSegmentLevel:  minSegmentLevel,
		MaxSegmentHeight: maxSegmentHeight,
		base:             base,
	})
	if err != nil {
		return err
	}

	defer iter.Close()

	readOptions := ReadOptions{NoCopyValue: true}

OUTER:
	for i := 0; true; i++ {
		if cancelCh != nil && i%cancelCheckEvery == 0 {
			select {
			case <-cancelCh:
				return ErrAborted
			default:
				// NO-OP.
			}
		}

		entryEx, key, val, err := iter.CurrentEx()
		if err == ErrIteratorDone {
			break
		}
		if err != nil {
			return err
		}

		if optimizeTail && len(iter.cursors) == 1 {
			// When only 1 cursor remains, copy the remains of the
			// last segment more directly instead of Next()'ing
			// through the iterator.
			cursor := iter.cursors[0]

			var op uint64
			var k, v []byte
			op, k, v = cursor.sc.Current()
			for op != 0 {
				err = dest.Mutate(op, k, v)
				if err != nil {
					return err
				}
				err = cursor.sc.Next()
				if err != nil {
					break
				}
				op, k, v = cursor.sc.Current()
			}
			if err != nil && err != ErrIteratorDone {
				return err
			}

			break OUTER
		}

		op := entryEx.Operation
		if op == OperationMerge {
			// TODO: the merge operator implementation is currently
			// inefficient and not lazy enough right now.
			val, err = ss.get(key, len(ss.a)-1, base, readOptions)
			if err != nil {
				return err
			}

			if val == nil {
				op = OperationDel
			} else {
				op = OperationSet
			}
		}

		err = dest.Mutate(op, key, val)
		if err != nil {
			return err
		}

		err = iter.Next()
		if err == ErrIteratorDone {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}
