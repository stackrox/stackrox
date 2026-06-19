//go:build test_e2e || test_e2e_vm || sql_integration || compliance || destructive || externalbackups || test_compatibility

package tests

import (
	"io"
	"regexp"

	"github.com/stackrox/rox/tests/vmhelpers"
)

// This file contains utilities for finding/detecting specific log lines to help
// validate expected behaviors that cannot be validated elsewhere, such
// as StackRox API responses.

func allMatch(reader io.ReadSeeker, matchers ...logMatcher) (bool, error) {
	return vmhelpers.AllMatch(reader, matchers...)
}

// containsLineMatching returns a simple line-based regex matcher to go with waitUntilLog.
// Note: currently limited by bufio.Reader default buffer size (4KB) for simplicity.
func containsLineMatching(re *regexp.Regexp) logMatcher {
	return vmhelpers.ContainsLineMatching(re)
}

// containsLineMatchingAfter mimics containsLineMatching but will only attempt to match
// lines that appear after fromByte.
func containsLineMatchingAfter(re *regexp.Regexp, fromByte int64) logMatcher {
	return vmhelpers.ContainsLineMatchingAfter(re, fromByte)
}

// containsMultipleLinesMatching is a line-based regex matcher to go with waitUntilLog
// that will return true when the desired number of lines are found matching the reg exp.
func containsMultipleLinesMatching(re *regexp.Regexp, numLines int) logMatcher {
	return vmhelpers.ContainsMultipleLinesMatching(re, numLines)
}

// containsMultipleLinesMatchingAfter mimics containsMultipleLinesMatching but will only attempt to match
// lines that appear after fromByte.
func containsMultipleLinesMatchingAfter(re *regexp.Regexp, numLines int, fromByte int64) logMatcher {
	return vmhelpers.ContainsMultipleLinesMatchingAfter(re, numLines, fromByte)
}

// containsMultiLinesMatching is a convenience method for creating a not found line matcher.
//
//lint:ignore U1000 unused - utility function that may help future e2e test writers
func containsNoLinesMatching(re *regexp.Regexp) logMatcher {
	return vmhelpers.ContainsNoLinesMatching(re)
}

// containsNoLinesMatchingAfter mimics containsMultiLinesMatching but will only attempt to match
// lines that appear after fromByte.
func containsNoLinesMatchingAfter(re *regexp.Regexp, fromByte int64) logMatcher {
	return vmhelpers.ContainsNoLinesMatchingAfter(re, fromByte)
}
