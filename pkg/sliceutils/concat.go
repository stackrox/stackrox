// Copyright 2021 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sliceutils

import "slices"

// Concat returns a new slice concatenating the passed in slices.
//
// This is directly copied from the implementation of https://pkg.go.dev/slices#Concat
// as of go1.22.2.
//
// This may be removed from the repository once go1.22 becomes the minimum required version.
func Concat[S ~[]E, E any](slcs ...S) S {
	size := 0
	for _, s := range slcs {
		size += len(s)
		if size < 0 {
			panic("len out of range")
		}
	}
	newslice := slices.Grow[S](nil, size)
	for _, s := range slcs {
		newslice = append(newslice, s...)
	}
	return newslice
}
