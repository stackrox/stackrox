package utils

import (
	"encoding/json"
	"text/template"
)

var (
	// BuiltinFuncs are helper functions for templates
	BuiltinFuncs = template.FuncMap{
		"jsonquote": jsonQuote,
	}
)

func jsonQuote(s string) (string, error) {
	bytes, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
