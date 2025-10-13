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

// UpperCaseAcronyms returns the uppercase variant of the input if it is one of a specific set of acronyms
// (currently API, CPU, ID, UID, UUID).
func UpperCaseAcronyms(s string) string {
	uc := strings.ToUpper(s)
	if _, found := acronymsToUpperCase[uc]; found {
		return uc
	}
	return s
}
