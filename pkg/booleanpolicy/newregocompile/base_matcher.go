package newregocompile

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
	// ErrRegoNotYetSupported is an error that indicates that a certain query is not yet supported by rego.
	// It will be removed once rego is supported for all queries.
	ErrRegoNotYetSupported = errors.New("as-yet unsupported rego path")
)

func sanitizeFuncName(name string) string {
	return invalidRegoFuncNameChars.ReplaceAllString(name, "_")
}

// getRegoFunctionName returns a rego function name for matching the field to the given value.
// The idx is also required, and is used to ensure the function name is unique.
func getRegoFunctionName(field string) string {
	return sanitizeFuncName(fmt.Sprintf("match%s", field))
}

func (s *simpleMatchFuncGenerator) GenerateRego() (string, error) {
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

type regoMatchFuncGenerator interface {
	GenerateRego() (string, error)
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
		matchCode = fmt.Sprintf(`startswith(val, "%s")`, value)
	}
	if negated {
		matchCode = fmt.Sprintf(`(%s) == false`, matchCode)
	}
	return matchCode, nil
}

func generateMultiMatchCode(values []string, matchFunc func(string) (string, error)) (string, error) {
	if len(values) == 0 {
		return "", errors.New("expect at least one value")
	}
	multiMatch, err := matchFunc(values[0])
	if len(values) == 1 {
		return multiMatch, err
	}
	for _, value := range values[1:] {
		if matchCode, err := matchFunc(value); err != nil {
			return "", fmt.Errorf("failed to compile for value %s in values %v", value, values)
		} else {
			multiMatch = fmt.Sprintf("%s, %s", multiMatch, matchCode)
		}
	}
	return fmt.Sprintf("or([%s])", multiMatch), nil
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

func getSimpleMatchFuncGenerators(query *query.FieldQuery, matchCodeGenerator func(string) (string, error)) ([]regoMatchFuncGenerator, error) {
	if len(query.Values) == 0 {
		return nil, fmt.Errorf("no value for field %s", query.Field)
	}
	var generators []regoMatchFuncGenerator
	matchCode, err := generateMultiMatchCode(query.Values, matchCodeGenerator)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate match code for field %s: %w", query.Field, err)
	}
	generators = append(generators, &simpleMatchFuncGenerator{
		Name:      getRegoFunctionName(query.Field),
		MatchCode: matchCode,
	})
	return generators, nil
}

func getStringMatchFuncGenerators(query *query.FieldQuery) ([]regoMatchFuncGenerator, error) {
	return getSimpleMatchFuncGenerators(query, generateStringMatchCode)
}

func getBoolMatchFuncGenerators(query *query.FieldQuery) ([]regoMatchFuncGenerator, error) {
	return getSimpleMatchFuncGenerators(query, generateBoolMatchCode)
}

func generateBaseMatcherHelper(query *query.FieldQuery, typ reflect.Type) ([]regoMatchFuncGenerator, error) {
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

type regoMatchFunc struct {
	functionCode string
	functionName string
}

func generateMatchersForField(fieldQuery *query.FieldQuery, typ reflect.Type) ([]regoMatchFunc, error) {
	if fieldQuery.Operator == query.And || fieldQuery.Negate {
		return nil, ErrRegoNotYetSupported
	}
	var generators []regoMatchFuncGenerator
	if fieldQuery.MatchAll {
		generators = []regoMatchFuncGenerator{
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
	var funcs []regoMatchFunc
	for _, gen := range generators {
		code, err := gen.GenerateRego()
		if err != nil {
			return nil, err
		}
		funcs = append(funcs, regoMatchFunc{functionCode: code, functionName: gen.FuncName()})
	}
	return funcs, nil
}
