package sliceutils

import (
	"reflect"
	"testing"
)

func TestCopySliceSorted(t *testing.T) {
	tests := []struct {
		name     string
		input    []int
		expected []int
	}{
		{"Empty slice", []int{}, []int{}},
		{"Single element", []int{42}, []int{42}},
		{"Already sorted", []int{1, 2, 3}, []int{1, 2, 3}},
		{"Unsorted slice", []int{3, 1, 2}, []int{1, 2, 3}},
		{"Duplicate elements", []int{4, 2, 2, 3}, []int{2, 2, 3, 4}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CopySliceSorted(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("CopySliceSorted(%v) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCopySliceSortedWithStrings(t *testing.T) {
	input := []string{"c", "d", "a", "b", "b"}
	expected := []string{"a", "b", "b", "c", "d"}
	got := CopySliceSorted(input)

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("CopySliceSorted(%v) = %v; want %v", input, got, expected)
	}
}
