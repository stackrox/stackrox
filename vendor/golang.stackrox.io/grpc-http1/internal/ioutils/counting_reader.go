// Copyright (c) 2020 StackRox Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License

package ioutils

import (
	"io"
	"sync/atomic"
)

type countingReader struct {
	reader io.Reader
	count  *int64
}

// NewCountingReader wraps the given reader in a reader that ensures the given count variable is atomically updated
// whenever data is read.
func NewCountingReader(reader io.Reader, count *int64) io.ReadCloser {
	return &countingReader{
		reader: reader,
		count:  count,
	}
}

func (r *countingReader) Close() error {
	if rc, _ := r.reader.(io.ReadCloser); rc != nil {
		return rc.Close()
	}
	return nil
}

func (r *countingReader) Read(buf []byte) (int, error) {
	n, err := r.reader.Read(buf)
	atomic.AddInt64(r.count, int64(n))
	return n, err
}
