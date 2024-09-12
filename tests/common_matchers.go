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

// This file contains utilities for finding/detecting specific log lines to help
// validate expected behaviors that cannot be validated elsewhere, such
// as StackRox API responses.

func allMatch(reader io.ReadSeeker, matchers ...logMatcher) (ok bool, err error) {
	for i, matcher := range matchers {
		_, err := reader.Seek(0, io.SeekStart)
		if err != nil {
			return false, fmt.Errorf("could not rewind the reader: %w", err)
		}
		ok, err := matcher.Match(reader)
		if err != nil {
			return false, fmt.Errorf("matcher %d returned an error: %w", i, err)
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

type logMatcher interface {
	Match(reader io.ReadSeeker) (bool, error)
	fmt.Stringer
}

// multiLineMatcher matches when the desired number of lines are found.
type multiLineMatcher struct {
	re       *regexp.Regexp
	numLines int
	fromByte int64
}

// containsLineMatching returns a simple line-based regex matcher to go with waitUntilLog.
// Note: currently limited by bufio.Reader default buffer size (4KB) for simplicity.
func containsLineMatching(re *regexp.Regexp) *multiLineMatcher {
	return &multiLineMatcher{re: re, numLines: 1}
}

// containsLineMatchingAfter mimics containsLineMatching but will only attempt to match
// lines that appear after fromByte.
func containsLineMatchingAfter(re *regexp.Regexp, fromByte int64) *multiLineMatcher {
	return &multiLineMatcher{re: re, numLines: 1, fromByte: fromByte}
}

// containsMultipleLinesMatching is a line-based regex matcher to go with waitUntilLog
// that will return true when the desired number of lines are found matching the reg exp.
//
//lint:ignore U1000 unused - utility function that may help future e2e test writers
func containsMultipleLinesMatching(re *regexp.Regexp, numLines int) *multiLineMatcher {
	return &multiLineMatcher{re: re, numLines: numLines}
}

// containsMultipleLinesMatchingAfter mimics containsMultipleLinesMatching but will only attempt to match
// lines that appear after fromByte.
func containsMultipleLinesMatchingAfter(re *regexp.Regexp, numLines int, fromByte int64) *multiLineMatcher {
	return &multiLineMatcher{re: re, numLines: numLines, fromByte: fromByte}
}

func (lm *multiLineMatcher) String() string {
	return fmt.Sprintf("contains line(s) matching %q", lm.re)
}

func (lm *multiLineMatcher) Match(reader io.ReadSeeker) (ok bool, err error) {
	br := bufio.NewReader(reader)

	if lm.fromByte != 0 {
		_, err = reader.Seek(lm.fromByte, io.SeekStart)
		if err != nil {
			return false, err
		}
	}

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
		if lm.re.Match(line) {
			lineMatchCount++
			if lineMatchCount >= lm.numLines {
				return true, nil
			}
		}
	}
}

// notFoundLineMatcher is a line-based regex matcher to go with waitUntilLog
// that will return true when NO lines are found that match the reg exp.
type notFoundLineMatcher struct {
	re       *regexp.Regexp
	fromByte int64
}

// containsMultiLinesMatching is a convenience method for creating a not found line matcher.
//
//lint:ignore U1000 unused - utility function that may help future e2e test writers
func containsNoLinesMatching(re *regexp.Regexp) *notFoundLineMatcher {
	return &notFoundLineMatcher{re: re}
}

// containsNoLinesMatchingAfter mimics containsMultiLinesMatching but will only attempt to match
// lines that appear after fromByte.
func containsNoLinesMatchingAfter(re *regexp.Regexp, fromByte int64) *notFoundLineMatcher {
	return &notFoundLineMatcher{re: re, fromByte: fromByte}
}

func (lm *notFoundLineMatcher) String() string {
	return fmt.Sprintf("contains NO lines matching %q", lm.re)
}

func (lm *notFoundLineMatcher) Match(reader io.ReadSeeker) (ok bool, err error) {
	br := bufio.NewReader(reader)

	if lm.fromByte != 0 {
		_, err = reader.Seek(lm.fromByte, io.SeekStart)
		if err != nil {
			return false, err
		}
	}

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

// orMatcher is a composite matcher to go with waitUntilLog
// that will return true when any sub matcher matches.
//
//lint:ignore U1000 unused - utility function that may help future e2e test writers
type orMatcher struct {
	matchers []logMatcher
}

//lint:ignore U1000 unused - utility function that may help future e2e test writers
func matchesAny(logMatchers ...logMatcher) *orMatcher {
	return &orMatcher{
		matchers: logMatchers,
	}
}

//lint:ignore U1000 unused - utility function that may help future e2e test writers
func (lm *orMatcher) String() string {
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

//lint:ignore U1000 unused - utility function that may help future e2e test writers
func (lm *orMatcher) Match(reader io.ReadSeeker) (ok bool, err error) {
	var errs []error
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
