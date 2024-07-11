package set

import (
	"strconv"
	"testing"
)

var (
	a IntSet
	s StringSet
)

func BenchmarkIntSet(b *testing.B) {
	a = NewIntSet()
	for n := 0; n < b.N*1000; n++ {
		a.Add(n)
	}
	for n := 0; n < b.N*1000; n++ {
		a.Remove(n)
	}
}

func BenchmarkStringSet(b *testing.B) {
	s = NewStringSet()
	for n := 0; n < b.N*1000; n++ {
		s.Add(strconv.Itoa(n))
	}
	for n := 0; n < b.N*1000; n++ {
		s.Remove(strconv.Itoa(n))
	}
}
