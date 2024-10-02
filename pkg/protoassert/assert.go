// Package protoassert provides utility functions for comparing protobuf-generated objects in tests.
//
// These functions are based on related ones found in https://github.com/stretchr/testify.
package protoassert

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/proto"
)

type message[V any] interface {
	*V
	proto.Message
	fmt.Stringer
	EqualVT(that *V) bool
}

var spewConfig = spew.ConfigState{
	Indent:                  " ",
	DisablePointerAddresses: true,
	DisableCapacities:       true,
	SortKeys:                true,
	DisableMethods:          true,
	MaxDepth:                10,
}

// Equal mimics [assert.Equal].
func Equal[T message[V], V any](t testing.TB, expected, actual T, msgAndArgs ...any) bool {
	t.Helper()
	if !actual.EqualVT(expected) {
		return assert.Fail(t, fmt.Sprintf("Not equal: \n"+
			"expected: %s\n"+
			"actual  : %s", expected, actual), msgAndArgs...)
	}
	return true
}

// NotEqual mimics [assert.NotEqual].
func NotEqual[T message[V], V any](t testing.TB, expected, actual T, msgAndArgs ...any) bool {
	t.Helper()
	if actual.EqualVT(expected) {
		return assert.Fail(t, fmt.Sprintf("Should not be: %#v\n", actual), msgAndArgs...)
	}
	return true
}

// SlicesEqual determines if the expected and actual slices are equal.
func SlicesEqual[S ~[]T, T message[V], V any](t testing.TB, expected, actual S, msgAndArgs ...any) bool {
	t.Helper()
	if !assert.Len(t, actual, len(expected), msgAndArgs...) {
		return false
	}
	for i := range expected {
		if !Equal(t, expected[i], actual[i], msgAndArgs...) {
			return false
		}
	}
	return true
}

// SliceContains mimics [assert.Contains].
func SliceContains[S ~[]T, T message[V], V any](t testing.TB, s S, contains T, msgAndArgs ...any) bool {
	t.Helper()
	for _, e := range s {
		// Do not use [Equal] here, as it will unnecessarily log unequal elements.
		if e.EqualVT(contains) {
			return true
		}
	}
	return assert.Fail(t, fmt.Sprintf("%#v does not contain %#v", s, contains), msgAndArgs...)
}

// SliceNotContains mimics [assert.NotContains].
func SliceNotContains[S ~[]T, T message[V], V any](t testing.TB, s S, contains T, msgAndArgs ...any) bool {
	t.Helper()
	for _, e := range s {
		// Do not use [Equal] here, as it will unnecessarily log unequal elements.
		if e.EqualVT(contains) {
			return assert.Fail(t, fmt.Sprintf("%#v should not contain %#v", s, contains), msgAndArgs...)
		}
	}
	return true
}

// ElementsMatch mimics [assert.ElementsMatch].
func ElementsMatch[S ~[]T, T message[V], V any](t testing.TB, expected, actual S, msgAndArgs ...any) bool {
	t.Helper()
	if len(expected) == 0 && len(actual) == 0 {
		return true
	}
	extraExpected, extraActual := diffSlices(expected, actual)
	if len(extraExpected) == 0 && len(extraActual) == 0 {
		return true
	}
	return assert.Fail(t, formatSliceDiff(expected, actual, extraExpected, extraActual), msgAndArgs...)
}

// MapSliceEqual determines if the expected and actual maps (from key to slice) are equal.
func MapSliceEqual[M ~map[K][]T, K comparable, T message[V], V any](t testing.TB, expected, actual M, msgAndArgs ...any) bool {
	t.Helper()
	expectedKeys := maps.Keys(expected)
	actualKeys := maps.Keys(actual)
	if !assert.ElementsMatch(t, expectedKeys, actualKeys, msgAndArgs...) {
		return false
	}
	for key, expectedV := range expected {
		actualV := actual[key]
		if !SlicesEqual(t, expectedV, actualV, msgAndArgs...) {
			return false
		}
	}
	return true
}

// MapEqual determines if the expected and actual maps are equal.
func MapEqual[M ~map[K]T, K comparable, T message[V], V any](t testing.TB, expected, actual M, msgAndArgs ...any) bool {
	t.Helper()
	expectedKeys := maps.Keys(expected)
	actualKeys := maps.Keys(actual)
	if !assert.ElementsMatch(t, expectedKeys, actualKeys, msgAndArgs...) {
		return false
	}
	for key, expectedV := range expected {
		actualV := actual[key]
		if !Equal(t, expectedV, actualV, msgAndArgs...) {
			return false
		}
	}
	return true
}

// diffSlices is based on [assert.diffLists]. The doc for it is copied below:
//
// diffLists diffs two arrays/slices and returns slices of elements that are only in A and only in B.
// If some element is present multiple times, each instance is counted separately (e.g. if something is 2x in A and
// 5x in B, it will be 0x in extraA and 3x in extraB). The order of items in both lists is ignored.
func diffSlices[V any, T message[V]](a, b []T) (extraA, extraB []T) {
	aLen, bLen := len(a), len(b)

	visited := make([]bool, bLen)
	for i := 0; i < aLen; i++ {
		element := a[i]
		found := false
		for j := 0; j < bLen; j++ {
			if visited[j] {
				continue
			}
			if b[j].EqualVT(element) {
				visited[j] = true
				found = true
				break
			}
		}
		if !found {
			extraA = append(extraA, element)
		}
	}

	for j := 0; j < bLen; j++ {
		if visited[j] {
			continue
		}
		extraB = append(extraB, b[j])
	}

	return
}

// formatSliceDiff is based on [assert.formatListDiff].
func formatSliceDiff[V any, T message[V]](sliceA, sliceB, extraA, extraB []T) string {
	var msg bytes.Buffer

	msg.WriteString("elements differ")
	if len(extraA) > 0 {
		msg.WriteString("\n\nextra elements in expected slice:\n")
		msg.WriteString(spewConfig.Sdump(extraA))
	}
	if len(extraB) > 0 {
		msg.WriteString("\n\nextra elements in actual slice:\n")
		msg.WriteString(spewConfig.Sdump(extraB))
	}
	msg.WriteString("\n\nexpected slice:\n")
	msg.WriteString(spewConfig.Sdump(sliceA))
	msg.WriteString("\n\nactual slice:\n")
	msg.WriteString(spewConfig.Sdump(sliceB))

	return msg.String()
}
