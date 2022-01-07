package gjson

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type modifierTestObject struct {
	List     []string                `json:"list"`
	BoolTest []boolReplaceTestObject `json:"boolTest"`
}

type boolReplaceTestObject struct {
	Bool bool `json:"bool"`
}

func TestListModifier(t *testing.T) {
	testExpression := "list.@list"

	testList := []string{"a", "b", "c", "d"}

	testObject := &modifierTestObject{
		List: testList,
	}

	expectedResult := `- a
- b
- c
- d`

	bytes, err := json.Marshal(testObject)
	require.NoError(t, err)

	res := gjson.GetBytes(bytes, testExpression)
	assert.Equal(t, expectedResult, res.String())
}

func TestBoolReplaceModifier(t *testing.T) {

	testObj := &modifierTestObject{
		BoolTest: []boolReplaceTestObject{
			{
				Bool: true,
			},
			{
				Bool: false,
			},
			{
				Bool: false,
			},
			{
				Bool: true,
			},
		},
		List: []string{"text with true", "text with false"},
	}

	bytes, err := json.Marshal(testObj)
	require.NoError(t, err)

	cases := map[string]struct {
		expression     string
		expectedResult string
	}{
		"should not replace anything with no config": {
			expression:     "boolTest.#.bool.@boolReplace",
			expectedResult: `["true","false","false","true"]`,
		},
		"should replace both true and false with configured values": {
			expression:     `boolTest.#.bool.@boolReplace:{"true": "x", "false": "-""}`,
			expectedResult: `["x","-","-","x"]`,
		},
		"should replace only true values": {
			expression:     `boolTest.#.bool.@boolReplace:{"true": "x"}`,
			expectedResult: `["x","false","false","x"]`,
		},
		"should replace only false values": {
			expression:     `boolTest.#.bool.@boolReplace:{"false": "-""}`,
			expectedResult: `["true","-","-","true"]`,
		},
		"should not replace non-bool values that contain true/false": {
			expression:     `list.#.@boolReplace:{"true":"x","false":"-"}`,
			expectedResult: `["\"text with true\"","\"text with false\""]`,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			res := gjson.GetBytes(bytes, c.expression)
			assert.Equal(t, c.expectedResult, res.String())
		})
	}
}
