package template

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

// required is for our .htpl files to validate that the required value is provided.
// This function mimics the same one available in Helm templates, see
// https://helm.sh/docs/howto/charts_tips_and_tricks/#using-the-required-function
func required(errMsg string, val interface{}) (interface{}, error) {
	if val == nil || reflect.ValueOf(val).IsZero() {
		// It is ok to provide empty errMsg because Go templates provides sufficient context information around the error.
		if errMsg == "" {
			errMsg = "required value was not specified"
		}
		return nil, errors.New(errMsg)
	}
	return val, nil
}
