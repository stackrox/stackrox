package regexutils

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
)

// A StringMatcher is something that can match a string against a regex
type StringMatcher interface {
	MatchString(s string) bool
}

// Flags represents regex flags.
// See the Flags table in https://github.com/google/re2/wiki/Syntax for the list
// of regex flags.
// For simplicity, we have not implemented all flags here, only what
// we have used, but we are using a struct to enable seamlessly adding more.
type Flags struct {
	CaseInsensitive bool
}

// regexRepr represents these flags in the regexp format.
// See the Flags table in https://github.com/google/re2/wiki/Syntax.
func (f *Flags) regexRepr() string {
	if f.CaseInsensitive {
		return "i"
	}
	return ""
}

// CompileWholeStringMatcher takes a regex and compiles it into a StringMatcher for whole strings that match the regex pattern.
// An empty regex matches _all_ strings.
func CompileWholeStringMatcher(re string, flags Flags) (StringMatcher, error) {
	if re == "" {
		return &stringMatcher{}, nil
	}
	compiled, err := regexp.Compile(fmt.Sprintf("^(?%s:%s)$", flags.regexRepr(), re))
	if err != nil {
		return nil, errors.Wrap(err, "invalid regex")
	}
	return &stringMatcher{r: compiled}, nil
}

// CompileContainsStringMatcher takes a regex and compiles it into a StringMatcher for strings containing the regex pattern.
// An empty regex matches _all_ strings.
func CompileContainsStringMatcher(re string, flags Flags) (StringMatcher, error) {
	if re == "" {
		return &stringMatcher{}, nil
	}
	compiled, err := regexp.Compile(fmt.Sprintf("(?%s:%s)", flags.regexRepr(), re))
	if err != nil {
		return nil, errors.Wrap(err, "invalid regex")
	}
	return &stringMatcher{r: compiled}, nil
}

type stringMatcher struct {
	r *regexp.Regexp
}

func (w *stringMatcher) MatchString(s string) bool {
	return w.r == nil || w.r.MatchString(s)
}
