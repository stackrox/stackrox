package main

import (
	"fmt"
	"reflect"
	"strings"
	"text/template"
	"unicode"
)

func splitWords(s string) []string {
	var words []string
	var currWord strings.Builder

	for _, r := range s {
		newWord, skip := false, false
		if unicode.IsUpper(r) && currWord.Len() > 0 {
			newWord = true
		} else if !unicode.IsOneOf([]*unicode.RangeTable{unicode.Letter, unicode.Digit}, r) {
			newWord, skip = true, true
		}

		if newWord && currWord.Len() > 0 {
			words = append(words, currWord.String())
			currWord.Reset()
		}
		if !skip {
			currWord.WriteRune(r)
		}
	}

	if currWord.Len() > 0 {
		words = append(words, currWord.String())
	}

	return words
}

func applyPointwise(ss []string, f func(string) string) {
	for i, s := range ss {
		ss[i] = f(s)
	}
}

func lowerCamelCase(s string) string {
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}
	words[0] = strings.ToLower(words[0])
	applyPointwise(words[1:], strings.Title)
	return strings.Join(words, "")
}

func upperCamelCase(s string) string {
	words := splitWords(s)
	if len(words) == 0 {
		return ""
	}
	applyPointwise(words, strings.Title)
	return strings.Join(words, "")
}

func elemComma(idx int, slice interface{}) string {
	enumSlice := reflect.ValueOf(slice)
	enumSliceLen := enumSlice.Len()
	if idx == enumSliceLen-1 {
		return ""
	}
	return ","
}

func valueExpansion(new, starting int64) string {
	var all []string
	for i := starting; i < starting+new; i++ {
		all = append(all, fmt.Sprintf("$%d", i+1))
	}
	return strings.Join(all, ", ")
}

var funcMap = template.FuncMap{
	"lowerCamelCase": lowerCamelCase,
	"upperCamelCase": upperCamelCase,
	"elemComma":      elemComma,
	"valueExpansion": valueExpansion,
}
