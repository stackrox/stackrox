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

	"github.com/blevesearch/mmap-go"
)

// mmapRef provides a ref-counting wrapper around a mmap handle.
type mmapRef struct {
	fref *FileRef
	mm   mmap.MMap
	buf  []byte
	m    sync.Mutex // Protects the fields that follow.
	refs int
	ext  interface{} // Extra user/associated data.
}

func (r *mmapRef) AddRef() *mmapRef {
	if r == nil {
		return nil
	}

	r.m.Lock()
	r.refs++
	r.m.Unlock()

	return r
}

func (r *mmapRef) DecRef() error {
	if r == nil {
		return nil
	}

	r.m.Lock()

	r.refs--
	if r.refs <= 0 {
		r.mm.Unmap()
		r.mm = nil

		r.buf = nil

		r.fref.DecRef()
		r.fref = nil
	}

	r.m.Unlock()

	return nil
}

func (r *mmapRef) Close() error {
	return r.DecRef()
}

func (r *mmapRef) SetExt(v interface{}) {
	r.m.Lock()
	r.ext = v
	r.m.Unlock()
}

func (r *mmapRef) GetExt() (v interface{}) {
	r.m.Lock()
	v = r.ext
	r.m.Unlock()
	return
}
