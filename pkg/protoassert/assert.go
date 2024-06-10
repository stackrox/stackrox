package protoassert

import (
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func Equal(t *testing.T, expected, actual proto.Message, msgAndArgs ...interface{}) bool {
	t.Helper()
	e, err := toJson(expected)
	require.NoError(t, err)
	a, err := toJson(actual)
	require.NoError(t, err)
	return assert.JSONEq(t, e, a, msgAndArgs)
}

func SlicesEqual[T proto.Message](t *testing.T, expected, actual []T, msgAndArgs ...interface{}) bool {
	t.Helper()
	areEqual := assert.Len(t, actual, len(expected), msgAndArgs)
	for i, e := range expected {
		a := actual[i]
		areEqual = areEqual && Equal(t, a, e, msgAndArgs)
	}
	return areEqual
}

func MapSliceEqual[K comparable, T proto.Message](t *testing.T, expected, actual map[K][]T, msgAndArgs ...interface{}) bool {
	t.Helper()
	ek := maps.Keys(expected)
	ea := maps.Keys(actual)
	if !assert.ElementMatch(t, ek, ea, msgAndArgs) {
		return false
	}
	for k, v := range expected {
		a := actual[k]
		if !SlicesEqual(t, v, a, msgAndArgs) {
			return false
		}
	}
	return true
}

func toJson(m proto.Message) (string, error) {
	if m == nil {
		return "", nil
	}

	marshaller := &jsonpb.Marshaler{
		EnumsAsInts:  false,
		EmitDefaults: false,
		Indent:       "  ",
	}

	s, err := marshaller.MarshalToString(m)
	if err != nil {
		return "", err
	}

	return s, nil
}
