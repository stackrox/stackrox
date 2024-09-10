//go:build test_e2e || sql_integration || compliance || destructive || externalbackups

package tests

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

type logMatcher interface {
	Match(reader io.ReadSeeker) (bool, error)
	fmt.Stringer
}

// multiLineMatcher matches when the desired number of lines are found.
type multiLineMatcher struct {
	re           *regexp.Regexp
	desiredLines int
	// fromLine represents the line after which to start considering matches
	fromLine int
}

// containsLineMatching returns a simple line-based regex matcher to go with waitUntilLog.
// Note: currently limited by bufio.Reader default buffer size (4KB) for simplicity.
func containsLineMatching(re *regexp.Regexp) *multiLineMatcher {
	return &multiLineMatcher{re: re, desiredLines: 1}
}

// containsLineMatchingAfter mimics containsLineMatching but will only attempt to match
// after the from line.
func containsLineMatchingAfter(re *regexp.Regexp, fromLine int) *multiLineMatcher {
	return &multiLineMatcher{re: re, desiredLines: 1, fromLine: fromLine}
}

// containsMultipleLinesMatching is a line-based regex matcher to go with waitUntilLog
// that will return true when the desired number of lines are found matching the reg exp.
func containsMultipleLinesMatching(re *regexp.Regexp, desiredLineCount int) *multiLineMatcher {
	return &multiLineMatcher{re: re, desiredLines: desiredLineCount}
}

func (lm *multiLineMatcher) String() string {
	return fmt.Sprintf("contains %d lines matching %q", lm.desiredLines, lm.re)
}

func (lm *multiLineMatcher) Match(reader io.ReadSeeker) (ok bool, err error) {
	br := bufio.NewReader(reader)
	var lineCount int
	var lineMatchCount int
	for {
		// We do not care about partial reads, as the things we look for should fit in default buf size.
		line, _, err := br.ReadLine()
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		lineCount++
		if lineCount < lm.fromLine {
			continue
		}
		if lm.re.Match(line) {
			lineMatchCount++
			if lineMatchCount >= lm.desiredLines {
				return true, nil
			}
		}
	}
}

// notFoundLineMatcher is a line-based regex matcher to go with waitUntilLog
// that will return true when NO lines are found that match the reg exp.
type notFoundLineMatcher struct {
	re *regexp.Regexp
}

// containsMultiLinesMatching is a convenience method for creating a not found line matcher.
func containsNoLinesMatching(re *regexp.Regexp) *notFoundLineMatcher {
	return &notFoundLineMatcher{re: re}
}

func (lm *notFoundLineMatcher) String() string {
	return fmt.Sprintf("contains NO lines matching %q", lm.re)
}

func (lm *notFoundLineMatcher) Match(reader io.ReadSeeker) (ok bool, err error) {
	br := bufio.NewReader(reader)
	for {
		// We do not care about partial reads, as the things we look for should fit in default buf size.
		line, _, err := br.ReadLine()
		if errors.Is(err, io.EOF) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		if lm.re.Match(line) {
			return false, nil
		}
	}
}

// anyMatcher is a composite matcher to go with waitUntilLog
// that will return true when any sub matcher matches.
type anyMatcher struct {
	matchers []logMatcher
}

func matchesAny(logMatchers ...logMatcher) *anyMatcher {
	return &anyMatcher{
		matchers: logMatchers,
	}
}

func (lm *anyMatcher) String() string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for _, m := range lm.matchers {
		if sb.Len() > 1 {
			sb.WriteString(", ")
		}
		sb.WriteString(m.String())
	}
	sb.WriteString("]")

	return fmt.Sprintf("matches any of %s", sb.String())
}

func (lm *anyMatcher) Match(reader io.ReadSeeker) (ok bool, err error) {
	errs := []error{}
	for _, m := range lm.matchers {
		_, err := reader.Seek(0, io.SeekStart)
		if err != nil {
			return false, fmt.Errorf("could not rewind the reader: %w", err)
		}

		ok, err := m.Match(reader)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		if ok {
			return true, nil
		}
	}

	return false, errors.Join(errs...)
}

// allMatcher is a composite matcher to go with waitUntilLog
// that will return true when all matchers match.
type allMatcher struct {
	matchers []logMatcher
}

func matchesAll(logMatchers ...logMatcher) *allMatcher {
	return &allMatcher{
		matchers: logMatchers,
	}
}

func (lm *allMatcher) String() string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for _, m := range lm.matchers {
		if sb.Len() > 1 {
			sb.WriteString(", ")
		}
		sb.WriteString(m.String())
	}
	sb.WriteString("]")

	return fmt.Sprintf("matches all of %s", sb.String())
}

func (lm *allMatcher) Match(reader io.ReadSeeker) (ok bool, err error) {
	for i, m := range lm.matchers {
		_, err := reader.Seek(0, io.SeekStart)
		if err != nil {
			return false, fmt.Errorf("could not rewind the reader: %w", err)
		}

		ok, err := m.Match(reader)
		if err != nil {
			return false, fmt.Errorf("matcher %d returned an error: %w", i, err)
		}

		if !ok {
			return false, nil
		}
	}

	return true, nil
}
