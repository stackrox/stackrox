package gjson

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type modifierTestObject struct {
	List                   []string                `json:"list"`
	BoolTest               []boolReplaceTestObject `json:"boolTest"`
	TextTest               []textReplaceTestObject `json:"textTest"`
	TextTestMultipleArrays []testTextMultipleArray `json:"textTestMultipleArrays"`
}

type boolReplaceTestObject struct {
	Bool bool `json:"bool"`
}

type textReplaceTestObject struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Age     int    `json:"age"`
}

type testTextMultipleArray struct {
	TextTest []textReplaceTestObject `json:"textTest"`
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

func TestTextModifier(t *testing.T) {
	testObject := &modifierTestObject{
		TextTest: []textReplaceTestObject{
			{
				Name:    "Harry Potter",
				Address: "Privet Drive",
				Age:     18,
			},
			{
				Name:    "Ron Weasley",
				Address: "The Burrow",
				Age:     18,
			},
			{
				Name:    "Hermione Granger",
				Address: "Heathgate",
				Age:     19,
			},
		},
	}
	bytes, err := json.Marshal(testObject)
	require.NoError(t, err)

	cases := map[string]struct {
		expression string
		result     string
	}{
		"without custom column names": {
			expression: "{textTest.#.name,textTest.#.age,textTest.#.address}.@text",
			result:     "[\"name:\\tHarry Potter\\nage:\\t18\\naddress:\\tPrivet Drive\",\"name:\\tRon Weasley\\nage:\\t18\\naddress:\\tThe Burrow\",\"name:\\tHermione Granger\\nage:\\t19\\naddress:\\tHeathgate\"]",
		},
		"without modifier should not modify the output": {
			expression: "{textTest.#.name,textTest.#.age,textTest.#.address}",
			result:     "{\"name\":[\"Harry Potter\",\"Ron Weasley\",\"Hermione Granger\"],\"age\":[18,18,19],\"address\":[\"Privet Drive\",\"The Burrow\",\"Heathgate\"]}",
		},
		"with custom column names": {
			expression: "{\"Super Cool Name\":textTest.#.name,textTest.#.age,textTest.#.address}.@text",
			result:     "[\"Super Cool Name:\\tHarry Potter\\nage:\\t18\\naddress:\\tPrivet Drive\",\"Super Cool Name:\\tRon Weasley\\nage:\\t18\\naddress:\\tThe Burrow\",\"Super Cool Name:\\tHermione Granger\\nage:\\t19\\naddress:\\tHeathgate\"]",
		},
		"with singular value": {
			expression: "{textTest.0.name,textTest.0.age,textTest.0.address}.@text",
			result:     "[\"name:\\tHarry Potter\\nage:\\t18\\naddress:\\tPrivet Drive\"]",
		},
		"without printing keys": {
			expression: `{textTest.#.name,textTest.#.age,textTest.#.address}.@text:{"printKeys": "false"}`,
			result:     "[\"Harry Potter\\n18\\nPrivet Drive\",\"Ron Weasley\\n18\\nThe Burrow\",\"Hermione Granger\\n19\\nHeathgate\"]",
		},
		"with custom separator": {
			expression: `{textTest.#.name,textTest.#.age,textTest.#.address}.@text:{"customSeparator": "-"}`,
			result:     "[\"name:\\tHarry Potter-age:\\t18-address:\\tPrivet Drive\",\"name:\\tRon Weasley-age:\\t18-address:\\tThe Burrow\",\"name:\\tHermione Granger-age:\\t19-address:\\tHeathgate\"]",
		},
		"without printing keys and with custom separator": {
			expression: `{textTest.#.name,textTest.#.age,textTest.#.address}.@text:{"customSeparator": "-", "printKeys": "false"}`,
			result:     "[\"Harry Potter-18-Privet Drive\",\"Ron Weasley-18-The Burrow\",\"Hermione Granger-19-Heathgate\"]",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			res := gjson.GetBytes(bytes, c.expression)
			assert.Equal(t, c.result, res.String())
		})
	}
}

func TestTextModifier_MultipleArrays(t *testing.T) {
	testObjectWithMultipleArrays := &modifierTestObject{
		TextTestMultipleArrays: []testTextMultipleArray{
			{
				TextTest: []textReplaceTestObject{
					{
						Name:    "Harry Potter",
						Address: "Privet Drive",
						Age:     18,
					},
					{
						Name:    "Ron Weasley",
						Address: "The Burrow",
						Age:     18,
					},
					{
						Name:    "Hermione Granger",
						Address: "Heathgate",
						Age:     19,
					},
				},
			},
		},
	}

	bytes, err := json.Marshal(testObjectWithMultipleArrays)
	require.NoError(t, err)

	cases := map[string]struct {
		expression string
		result     string
	}{
		"without custom column names": {
			expression: "{textTestMultipleArrays.#.textTest.#.name,textTestMultipleArrays.#.textTest.#.age,textTestMultipleArrays.#.textTest.#.address}.@text",
			result:     "[\"name:\\tHarry Potter\\nage:\\t18\\naddress:\\tPrivet Drive\",\"name:\\tRon Weasley\\nage:\\t18\\naddress:\\tThe Burrow\",\"name:\\tHermione Granger\\nage:\\t19\\naddress:\\tHeathgate\"]",
		},
		"without modifier should not modify the output": {
			expression: "{textTestMultipleArrays.#.textTest.#.name,textTestMultipleArrays.#.textTest.#.age,textTestMultipleArrays.#.textTest.#.address}",
			result:     "{\"name\":[[\"Harry Potter\",\"Ron Weasley\",\"Hermione Granger\"]],\"age\":[[18,18,19]],\"address\":[[\"Privet Drive\",\"The Burrow\",\"Heathgate\"]]}",
		},
		"with custom column names": {
			expression: "{\"Super Cool Name\":textTestMultipleArrays.#.textTest.#.name,textTestMultipleArrays.#.textTest.#.age,textTestMultipleArrays.#.textTest.#.address}.@text",
			result:     "[\"Super Cool Name:\\tHarry Potter\\nage:\\t18\\naddress:\\tPrivet Drive\",\"Super Cool Name:\\tRon Weasley\\nage:\\t18\\naddress:\\tThe Burrow\",\"Super Cool Name:\\tHermione Granger\\nage:\\t19\\naddress:\\tHeathgate\"]",
		},
		"with singular value": {
			expression: "{textTestMultipleArrays.#.textTest.0.name,textTestMultipleArrays.#.textTest.0.age,textTestMultipleArrays.#.textTest.0.address}.@text",
			result:     "[\"name:\\tHarry Potter\\nage:\\t18\\naddress:\\tPrivet Drive\"]",
		},
		"without printing keys": {
			expression: `{textTestMultipleArrays.#.textTest.#.name,textTestMultipleArrays.#.textTest.#.age,textTestMultipleArrays.#.textTest.#.address}.@text:{"printKeys": "false"}`,
			result:     "[\"Harry Potter\\n18\\nPrivet Drive\",\"Ron Weasley\\n18\\nThe Burrow\",\"Hermione Granger\\n19\\nHeathgate\"]",
		},
		"with custom separator": {
			expression: `{textTestMultipleArrays.#.textTest.#.name,textTestMultipleArrays.#.textTest.#.age,textTestMultipleArrays.#.textTest.#.address}.@text:{"customSeparator": "-"}`,
			result:     "[\"name:\\tHarry Potter-age:\\t18-address:\\tPrivet Drive\",\"name:\\tRon Weasley-age:\\t18-address:\\tThe Burrow\",\"name:\\tHermione Granger-age:\\t19-address:\\tHeathgate\"]",
		},
		"without printing keys and with custom separator": {
			expression: `{textTestMultipleArrays.#.textTest.#.name,textTestMultipleArrays.#.textTest.#.age,textTestMultipleArrays.#.textTest.#.address}.@text:{"customSeparator": "-", "printKeys": "false"}`,
			result:     "[\"Harry Potter-18-Privet Drive\",\"Ron Weasley-18-The Burrow\",\"Hermione Granger-19-Heathgate\"]",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			res := gjson.GetBytes(bytes, c.expression)
			assert.Equal(t, c.result, res.String())
		})
	}
}
