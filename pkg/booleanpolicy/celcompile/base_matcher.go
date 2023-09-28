package celcompile

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

{{ if .NegateFuncName }}
{{ .NegateFuncName }}(val) {
    {{- range .MatchCodes }}
    negate({{.}})
	{{- end }}
}
{{ end }}

{{ if and .MatchFuncName .NegateFuncName }}
{{ .MatchFuncName }}(val) {
	not {{ .NegateFuncName }}(val)
}
{{ end }}

{{.Name}}(val) = result {
	{{ if .MatchFuncName }}
	result := { "match": {{ .MatchFuncName }}(val), "values": [val] }
    {{ else }}
	result := { "match": {{ index .MatchCodes 0 }}, "values": [val] }
    {{ end }}
}
`))
	filterCodeTemplate = template.Must(template.New("").Parse(`
		obj.{{.FieldName}}.startsWith("TopLevelValA")

`))
	mapCodeTemplate = template.Must(template.New("").Parse(`
result.map(t, t.with({"TopLevelA": [obj.ValA]}))
`))
)

type simpleMatchFuncGenerator struct {
	Name           string
	MatchFuncName  string
	NegateFuncName string
	MatchCodes     []string
}

var (
	invalidRegoFuncNameChars = regexp.MustCompile(`[^a-zA-Z0-9_]`)
)

var (
	// ErrCelNotYetSupported is an error that indicates that a certain query is not yet supported by rego.
	// It will be removed once rego is supported for all queries.
	ErrCelNotYetSupported = errors.New("as-yet unsupported cel path")
)

func sanitizeFuncName(name string) string {
	return invalidRegoFuncNameChars.ReplaceAllString(name, "_")
}

// getRegoFunctionName returns a rego function name for matching the field to the given value.
// The idx is also required, and is used to ensure the function name is unique.
func getRegoFunctionName(field string) string {
	return sanitizeFuncName(fmt.Sprintf("matchAndGetResultFor%s", field))
}

func getMatchFuncName(field string) string {
	return sanitizeFuncName(fmt.Sprintf("matchFor%s", field))
}

func getNegateFuncName(field string) string {
	return sanitizeFuncName(fmt.Sprintf("negateMatches%s", field))
}

func (s *simpleMatchFuncGenerator) Generate(v string) string {
	orClauses := []string{}
	for _, mc := range s.MatchCodes {
		if strings.Contains(mc, `%s`) {
			orClauses = append(orClauses, fmt.Sprintf(mc, v))
		} else {
			orClauses = append(orClauses, mc)
		}
	}
	code := strings.Join(orClauses, " || ")
	return code
}

func generateCheckCode(v string) string {
	parts := strings.Split(v, ".")
	code := ""
	p := parts[0]
	for _, n := range parts[1:] {
		p = fmt.Sprintf("%s.%s", p, n)
		if code == "" {
			code = fmt.Sprintf("has(%s)", p)
		} else {
			code = fmt.Sprintf("%s && has(%s)", code, p)
		}
	}
	if code == "" {
		code = "true"
	}
	return code
}

func (s *simpleMatchFuncGenerator) FuncName() string {
	return s.Name
}

type matchFuncGenerator interface {
	Generate(v string) string
}

const (
	strMatcher = "%s"
)

func generateStringMatchCode(value string) (string, error) {
	negated := strings.HasPrefix(value, search.NegationPrefix)
	if negated {
		value = strings.TrimPrefix(value, search.NegationPrefix)

	}
	var matchCode string
	if strings.HasPrefix(value, search.RegexPrefix) {
		// Cel does not process escape
		m := strings.TrimPrefix(value, search.RegexPrefix)
		m = strings.ReplaceAll(m, `\`, `\\`)
		matchCode = fmt.Sprintf("%s.matches('^(?i:%s)$')", strMatcher, m)
	} else if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) && len(value) > 1 {
		matchCode = fmt.Sprintf(`%s == "%s"`, strMatcher, value[1:len(value)-1])
	} else {
		matchCode = fmt.Sprintf(`%s.startsWith("%s")`, strMatcher, value)
	}
	if negated {
		matchCode = fmt.Sprintf(`(%s) == false`, matchCode)
	}
	return matchCode, nil
}

func generateMultiMatchCode(values []string, matchFunc func(string) (string, error)) ([]string, error) {
	matchCodes := make([]string, 0)
	if len(values) == 0 {
		return nil, errors.New("expect at least one value")
	}
	for _, value := range values {
		if matchCode, err := matchFunc(value); err != nil {
			return nil, fmt.Errorf("failed to compile for value %s in values %v", value, values)
		} else {
			matchCodes = append(matchCodes, matchCode)
		}
	}
	return matchCodes, nil
}

func generateBoolMatchCode(value string) (string, error) {
	boolValue, err := parse.FriendlyParseBool(value)
	if err != nil {
		return "", err
	}
	if boolValue {
		return "%s", nil
	}
	return "%s == false", nil
}

func getSimpleMatchFuncGenerators(query *query.FieldQuery, matchCodeGenerator func(string) (string, error)) ([]matchFuncGenerator, error) {
	if len(query.Values) == 0 {
		return nil, fmt.Errorf("no value for field %s", query.Field)
	}
	var generators []matchFuncGenerator
	matchCodes, err := generateMultiMatchCode(query.Values, matchCodeGenerator)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate match code for field %s: %w", query.Field, err)
	}
	if len(matchCodes) == 1 {
		generators = append(generators, &simpleMatchFuncGenerator{
			Name:       getRegoFunctionName(query.Field),
			MatchCodes: matchCodes,
		})
	} else {
		generators = append(generators, &simpleMatchFuncGenerator{
			Name:           getRegoFunctionName(query.Field),
			MatchFuncName:  getMatchFuncName(query.Field),
			NegateFuncName: getNegateFuncName(query.Field),
			MatchCodes:     matchCodes,
		})
	}
	return generators, nil
}

func getStringMatchFuncGenerators(query *query.FieldQuery) ([]matchFuncGenerator, error) {
	return getSimpleMatchFuncGenerators(query, generateStringMatchCode)
}

func getBoolMatchFuncGenerators(query *query.FieldQuery) ([]matchFuncGenerator, error) {
	return getSimpleMatchFuncGenerators(query, generateBoolMatchCode)
}

func generateBaseMatcherHelper(query *query.FieldQuery, typ reflect.Type) ([]matchFuncGenerator, error) {
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
	return nil, ErrCelNotYetSupported
}

type regoMatchFunc struct {
	filterCode   string
	mapCode      string
	functionName string // not in use
	functionCode string // not in use
}

func generateMatchCodeForField(fieldQuery *query.FieldQuery, typ reflect.Type, v string) (string, error) {
	if fieldQuery.Operator == query.And || fieldQuery.Negate {
		return "", ErrCelNotYetSupported
	}
	var generators []matchFuncGenerator
	if fieldQuery.MatchAll {
		generators = []matchFuncGenerator{
			&simpleMatchFuncGenerator{Name: sanitizeFuncName(fmt.Sprintf("matchAll%s", fieldQuery.Field)), MatchCodes: []string{"true"}},
		}
	} else {
		var err error
		generators, err = generateBaseMatcherHelper(fieldQuery, typ)
		if err != nil {
			return "", err
		}
	}
	if len(generators) == 0 {
		return "", fmt.Errorf("got no generators for fieldQuery %+v", fieldQuery)
	}
	return generators[0].Generate(v), nil
}
