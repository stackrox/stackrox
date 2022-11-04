//  Copyright (c) 2014 Couchbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 		http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package collector

import (
	"sort"

	"github.com/blevesearch/bleve/search"
)

type collectStoreRawSlice struct {
	slice   search.DocumentMatchCollection
	compare collectorCompare
}

func newStoreRawSlice(capacity int, compare collectorCompare) *collectStoreRawSlice {
	rv := &collectStoreRawSlice{
		slice:   make(search.DocumentMatchCollection, 0, capacity),
		compare: compare,
	}
	return rv
}

func (c *collectStoreRawSlice) AddNotExceedingSize(doc *search.DocumentMatch, size int) *search.DocumentMatch {
	c.slice = append(c.slice, doc)
	return nil
}

func (c *collectStoreRawSlice) Final(skip int, fixup collectorFixup) (search.DocumentMatchCollection, error) {
	if skip >= len(c.slice) {
		return search.DocumentMatchCollection{}, nil
	}
	sort.Slice(c.slice, func(i, j int) bool {
		return c.compare(c.slice[i], c.slice[j]) < 0
	})
	c.slice = c.slice[skip:]
	for _, doc := range c.slice {
		err := fixup(doc)
		if err != nil {
			return nil, err
		}
	}
	return c.slice, nil
}

func (c *collectStoreRawSlice) len() int {
	return len(c.slice)
}
