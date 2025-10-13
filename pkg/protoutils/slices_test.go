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
			msg: &storage.Role{Name: "something"},
		},
		"slice with len(0) should not contain role": {
			msg:   &storage.Role{Name: "something"},
			slice: []*storage.Role{},
		},
		"slice with no matching elements should not contain role": {
			msg: &storage.Role{Name: "something"},
			slice: []*storage.Role{
				{Name: "somewhere"},
				{Name: "over"},
				{Name: "the"},
				{Name: "rainbow"},
			},
		},
		"slice with matching elements should contain role": {
			msg: &storage.Role{Name: "something"},
			slice: []*storage.Role{
				{Name: "somewhere"},
				{Name: "over"},
				{Name: "the"},
				{Name: "rainbow"},
				{Name: "something"},
			},
			contains: true,
		},
		"slice with multiple matching elements should contain role": {
			msg: &storage.Role{Name: "something"},
			slice: []*storage.Role{
				{Name: "somewhere"},
				{Name: "over"},
				{Name: "something"},
				{Name: "the"},
				{Name: "rainbow"},
				{Name: "something"},
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
				{Name: "somewhere"},
				{Name: "over"},
				{Name: "the"},
				{Name: "rainbow"},
			},
		},
		"arrays with the same length and elements should be equal": {
			first: []*storage.Role{
				{Name: "somewhere"},
				{Name: "over"},
				{Name: "the"},
				{Name: "rainbow"},
			},
			second: []*storage.Role{
				{Name: "somewhere"},
				{Name: "over"},
				{Name: "the"},
				{Name: "rainbow"},
			},
			equal: true,
		},
		"arrays with the same length and elements in the wrong order should not be equal": {
			first: []*storage.Role{
				{Name: "somewhere"},
				{Name: "over"},
				{Name: "the"},
				{Name: "rainbow"},
			},
			second: []*storage.Role{
				{Name: "somewhere"},
				{Name: "the"},
				{Name: "rainbow"},
				{Name: "over"},
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
				{Name: "somewhere"},
				{Name: "over"},
				{Name: "the"},
				{Name: "rainbow"},
			},
			unique: []*storage.Role{
				{Name: "somewhere"},
				{Name: "over"},
				{Name: "the"},
				{Name: "rainbow"},
			},
		},
		"slice with duplicate values should be removed": {
			slice: []*storage.Role{
				{Name: "somewhere"},
				{Name: "over"},
				{Name: "the"},
				{Name: "rainbow"},
				{Name: "somewhere"},
				{Name: "over"},
				{Name: "the"},
				{Name: "rainbow"},
			},
			unique: []*storage.Role{
				{Name: "somewhere"},
				{Name: "over"},
				{Name: "the"},
				{Name: "rainbow"},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			protoassert.SlicesEqual(t, tc.unique, SliceUnique(tc.slice))
		})
	}
}
