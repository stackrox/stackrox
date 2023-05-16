package protoutils

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stackrox/rox/generated/test"
	test3 "github.com/stackrox/rox/pkg/protoconvert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertEquality(t *testing.T, t1, t2 interface{}) {
	t1Str, err := json.Marshal(t1)
	require.NoError(t, err)

	t2Str, err := json.Marshal(t2)
	require.NoError(t, err)

	assert.Equal(t, t1Str, t2Str)
}

func TestConvert(t *testing.T) {
	t1 := &test.TestClone{}
	t2 := test3.ConvertTestTestCloneToTest2TestClone(t1)
	assertEquality(t, t1, t2)

	originalT1 := getFilledStruct()
	t2 = test3.ConvertTestTestCloneToTest2TestClone(originalT1)
	assertEquality(t, originalT1, t2)

	convertedT1 := test3.ConvertTest2TestCloneToTestTestClone(t2)
	assertEquality(t, originalT1, convertedT1)
}

func TestConvertCheckAliasing(t *testing.T) {
	obj := getFilledStruct()
	convertedObj := test3.ConvertTestTestCloneToTest2TestClone(obj)

	checkAliasRecursive(t, reflect.ValueOf(obj), reflect.ValueOf(convertedObj))
}
