//  Copyright (c) 2017 Couchbase, Inc.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the
//  License. You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing,
//  software distributed under the License is distributed on an "AS
//  IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
//  express or implied. See the License for the specific language
//  governing permissions and limitations under the License.

package ghistogram

// HistogramMutator represents the subset of Histogram methods related
// to mutation operations.
type HistogramMutator interface {
	Add(dataPoint uint64, count uint64)
}

// histogramMutator implements the HistogramMutator interface for a
// given Histogram.
type histogramMutator struct {
	*Histogram // An anonymous field of type Histogram
}

// Add increases the count in the histogram bin for the given dataPoint.
func (h *histogramMutator) Add(dataPoint uint64, count uint64) {
	h.addUNLOCKED(dataPoint, count)
}
