// Package protoassert provides utility functions for comparing protobuf-generated objects in tests.
//
// These functions are based on related ones found in https://github.com/stretchr/testify.
package protoassert

import (
	"bufio"
	"bytes"
	"fmt"
	"slices"
	"testing"

	"maps"

	"github.com/davecgh/go-spew/spew"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/assert"
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
	if actual.EqualVT(expected) {
		return true
	}
	diff := diff(expected, actual)
	e, a := formatUnequalValues(expected, actual)
	return assert.Fail(t, fmt.Sprintf("Not equal: \n"+
		"expected: %s\n"+
		"actual  : %s%s", e, a, diff), msgAndArgs...)
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
	pass := true
	for i := range expected {
		pass = Equal(t, expected[i], actual[i], append([]any{fmt.Sprintf("index: %d\n", i)}, msgAndArgs...)...) && pass
	}
	return pass
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
	for i, e := range s {
		// Do not use [Equal] here, as it will unnecessarily log unequal elements.
		if e.EqualVT(contains) {
			return assert.Fail(t, fmt.Sprintf("index: %d\n%#v should not contain %#v", i, s, contains), msgAndArgs...)
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
	expectedKeys := slices.Collect(maps.Keys(expected))
	actualKeys := slices.Collect(maps.Keys(actual))
	if !assert.ElementsMatch(t, expectedKeys, actualKeys, append([]any{"keys differ:\n"}, msgAndArgs...)...) {
		return false
	}
	pass := true
	for key, expectedV := range expected {
		actualV := actual[key]
		pass = SlicesEqual(t, expectedV, actualV, append([]any{fmt.Sprintf("key: %v\n", key)}, msgAndArgs...)...) && pass
	}
	return pass
}

// MapEqual determines if the expected and actual maps are equal.
func MapEqual[M ~map[K]T, K comparable, T message[V], V any](t testing.TB, expected, actual M, msgAndArgs ...any) bool {
	t.Helper()
	expectedKeys := slices.Collect(maps.Keys(expected))
	actualKeys := slices.Collect(maps.Keys(actual))
	if !assert.ElementsMatch(t, expectedKeys, actualKeys, append([]any{"keys differ:\n"}, msgAndArgs...)...) {
		return false
	}
	pass := true
	for key, expectedV := range expected {
		actualV := actual[key]
		pass = Equal(t, expectedV, actualV, append([]any{fmt.Sprintf("key: %v\n", key)}, msgAndArgs...)...) && pass
	}
	return pass
}

// diff is based on [assert.diff]. The doc for it is copied below:
//
// diff returns a diff of both values as long as both are of the same type and
// are a struct, map, slice, array or string. Otherwise it returns an empty string.
func diff[T message[V], V any](expected, actual T) string {
	if expected == nil || actual == nil {
		return ""
	}

	e := spewConfig.Sdump(expected)
	a := spewConfig.Sdump(actual)

	diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(e),
		B:        difflib.SplitLines(a),
		FromFile: "Expected",
		FromDate: "",
		ToFile:   "Actual",
		ToDate:   "",
		Context:  1,
	})

	return "\n\nDiff:\n" + diff
}

// formatUnequalValues mimics [assert.formatUnequalValues]. The doc for it is copied below:
//
// formatUnequalValues takes two values of arbitrary types and returns string
// representations appropriate to be presented to the user.
//
// If the values are not of like type, the returned strings will be prefixed
// with the type name, and the value will be enclosed in parentheses similar
// to a type conversion in the Go grammar.
func formatUnequalValues[T message[V], V any](expected, actual T) (string, string) {
	return truncatingFormat(expected), truncatingFormat(actual)
}

// truncatingFormat mimics [assert.truncatingFormat]. The doc for it is copied below:
//
// truncatingFormat formats the data and truncates it if it's too long.
//
// This helps keep formatted error messages lines from exceeding the
// bufio.MaxScanTokenSize max line length that the go testing framework imposes.
func truncatingFormat(data any) string {
	value := fmt.Sprintf("%#v", data)
	max := bufio.MaxScanTokenSize - 100 // Give us some space the type info too if needed.
	if len(value) > max {
		value = value[0:max] + "<... truncated>"
	}
	return value
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
