// +build static

package gorocksdb

// #cgo LDFLAGS: -l:librocksdb.a -l:libstdc++.a -l:libz.a -l:libbz2.a -l:libsnappy.a -l:liblz4.a -l:libzstd.a -lm -ldl
import "C"
