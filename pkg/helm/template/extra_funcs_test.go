package template

import (
	"testing"
	"text/template"

	"github.com/stackrox/stackrox/pkg/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testTemplate = template.Must(InitTemplate("test").Parse(`{{ .OptionalValue }} / {{ required "" .MandatoryValue }}`))

type testDataStruct struct {
	OptionalValue  string
	MandatoryValue string
}

func TestRequiredStrongTyped(t *testing.T) {
	casesStruct := map[string]testDataStruct{
		"optional-in-struct / mandatory-in-struct": {
			OptionalValue:  "optional-in-struct",
			MandatoryValue: "mandatory-in-struct",
		},
		" / mandatory-in-struct": {
			MandatoryValue: "mandatory-in-struct",
		},
	}

	for expected, inputStruct := range casesStruct {
		t.Run(expected, func(t *testing.T) {
			executeAndAssertResult(t, inputStruct, expected)
		})
	}
}

func TestRequiredStringMap(t *testing.T) {
	casesMap := map[string]map[string]string{
		"optional-in-map / mandatory-in-map": {
			"OptionalValue":  "optional-in-map",
			"MandatoryValue": "mandatory-in-map",
		},
		" / mandatory-in-map": {
			"OptionalValue":  "",
			"MandatoryValue": "mandatory-in-map",
		},
		// This output might seem a bit surprising but that's how the templating works with absent keys in map...
		"<no value> / mandatory-in-map": {
			"MandatoryValue": "mandatory-in-map",
		},
	}

	for expected, inputMap := range casesMap {
		t.Run(expected, func(t *testing.T) {
			executeAndAssertResult(t, inputMap, expected)
		})
	}
}

func executeAndAssertResult(t *testing.T, data interface{}, expectedResult string) {
	result, err := templates.ExecuteToString(testTemplate, data)
	require.NoError(t, err)
	assert.Equal(t, expectedResult, result)
}

func TestRequiredError(t *testing.T) {
	cases := map[string]struct {
		tpl                    *template.Template
		data                   interface{}
		expectedErrorFragments []string
	}{
		"default message and struct": {
			tpl:                    testTemplate,
			data:                   testDataStruct{},
			expectedErrorFragments: []string{"MandatoryValue", "required value was not specified"},
		},
		"default message and map": {
			tpl:                    testTemplate,
			data:                   map[string]string{},
			expectedErrorFragments: []string{"MandatoryValue", "required value was not specified"},
		},
		"custom message": {
			tpl:                    template.Must(InitTemplate("test").Parse(`{{ required "You must obey and give me what I want" .MandatoryValue }}`)),
			data:                   testDataStruct{},
			expectedErrorFragments: []string{"MandatoryValue", "You must obey and give me what I want"},
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := templates.ExecuteToString(c.tpl, c.data)
			assert.Empty(t, result)
			assert.Error(t, err)
			for _, errorFragment := range c.expectedErrorFragments {
				assert.Contains(t, err.Error(), errorFragment)
			}
		})
	}
}
