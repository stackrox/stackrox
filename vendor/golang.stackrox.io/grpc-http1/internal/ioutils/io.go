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

import "io"

// CopyNFull does the same as io.CopyN, but it returns io.ErrUnexpectedEOF
// if CopyN returns io.EOF and the number of bytes written greater than zero.
func CopyNFull(dst io.Writer, src io.Reader, n int64) (int64, error) {
	written, err := io.CopyN(dst, src, n)
	if err == io.EOF && written != 0 {
		err = io.ErrUnexpectedEOF
	}

	return written, err
}
