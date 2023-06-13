package stringutils

import (
	"strings"
)

var (
	acronymsToUpperCase = map[string]struct{}{
		"API":  {},
		"CPU":  {},
		"ID":   {},
		"UID":  {},
		"UUID": {},
	}
)

func UpperCaseAcronyms(s string) string {
	uc := strings.ToUpper(s)
	if _, found := acronymsToUpperCase[uc]; found {
		return uc
	}
	return s
}
