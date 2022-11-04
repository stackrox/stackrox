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

import (
	"errors"
	"io"
	"strings"
)

// Histograms represents a map of histograms identified by
// unique names (string).
type Histograms map[string]*Histogram

// API that converts the contents of all histograms within
// the map into a string and returns the string to caller.
func (hmap Histograms) String() string {
	var output []string

	for _, v := range hmap {
		output = append(output, v.EmitGraph(nil, nil).String())
	}

	return strings.Join(output, "\n")
}

// Emits the ASCII graphs of all histograms held within
// the map through the provided writer.
func (hmap Histograms) Fprint(w io.Writer) (int, error) {
	wrote, err := w.Write([]byte(hmap.String()))
	return wrote, err
}

// Adds all entries/records from all histograms within the
// given map, to all histograms in the current map.
// If a histogram from the source doesn't exist in the
// destination map, it will be created first.
func (hmap Histograms) AddAll(srcmap Histograms) error {
	for k, v := range srcmap {
		if hmap[k] == nil {
			// Histogram entry not found, create a new one, based
			// on the same creation parameters
			hmap[k] = v.CloneEmpty()
		} else if (len(hmap[k].Counts) != len(v.Counts)) ||
			(len(hmap[k].Ranges) != len(v.Ranges)) {
			return errors.New("Mismatch in histogram creation parameters")
		} else {
			for i := 0; i < len(v.Ranges); i++ {
				if hmap[k].Ranges[i] != v.Ranges[i] {
					return errors.New("Mismatch in histogram creation parmeters")
				}
			}
		}
	}

	for k, v := range srcmap {
		hmap[k].AddAll(v)
	}

	return nil
}
