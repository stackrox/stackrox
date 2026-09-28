package a

import "github.com/stackrox/rox/pkg/set"

func frozenSetInRange() {
	fs := set.NewFrozenSet("a", "b", "c")
	for _, x := range fs.AsSlice() { // want `use \.All\(\) instead of \.AsSlice\(\) in range loops to avoid allocation`
		_ = x
	}
}

func frozenSetAllOK() {
	fs := set.NewFrozenSet("a", "b", "c")
	for x := range fs.All() {
		_ = x
	}
}

func frozenSetAsSliceNotInRange() {
	fs := set.NewFrozenSet("a", "b", "c")
	_ = fs.AsSlice()
}

func mutableSetInRange() {
	s := make(set.StringSet)
	for _, x := range s.AsSlice() { // want `range over the set directly instead of calling \.AsSlice\(\)`
		_ = x
	}
}

func mutableSetDirectRangeOK() {
	s := make(set.StringSet)
	for x := range s {
		_ = x
	}
}

func frozenStringSetAlias() {
	fs := set.NewFrozenSet("a", "b")
	var fss set.FrozenStringSet = fs
	for _, x := range fss.AsSlice() { // want `use \.All\(\) instead of \.AsSlice\(\) in range loops to avoid allocation`
		_ = x
	}
}
