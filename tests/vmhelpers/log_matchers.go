package vmhelpers

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
)

// LogMatcher describes a log predicate used by e2e helpers and unit tests.
type LogMatcher interface {
	Match(reader io.ReadSeeker) (bool, error)
	fmt.Stringer
}

// AllMatch returns true when every matcher succeeds against the provided reader.
func AllMatch(reader io.ReadSeeker, matchers ...LogMatcher) (bool, error) {
	for i, matcher := range matchers {
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

// multiLineLogMatcher matches when the desired number of lines are found.
type multiLineLogMatcher struct {
	re       *regexp.Regexp
	numLines int
	fromByte int64
}

// ContainsLineMatching returns a simple line-based regex matcher to go with waitUntilLog.
// Note: currently limited by bufio.Reader default buffer size (4KB) for simplicity.
func ContainsLineMatching(re *regexp.Regexp) LogMatcher {
	return containsMultipleLinesMatching(re, 1)
}

// ContainsLineMatchingAfter mimics ContainsLineMatching but will only attempt to match
// lines that appear after fromByte.
func ContainsLineMatchingAfter(re *regexp.Regexp, fromByte int64) LogMatcher {
	return containsMultipleLinesMatchingAfter(re, 1, fromByte)
}

// ContainsMultipleLinesMatching is a line-based regex matcher to go with waitUntilLog
// that will return true when the desired number of lines are found matching the reg exp.
func ContainsMultipleLinesMatching(re *regexp.Regexp, numLines int) LogMatcher {
	return containsMultipleLinesMatching(re, numLines)
}

func containsMultipleLinesMatching(re *regexp.Regexp, numLines int) *multiLineLogMatcher {
	return &multiLineLogMatcher{re: re, numLines: numLines}
}

// ContainsMultipleLinesMatchingAfter mimics ContainsMultipleLinesMatching but will only attempt to match
// lines that appear after fromByte.
func ContainsMultipleLinesMatchingAfter(re *regexp.Regexp, numLines int, fromByte int64) LogMatcher {
	return containsMultipleLinesMatchingAfter(re, numLines, fromByte)
}

func containsMultipleLinesMatchingAfter(re *regexp.Regexp, numLines int, fromByte int64) *multiLineLogMatcher {
	return &multiLineLogMatcher{re: re, numLines: numLines, fromByte: fromByte}
}

func (lm *multiLineLogMatcher) String() string {
	return fmt.Sprintf("contains line(s) matching %q", lm.re)
}

func (lm *multiLineLogMatcher) Match(reader io.ReadSeeker) (ok bool, err error) {
	if lm.re == nil {
		return false, errors.New("invalid matcher config, re is nil")
	}

	if lm.numLines <= 0 {
		return false, fmt.Errorf("invalid matcher config, numLines (%d) is <= 0", lm.numLines)
	}

	if lm.fromByte < 0 {
		return false, fmt.Errorf("invalid matcher config, fromByte (%d) is < 0", lm.fromByte)
	}

	br := bufio.NewReader(reader)

	_, err = reader.Seek(lm.fromByte, io.SeekStart)
	if err != nil {
		return false, fmt.Errorf("could not seek to pos %d: %w", lm.fromByte, err)
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

// ContainsNoLinesMatching is a convenience method for creating a not-found line matcher.
func ContainsNoLinesMatching(re *regexp.Regexp) LogMatcher {
	return &notFoundLineMatcher{re: re}
}

// ContainsNoLinesMatchingAfter mimics ContainsNoLinesMatching but will only attempt to match
// lines that appear after fromByte.
func ContainsNoLinesMatchingAfter(re *regexp.Regexp, fromByte int64) LogMatcher {
	return &notFoundLineMatcher{re: re, fromByte: fromByte}
}

func (lm *notFoundLineMatcher) String() string {
	return fmt.Sprintf("contains NO lines matching %q", lm.re)
}

func (lm *notFoundLineMatcher) Match(reader io.ReadSeeker) (ok bool, err error) {
	if lm.re == nil {
		return false, errors.New("invalid matcher config, re is nil")
	}

	if lm.fromByte < 0 {
		return false, fmt.Errorf("invalid matcher config, fromByte (%d) is < 0", lm.fromByte)
	}

	br := bufio.NewReader(reader)

	_, err = reader.Seek(lm.fromByte, io.SeekStart)
	if err != nil {
		return false, fmt.Errorf("could not seek to pos %d: %w", lm.fromByte, err)
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
