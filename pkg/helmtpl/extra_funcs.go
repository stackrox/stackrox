package helmtpl

import (
	"errors"
	"reflect"
	"text/template"
)

var (
	extraFuncMap = template.FuncMap{
		"required": required,
	}
)

func required(errMsg string, val interface{}) (interface{}, error) {
	if val == nil || reflect.ValueOf(val).IsZero() {
		return nil, errors.New(errMsg)
	}
	return val, nil
}
