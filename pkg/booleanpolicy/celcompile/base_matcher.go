package celcompile

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/parse"
	"github.com/stackrox/rox/pkg/search"
)

type simpleMatchCodesGenerator struct {
	MatchCodes []string
}

var (
	// ErrCelNotYetSupported is an error that indicates that a certain query is not yet supported.
	// We will further look to support more
	ErrCelNotYetSupported = errors.New("as-yet unsupported cel path")
)

func (s *simpleMatchCodesGenerator) Generate(v string) string {
	var orClauses []string
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

type matchCodesGenerator interface {
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
		matchCode, err := matchFunc(value)
		if err != nil {
			return nil, fmt.Errorf("failed to compile for value %s in values %v", value, values)
		}
		matchCodes = append(matchCodes, matchCode)
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
	return "!%s", nil
}

func getSimpleMatchFuncGenerators(query *query.FieldQuery, matchCodeGenerator func(string) (string, error)) ([]matchCodesGenerator, error) {
	if len(query.Values) == 0 {
		return nil, fmt.Errorf("no value for field %s", query.Field)
	}
	matchCodes, err := generateMultiMatchCode(query.Values, matchCodeGenerator)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate match code for field %s: %w", query.Field, err)
	}
	return []matchCodesGenerator{
		&simpleMatchCodesGenerator{MatchCodes: matchCodes},
	}, nil
}

func getStringMatchFuncGenerators(query *query.FieldQuery) ([]matchCodesGenerator, error) {
	return getSimpleMatchFuncGenerators(query, generateStringMatchCode)
}

func getBoolMatchFuncGenerators(query *query.FieldQuery) ([]matchCodesGenerator, error) {
	return getSimpleMatchFuncGenerators(query, generateBoolMatchCode)
}

func generateBaseMatcherHelper(query *query.FieldQuery, typ reflect.Type) ([]matchCodesGenerator, error) {
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

func generateMatchCodeForField(fieldQuery *query.FieldQuery, typ reflect.Type, v string) (string, error) {
	if fieldQuery.Operator == query.And || fieldQuery.Negate {
		return "", ErrCelNotYetSupported
	}
	var generators []matchCodesGenerator
	if fieldQuery.MatchAll {
		generators = []matchCodesGenerator{
			&simpleMatchCodesGenerator{MatchCodes: []string{"true"}},
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
