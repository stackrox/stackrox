package utils

import (
	"regexp"
)

// CompileStringRegex returns the compiled regex if string is not empty,
// otherwise nil is returned.
func CompileStringRegex(policy string) (*regexp.Regexp, error) {
	if policy == "" {
		return nil, nil
	}
	return regexp.Compile(policy)
}
