package mapper

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
)

// CustomModifier is a type alias for a gjson.Modifier function used within gjson.AddModifier
type CustomModifier = func(json, arg string) string

var customGJSONModifiers = map[string]CustomModifier{
	"list":        ListModifier(),
	"boolReplace": BoolReplaceModifier(),
}

func init() {
	addCustomModifiersToGJSON(customGJSONModifiers)
}

func addCustomModifiersToGJSON(m map[string]CustomModifier) {
	for modifierName, modifierFunc := range m {
		gjson.AddModifier(modifierName, modifierFunc)
	}
}

// ListModifier provides the @list modifier for gjson which creates a single string representing a list from
// any array. Each string will be prefixed with "-" and have a trailing newline.
// No configuration options are currently available for the modifier
func ListModifier() CustomModifier {
	return func(json, arg string) string {
		res := gjson.Parse(json)
		stringList := getStringValuesFromNestedArrays(res, []string{})

		return createListString(stringList)
	}
}

func createListString(strings []string) string {
	listString := ""
	for _, s := range strings {
		listString += fmt.Sprintf("- %s\n", s)
	}
	bytes, _ := json.Marshal(listString)
	return string(bytes)
}

// BoolReplaceModifier provides the @boolReplace modifier for gjson which replaces "true" and "false" with the given
// strings, the configuration happens via a json object.
// The JSON object has the following structure:
// {"true": "string-you-want-to-replace-with", "false": "string-you-want-to-replace-with"}
func BoolReplaceModifier() CustomModifier {
	return func(j, arg string) string {
		opts := defaultReplaceBoolOptions()
		if arg != "" {
			gjson.Parse(arg).ForEach(func(key, value gjson.Result) bool {
				switch key.Bool() {
				case true:
					opts.TrueReplace = value.String()
				case false:
					opts.FalseReplace = value.String()
				}
				return true
			})
		}

		res := gjson.Parse(j)
		if res.Type == gjson.True {
			j = strings.ReplaceAll(j, "true", opts.TrueReplace)
		} else if res.Type == gjson.False {
			j = strings.ReplaceAll(j, "false", opts.FalseReplace)
		}
		bytes, _ := json.Marshal(j)
		return string(bytes)
	}
}

type replaceBoolOptions struct {
	TrueReplace  string `json:"true"`
	FalseReplace string `json:"false"`
}

func defaultReplaceBoolOptions() *replaceBoolOptions {
	return &replaceBoolOptions{
		TrueReplace:  "true",
		FalseReplace: "false",
	}
}
