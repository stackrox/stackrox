package main

import (
	"regexp"
	"strings"
)

var devSuffixRe = regexp.MustCompile(`\.x(-|$)`)

// toBundleTag converts a main image tag to the operator bundle image tag:
// replace ".x" with ".0" in the version segment and prefix "v". A "-dirty"
// suffix is preserved, matching the bundle tag produced by operator/Makefile.
func toBundleTag(tag string) string {
	return "v" + devSuffixRe.ReplaceAllStringFunc(tag, func(m string) string {
		return strings.Replace(m, ".x", ".0", 1)
	})
}
