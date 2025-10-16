package protoutils

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestSliceContains(t *testing.T) {
	cases := map[string]struct {
		msg      *storage.Role
		slice    []*storage.Role
		contains bool
	}{
		"empty slice should not contain role": {
			msg: storage.Role_builder{Name: "something"}.Build(),
		},
		"slice with len(0) should not contain role": {
			msg:   storage.Role_builder{Name: "something"}.Build(),
			slice: []*storage.Role{},
		},
		"slice with no matching elements should not contain role": {
			msg: storage.Role_builder{Name: "something"}.Build(),
			slice: []*storage.Role{
				storage.Role_builder{Name: "somewhere"}.Build(),
				storage.Role_builder{Name: "over"}.Build(),
				storage.Role_builder{Name: "the"}.Build(),
				storage.Role_builder{Name: "rainbow"}.Build(),
			},
		},
		"slice with matching elements should contain role": {
			msg: storage.Role_builder{Name: "something"}.Build(),
			slice: []*storage.Role{
				storage.Role_builder{Name: "somewhere"}.Build(),
				storage.Role_builder{Name: "over"}.Build(),
				storage.Role_builder{Name: "the"}.Build(),
				storage.Role_builder{Name: "rainbow"}.Build(),
				storage.Role_builder{Name: "something"}.Build(),
			},
			contains: true,
		},
		"slice with multiple matching elements should contain role": {
			msg: storage.Role_builder{Name: "something"}.Build(),
			slice: []*storage.Role{
				storage.Role_builder{Name: "somewhere"}.Build(),
				storage.Role_builder{Name: "over"}.Build(),
				storage.Role_builder{Name: "something"}.Build(),
				storage.Role_builder{Name: "the"}.Build(),
				storage.Role_builder{Name: "rainbow"}.Build(),
				storage.Role_builder{Name: "something"}.Build(),
			},
			contains: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.contains, SliceContains(tc.msg, tc.slice))
		})
	}
}

func TestSlicesEqual(t *testing.T) {
	cases := map[string]struct {
		first  []*storage.Role
		second []*storage.Role
		equal  bool
	}{
		"empty slices should be equal": {
			equal: true,
		},
		"slices with len 0 should be equal": {
			first:  []*storage.Role{},
			second: []*storage.Role{},
			equal:  true,
		},
		"empty and slice with len 0 should be equal": {
			first: []*storage.Role{},
			equal: true,
		},
		"arrays with different lengths should not be equal": {
			first: []*storage.Role{},
			second: []*storage.Role{
				storage.Role_builder{Name: "somewhere"}.Build(),
				storage.Role_builder{Name: "over"}.Build(),
				storage.Role_builder{Name: "the"}.Build(),
				storage.Role_builder{Name: "rainbow"}.Build(),
			},
		},
		"arrays with the same length and elements should be equal": {
			first: []*storage.Role{
				storage.Role_builder{Name: "somewhere"}.Build(),
				storage.Role_builder{Name: "over"}.Build(),
				storage.Role_builder{Name: "the"}.Build(),
				storage.Role_builder{Name: "rainbow"}.Build(),
			},
			second: []*storage.Role{
				storage.Role_builder{Name: "somewhere"}.Build(),
				storage.Role_builder{Name: "over"}.Build(),
				storage.Role_builder{Name: "the"}.Build(),
				storage.Role_builder{Name: "rainbow"}.Build(),
			},
			equal: true,
		},
		"arrays with the same length and elements in the wrong order should not be equal": {
			first: []*storage.Role{
				storage.Role_builder{Name: "somewhere"}.Build(),
				storage.Role_builder{Name: "over"}.Build(),
				storage.Role_builder{Name: "the"}.Build(),
				storage.Role_builder{Name: "rainbow"}.Build(),
			},
			second: []*storage.Role{
				storage.Role_builder{Name: "somewhere"}.Build(),
				storage.Role_builder{Name: "the"}.Build(),
				storage.Role_builder{Name: "rainbow"}.Build(),
				storage.Role_builder{Name: "over"}.Build(),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.equal, SlicesEqual(tc.first, tc.second))
		})
	}
}

func TestSliceUnique(t *testing.T) {
	cases := map[string]struct {
		slice  []*storage.Role
		unique []*storage.Role
	}{
		"empty slice should return empty slice": {},
		"slice of len 0 should return empty slice": {
			slice: []*storage.Role{},
		},
		"unique slice should return unique slice as-is": {
			slice: []*storage.Role{
				storage.Role_builder{Name: "somewhere"}.Build(),
				storage.Role_builder{Name: "over"}.Build(),
				storage.Role_builder{Name: "the"}.Build(),
				storage.Role_builder{Name: "rainbow"}.Build(),
			},
			unique: []*storage.Role{
				storage.Role_builder{Name: "somewhere"}.Build(),
				storage.Role_builder{Name: "over"}.Build(),
				storage.Role_builder{Name: "the"}.Build(),
				storage.Role_builder{Name: "rainbow"}.Build(),
			},
		},
		"slice with duplicate values should be removed": {
			slice: []*storage.Role{
				storage.Role_builder{Name: "somewhere"}.Build(),
				storage.Role_builder{Name: "over"}.Build(),
				storage.Role_builder{Name: "the"}.Build(),
				storage.Role_builder{Name: "rainbow"}.Build(),
				storage.Role_builder{Name: "somewhere"}.Build(),
				storage.Role_builder{Name: "over"}.Build(),
				storage.Role_builder{Name: "the"}.Build(),
				storage.Role_builder{Name: "rainbow"}.Build(),
			},
			unique: []*storage.Role{
				storage.Role_builder{Name: "somewhere"}.Build(),
				storage.Role_builder{Name: "over"}.Build(),
				storage.Role_builder{Name: "the"}.Build(),
				storage.Role_builder{Name: "rainbow"}.Build(),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			protoassert.SlicesEqual(t, tc.unique, SliceUnique(tc.slice))
		})
	}
}
