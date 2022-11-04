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
	"sync"
)

// MergeOperatorStringAppend implements a simple merger that appends
// strings.  It was originally built for testing and sample purposes.
type MergeOperatorStringAppend struct {
	Sep        string // The separator string between operands.
	m          sync.Mutex
	numFull    int
	numPartial int
}

// Name returns the name of this merge operator implemenation
func (mo *MergeOperatorStringAppend) Name() string {
	return "MergeOperatorStringAppend"
}

// FullMerge performs the full merge of a string append operation
func (mo *MergeOperatorStringAppend) FullMerge(key, existingValue []byte,
	operands [][]byte) ([]byte, bool) {
	mo.m.Lock()
	mo.numFull++
	mo.m.Unlock()

	s := string(existingValue)
	for _, operand := range operands {
		s = s + mo.Sep + string(operand)
	}
	return []byte(s), true
}

// PartialMerge performs the partial merge of a string append operation
func (mo *MergeOperatorStringAppend) PartialMerge(key,
	leftOperand, rightOperand []byte) ([]byte, bool) {
	mo.m.Lock()
	mo.numPartial++
	mo.m.Unlock()

	return []byte(string(leftOperand) + mo.Sep + string(rightOperand)), true
}
