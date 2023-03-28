package gjson

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/tidwall/gjson"
)

// CustomModifier is a type alias for a gjson.Modifier function used within gjson.AddModifier
type CustomModifier = func(json, arg string) string

var boolReplaceRegex = regexp.MustCompile(`.@boolReplace.*(})`)
var listReplaceRegex = regexp.MustCompile(`.@list`)
var textReplaceRegex = regexp.MustCompile(`.@text.*(})`)

// modifiersRegexp provides a list of regex expressions that match all custom modifier prefixes for
// sanitizing of queries.
func modifiersRegexp() []*regexp.Regexp {
	return []*regexp.Regexp{boolReplaceRegex, listReplaceRegex, textReplaceRegex}
}

var customGJSONModifiers = map[string]CustomModifier{
	"list":        ListModifier(),
	"boolReplace": BoolReplaceModifier(),
	"text":        TextModifier(),
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

func createListString(stringValues []string) string {
	if len(stringValues) == 0 {
		return ""
	}

	// Ensure that no trailing newline is added, as this previously caused table paddings to be all over the place.
	var sb strings.Builder
	sb.WriteString("- ")
	sb.WriteString(stringValues[0])
	if len(stringValues) >= 2 {
		for _, s := range stringValues[1:] {
			sb.WriteString("\n- ")
			sb.WriteString(s)
		}
	}
	bytes, _ := json.Marshal(sb.String())
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

type textOptions struct {
	PrintKeys       bool   `json:"printKeys"`
	CustomSeparator string `json:"customSeparator"`
}

func defaultTextOptions() *textOptions {
	return &textOptions{
		PrintKeys: true,
	}
}

// TextModifier provides the @text modifier for gjson which creates a single string from the result, which contains
// the key value pairs and a newline as separator.
func TextModifier() CustomModifier {
	return func(jsonString, arg string) string {
		opts := defaultTextOptions()
		if arg != "" {
			gjson.Parse(arg).ForEach(func(key, value gjson.Result) bool {
				switch key.String() {
				case "printKeys":
					opts.PrintKeys = value.Bool()
				case "customSeparator":
					opts.CustomSeparator = value.String()
				}
				return true
			})
		}
		modifier := resultToTextModifier{opts: opts}
		res := gjson.Parse(jsonString)
		texts := map[int]string{}
		res.ForEach(func(key, value gjson.Result) bool {
			toText(texts, key, value, modifier, 0)
			return true
		})
		// Ensure we keep the same order for the texts we generated.
		keys := maputil.Keys(texts)
		sort.Ints(keys)
		var result []string
		for _, key := range keys {
			result = append(result, modifier.trimSeparator(texts[key]))
		}
		bytes, _ := json.Marshal(result)
		return string(bytes)
	}
}

func toText(texts map[int]string, key gjson.Result, value gjson.Result, modifier resultToTextModifier, index int) int {
	if !value.IsArray() {
		texts[index] += modifier.resultToText(key, value)
		index++
		return index
	}
	for _, val := range value.Array() {
		if val.IsArray() {
			index = toText(texts, key, val, modifier, index)
			continue
		}
		texts[index] += modifier.resultToText(key, val)
		index++
	}
	return index
}

type resultToTextModifier struct {
	opts *textOptions
}

func (r *resultToTextModifier) resultToText(key, value gjson.Result) string {
	var sb strings.Builder

	if r.opts.PrintKeys {
		sb.WriteString(key.String())
		// Add a colon, and tab space between key and value.
		sb.WriteString(":\t")
	}
	sb.WriteString(value.String())
	sb.WriteString(utils.IfThenElse(r.opts.CustomSeparator != "", r.opts.CustomSeparator, "\n"))
	return sb.String()
}

func (r *resultToTextModifier) trimSeparator(s string) string {
	return strings.TrimSuffix(s, utils.IfThenElse(r.opts.CustomSeparator != "", r.opts.CustomSeparator, "\n"))
}
