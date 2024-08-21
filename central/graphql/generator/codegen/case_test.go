package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_camelCase(t *testing.T) {
	tests := map[string]string{
		"":                 "",
		"_my_field_name_2": "XMyFieldName_2",
		"my_field_name_2":  "MyFieldName_2",
		"My_field_name_2":  "MyFieldName_2",
		"My_Field_Name_2":  "My_Field_Name_2",
		"MyFieldName2":     "MyFieldName2",
		"__field_name__":   "XFieldName__",
	}
	for given, expected := range tests {
		t.Run(given, func(t *testing.T) {
			actual := camelCase(given)
			assert.Equal(t, expected, actual)
		})
	}
}
