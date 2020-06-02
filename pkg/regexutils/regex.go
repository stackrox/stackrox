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

// CompileWholeStringMatcher takes a regex and compiles it into a WholeStringMatcher.
// An empty regex matches _all_ strings.
func CompileWholeStringMatcher(re string) (WholeStringMatcher, error) {
	if re == "" {
		return &wholeStringMatcher{}, nil
	}
	compiled, err := regexp.Compile(fmt.Sprintf("^(?:%s)$", re))
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
