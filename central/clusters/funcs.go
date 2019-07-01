package clusters

import (
	"encoding/json"
	"text/template"
)

var (
	builtinFuncs = template.FuncMap{
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
