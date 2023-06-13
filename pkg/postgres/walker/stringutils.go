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

func snakeCaseToLowerCamelCase(s string) string {
	words := strings.Split(s, "_")
	if len(words) == 0 {
		return s
	}
	words[0] = strings.ToLower(words[0])
	if len(words) == 1 && words[0] == "id" {
		return "id"
	}
	applyPointwise(words[1:], strings.Title)
	applyPointwise(words, stringutils.UpperCaseAcronyms)
	return strings.Join(words, "")
}
