package walker

import (
	"strings"

	"github.com/stackrox/rox/pkg/stringutils"
)

func applyPointwise(ss []string, f func(string) string) {
	for i, s := range ss {
		ss[i] = f(s)
	}
}

func makeSingular(s string) string {
	l := len(s)
	if l == 0 {
		return s
	}
	if l == 1 && (s[0] == 's' || s[0] == 'S') {
		return s[:l-1]
	}
	if s[l-1] == 's' {
		return s[:l-1]
	}
	return s
}

func snakeCaseToSingularLowerCamelCase(s string) string {
	words := strings.Split(s, "_")
	if len(words) == 0 {
		return s
	}
	words[0] = strings.ToLower(words[0])
	words[0] = makeSingular(words[0])
	if len(words) == 1 && words[0] == "id" {
		return "id"
	}
	applyPointwise(words[1:], strings.Title)
	applyPointwise(words[1:], makeSingular)
	applyPointwise(words, stringutils.UpperCaseAcronyms)
	return strings.Join(words, "")
}
