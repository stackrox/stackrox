package boltdb

import (
	"github.com/deckarep/golang-set"
)

func newStringSet(strs []string) mapset.Set {
	set := mapset.NewSet()
	for _, s := range strs {
		set.Add(s)
	}
	return set
}
