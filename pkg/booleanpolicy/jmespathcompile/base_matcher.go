package jmespathcompile

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"text/template"

	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/parse"
	"github.com/stackrox/rox/pkg/search"
)

var (
	simpleMatchFuncTemplate = template.Must(template.New("").Parse(`
{{.Name}}(val) = result {
	result := { "match": {{ .MatchCode }}, "values": [val] }
}
`))
)

type simpleMatchFuncGenerator struct {
	Name      string
	MatchCode string
}

var (
	invalidRegoFuncNameChars = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

var (
	// ErrRegoNotYetSupported is an error that indicates that a certain query is not yet supported by jmespath.
	// It will be removed once jmespath is supported for all queries.
	ErrRegoNotYetSupported = errors.New("as-yet unsupported jmespath path")
)

func sanitizeFuncName(name string) string {
	return invalidRegoFuncNameChars.ReplaceAllString(name, "_")
}

// getRegoFunctionName returns a jmespath function name for matching the field to the given value.
// The idx is also required, and is used to ensure the function name is unique.
func getRegoFunctionName(field, value string, idx int) string {
	return sanitizeFuncName(fmt.Sprintf("match%sTo%d%s", field, idx, value))
}

func (s *simpleMatchFuncGenerator) GenerateJMESPath() (string, error) {
	var sb strings.Builder
	err := simpleMatchFuncTemplate.Execute(&sb, s)
	if err != nil {
		return "", err
	}
	return sb.String(), nil
}

func (s *simpleMatchFuncGenerator) FuncName() string {
	return s.Name
}

type jmespathMatchFuncGenerator interface {
	GenerateJMESPath() (string, error)
	FuncName() string
}

func generateStringMatchCode(value string) (string, error) {
	negated := strings.HasPrefix(value, search.NegationPrefix)
	if negated {
		value = strings.TrimPrefix(value, search.NegationPrefix)

	}
	var matchCode string
	if strings.HasPrefix(value, search.RegexPrefix) {
		matchCode = fmt.Sprintf("regex.match(`^(?i:%s)$`, val)", strings.TrimPrefix(value, search.RegexPrefix))
	} else if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) && len(value) > 1 {
		matchCode = fmt.Sprintf(`val == "%s"`, value[1:len(value)-1])
	} else {
		matchCode = fmt.Sprintf("starts_with(val, `%s`)", value)
	}
	if negated {
		matchCode = fmt.Sprintf(`(%s) == false`, matchCode)
	}
	return matchCode, nil
}

func generateBoolMatchCode(value string) (string, error) {
	boolValue, err := parse.FriendlyParseBool(value)
	if err != nil {
		return "", err
	}
	if boolValue {
		return "val", nil
	}
	return "val == false", nil
}

func getSimpleMatchFuncGenerators(query *query.FieldQuery, matchCodeGenerator func(string) (string, error)) ([]jmespathMatchFuncGenerator, error) {
	if len(query.Values) == 0 {
		return nil, fmt.Errorf("no value for field %s", query.Field)
	}
	var generators []jmespathMatchFuncGenerator
	for i, val := range query.Values {
		matchCode, err := matchCodeGenerator(val)
		if err != nil {
			return nil, fmt.Errorf("couldn't generate match code for val %s from field %s: %w", val, query.Field, err)
		}
		generators = append(generators, &simpleMatchFuncGenerator{
			Name:      getRegoFunctionName(query.Field, val, i),
			MatchCode: matchCode,
		})
	}
	return generators, nil

}

func getStringMatchFuncGenerators(query *query.FieldQuery) ([]jmespathMatchFuncGenerator, error) {
	return getSimpleMatchFuncGenerators(query, generateStringMatchCode)
}

func getBoolMatchFuncGenerators(query *query.FieldQuery) ([]jmespathMatchFuncGenerator, error) {
	return getSimpleMatchFuncGenerators(query, generateBoolMatchCode)
}

func generateBaseMatcherHelper(query *query.FieldQuery, typ reflect.Type) ([]jmespathMatchFuncGenerator, error) {
	switch kind := typ.Kind(); kind {
	case reflect.String:
		return getStringMatchFuncGenerators(query)
	case reflect.Ptr:
		// return generatePtrMatcher
	case reflect.Array, reflect.Slice:
		// return generateSliceMatcher
	case reflect.Map:
		// return generateMapMatcher
	case reflect.Bool:
		return getBoolMatchFuncGenerators(query)
	case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int:
		// return generateIntMatcher
	case reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:
		// return generateUintMatcher
	case reflect.Float64, reflect.Float32:
		// return generateFloatMatcher
	default:
		return nil, fmt.Errorf("invalid kind for base query: %s", kind)
	}
	return nil, ErrRegoNotYetSupported
}

type jmespathMatchFunc struct {
	functionCode string
	functionName string
}

func generateMatchersForField(fieldQuery *query.FieldQuery, typ reflect.Type) ([]jmespathMatchFunc, error) {
	if fieldQuery.Operator == query.And || fieldQuery.Negate {
		return nil, ErrRegoNotYetSupported
	}
	var generators []jmespathMatchFuncGenerator
	if fieldQuery.MatchAll {
		generators = []jmespathMatchFuncGenerator{
			&simpleMatchFuncGenerator{Name: sanitizeFuncName(fmt.Sprintf("matchAll%s", fieldQuery.Field)), MatchCode: "true"},
		}
	} else {
		var err error
		generators, err = generateBaseMatcherHelper(fieldQuery, typ)
		if err != nil {
			return nil, err
		}
	}
	if len(generators) == 0 {
		return nil, fmt.Errorf("got no generators for fieldQuery %+v", fieldQuery)
	}
	var funcs []jmespathMatchFunc
	for _, gen := range generators {
		code, err := gen.GenerateJMESPath()
		if err != nil {
			return nil, err
		}
		funcs = append(funcs, jmespathMatchFunc{functionCode: code, functionName: gen.FuncName()})
	}
	return funcs, nil
}
