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

// +build safe

package moss

import (
	"bytes"
	"encoding/binary"
)

// Uint64SliceToByteSlice gives access to []uint64 as []byte
func Uint64SliceToByteSlice(in []uint64) ([]byte, error) {
	buffer := bytes.NewBuffer(make([]byte, 0, len(in)*8))
	err := binary.Write(buffer, STORE_ENDIAN, in)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// ByteSliceToUint64Slice gives access to []byte as []uint64
func ByteSliceToUint64Slice(in []byte) ([]uint64, error) {
	buffer := bytes.NewBuffer(in)

	out := make([]uint64, len(in)/8)
	err := binary.Read(buffer, STORE_ENDIAN, &out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// --------------------------------------------------------------

func endian() string {
	return "unknown" // Need unsafe package to tell endian'ess.
}
