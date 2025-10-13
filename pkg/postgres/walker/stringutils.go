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

// makeSingular aims at converting a title or lower-case word to its singular form.
// assumption is made that any multi-letter word ending with s is a plural word and its singular form
// is without trailing s.
// Although this is not always a valid assumption (example of assumption flaw: matress),
// this seems to be verified with the postgres table and children-table names.
func makeSingular(s string) string {
	l := len(s)
	if l <= 1 {
		return s
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
