package protoassert

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type message[V any] interface {
	*V
	proto.Message
	String() string
	EqualVT(other *V) bool
}

func Equal[V any, T message[V]](t testing.TB, expected, actual T, msgAndArgs ...interface{}) bool {
	t.Helper()
	if expected.EqualVT(actual) {
		return true
	}
	e, err := toJson(expected)
	require.NoError(t, err)
	a, err := toJson(actual)
	require.NoError(t, err)
	return assert.JSONEq(t, e, a, msgAndArgs)
}

func NotEqual[V any, T message[V]](t testing.TB, expected, actual T, msgAndArgs ...interface{}) bool {
	t.Helper()
	return assert.False(t, expected.EqualVT(actual), msgAndArgs)
}

func SlicesEqual[V any, T message[V]](t testing.TB, expected, actual []T, msgAndArgs ...interface{}) bool {
	t.Helper()
	areEqual := assert.Len(t, actual, len(expected))
	for i, e := range expected {
		a := actual[i]
		areEqual = Equal(t, a, e) && areEqual
	}
	if !areEqual {
		t.Log(msgAndArgs...)
	}
	return areEqual
}

func SliceContains[V any, T message[V]](t testing.TB, slice []T, element T, msgAndArgs ...interface{}) bool {
	t.Helper()
	for _, e := range slice {
		if proto.Equal(e, element) {
			return true
		}
	}
	return assert.Failf(t, "Slice does not contain element", "%q %v", element.String(), msgAndArgs)
}

func SliceNotContains[V any, T message[V]](t testing.TB, slice []T, element T, msgAndArgs ...interface{}) bool {
	t.Helper()
	for _, e := range slice {
		if proto.Equal(e, element) {
			return assert.Failf(t, "Slice contain element", "%q %v", element.String(), msgAndArgs)
		}
	}
	return true
}

func ElementsMatch[V any, T message[V]](t testing.TB, expected, actual []T, msgAndArgs ...interface{}) bool {
	t.Helper()
	areEqual := assert.Len(t, actual, len(expected))
	for _, e := range expected {
		areEqual = SliceContains(t, actual, e) && areEqual
	}
	for _, a := range actual {
		areEqual = SliceContains(t, expected, a) && areEqual
	}
	if !areEqual {
		t.Log(msgAndArgs...)
	}
	return areEqual
}

func MapSliceEqual[K comparable, V any, T message[V]](t testing.TB, expected, actual map[K][]T, msgAndArgs ...interface{}) bool {
	t.Helper()
	expectedKeys := maps.Keys(expected)
	actualKeys := maps.Keys(actual)
	areEqual := !assert.ElementsMatch(t, expectedKeys, actualKeys)
	for expectedKey, expectedValue := range expected {
		a := actual[expectedKey]
		areEqual = SlicesEqual(t, expectedValue, a, expectedKey) && areEqual
	}
	if !areEqual {
		t.Log(msgAndArgs...)
	}
	return areEqual
}

func MapEqual[K comparable, V any, T message[V]](t testing.TB, expected, actual map[K]T, msgAndArgs ...interface{}) bool {
	t.Helper()
	expectedKeys := maps.Keys(expected)
	actualKeys := maps.Keys(actual)
	areEqual := !assert.ElementsMatch(t, expectedKeys, actualKeys)
	for expectedKey, expectedValue := range expected {
		actualValue := actual[expectedKey]
		areEqual = Equal(t, expectedValue, actualValue, expectedKey) && areEqual
	}
	if !areEqual {
		t.Log(msgAndArgs...)
	}
	return areEqual
}

func toJson(m proto.Message) (string, error) {
	if m == nil {
		return "", nil
	}

	marshaller := &protojson.MarshalOptions{
		Indent: "  ",
	}

	s, err := marshaller.Marshal(m)
	if err != nil {
		return "", err
	}

	return string(s), nil
}
