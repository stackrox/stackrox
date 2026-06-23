package vmhelpers

import (
	"io"
	"regexp"

	"github.com/stackrox/rox/tests/logmatchers"
)

// LogMatcher is re-exported from the logmatchers package for backward compatibility.
type LogMatcher = logmatchers.LogMatcher

// AllMatch returns true when every matcher succeeds against the provided reader.
func AllMatch(reader io.ReadSeeker, matchers ...LogMatcher) (bool, error) {
	return logmatchers.AllMatch(reader, matchers...)
}

// ContainsLineMatching returns a simple line-based regex matcher.
func ContainsLineMatching(re *regexp.Regexp) LogMatcher {
	return logmatchers.ContainsLineMatching(re)
}

// ContainsLineMatchingAfter mimics ContainsLineMatching but will only attempt to match
// lines that appear after fromByte.
func ContainsLineMatchingAfter(re *regexp.Regexp, fromByte int64) LogMatcher {
	return logmatchers.ContainsLineMatchingAfter(re, fromByte)
}

// ContainsMultipleLinesMatching is a line-based regex matcher that will return true
// when the desired number of lines are found matching the regexp.
func ContainsMultipleLinesMatching(re *regexp.Regexp, numLines int) LogMatcher {
	return logmatchers.ContainsMultipleLinesMatching(re, numLines)
}

// ContainsMultipleLinesMatchingAfter mimics ContainsMultipleLinesMatching but will only
// attempt to match lines that appear after fromByte.
func ContainsMultipleLinesMatchingAfter(re *regexp.Regexp, numLines int, fromByte int64) LogMatcher {
	return logmatchers.ContainsMultipleLinesMatchingAfter(re, numLines, fromByte)
}

// ContainsNoLinesMatching returns a matcher that succeeds when no lines match the regexp.
func ContainsNoLinesMatching(re *regexp.Regexp) LogMatcher {
	return logmatchers.ContainsNoLinesMatching(re)
}

// ContainsNoLinesMatchingAfter mimics ContainsNoLinesMatching but will only attempt to match
// lines that appear after fromByte.
func ContainsNoLinesMatchingAfter(re *regexp.Regexp, fromByte int64) LogMatcher {
	return logmatchers.ContainsNoLinesMatchingAfter(re, fromByte)
}
