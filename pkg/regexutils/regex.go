package regexutils

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
)

// A WholeStringMatcher is something that can match a string against a regex, but only matches
// strings where the entire string matches the regex.
type WholeStringMatcher interface {
	MatchWholeString(s string) bool
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

// CompileWholeStringMatcher takes a regex and compiles it into a WholeStringMatcher.
// An empty regex matches _all_ strings.
func CompileWholeStringMatcher(re string, flags Flags) (WholeStringMatcher, error) {
	if re == "" {
		return &wholeStringMatcher{}, nil
	}
	compiled, err := regexp.Compile(fmt.Sprintf("^(?%s:%s)$", flags.regexRepr(), re))
	if err != nil {
		return nil, errors.Wrap(err, "invalid regex")
	}
	return &wholeStringMatcher{r: compiled}, nil
}

type wholeStringMatcher struct {
	r *regexp.Regexp
}

func (w *wholeStringMatcher) MatchWholeString(s string) bool {
	return w.r == nil || w.r.MatchString(s)
}
