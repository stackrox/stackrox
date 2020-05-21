package violations

import (
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/pkg/errors"
)

func stringSliceToSortedSentence(s []string) string {
	sort.Strings(s)
	return stringSliceToSentence(s)
}

func stringSliceToSentence(s []string) string {
	var sb strings.Builder
	switch sLen := len(s); {
	case sLen == 1:
		fmt.Fprintf(&sb, "%s", s[0])
	case sLen == 2:
		fmt.Fprintf(&sb, "%s and %s", s[0], s[1])
	default:
		for idx, elem := range s {
			if idx < sLen-1 {
				fmt.Fprintf(&sb, "%s, ", elem)
			} else {
				fmt.Fprintf(&sb, "and %s", elem)
			}
		}
	}
	return sb.String()
}

func maybeGetSingleValueFromFieldMap(f string, fieldMap map[string][]string) string {
	values, ok := fieldMap[f]
	if !ok {
		return ""
	}
	if lenValues := len(values); lenValues != 1 {
		return ""
	}
	return values[0]
}

func getSingleValueFromFieldMap(f string, fieldMap map[string][]string) (string, error) {
	values, ok := fieldMap[f]
	if !ok {
		return "", errors.Errorf("missing field %s", f)
	}
	if lenValues := len(values); lenValues != 1 {
		return "", errors.Errorf("unexpected number of values for field(%s)=%d", f, lenValues)
	}
	return values[0], nil
}

func executeTemplate(tpl string, values interface{}) ([]string, error) {
	tmpl, err := template.New("").Parse(tpl)
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	err = tmpl.Execute(&sb, values)
	if err != nil {
		return nil, err
	}
	return []string{sb.String()}, nil
}
