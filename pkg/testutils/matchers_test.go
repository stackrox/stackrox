package testutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPredMatcherWithTypeMismatch(t *testing.T) {
	t.Parallel()

	m := PredMatcher("", func(x string) bool { return true })
	assert.False(t, m.Matches(3))
}

func TestPredMatcherWithExactTypeMatch(t *testing.T) {
	t.Parallel()

	m := PredMatcher("", func(x string) bool { return true })
	assert.True(t, m.Matches("foo"))
}

type mockError struct{}

func (e mockError) Error() string { return "" }

func TestPredMatcherWithConversionTypeMatch(t *testing.T) {
	t.Parallel()

	m := PredMatcher("", func(x error) bool { return true })
	assert.True(t, m.Matches(mockError{}))
}

func TestAssertionMatcher(t *testing.T) {
	t.Parallel()

	m := AssertionMatcher(assert.ElementsMatch, []string{"a", "b"})
	assert.True(t, m.Matches([]string{"b", "a"}))
	assert.False(t, m.Matches([]string{"a", "c"}))

	assert.Regexp(t, `.*ElementsMatch\(\[a b\]\)`, m.String())
}
