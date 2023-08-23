package sliceutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testType string

func (t testType) String() string {
	return string(t)
}

func TestStringSlice(t *testing.T) {
	in := []testType{
		"these", "are", "test", "values",
	}

	s := StringSlice(in...)

	assert.Equal(t, []string{"these", "are", "test", "values"}, s)
}

func TestFromStringSlice(t *testing.T) {
	in := []string{
		"these", "are", "test", "values",
	}

	testTypes := FromStringSlice[testType](in...)

	assert.IsType(t, []testType{}, testTypes)

	assert.ElementsMatch(t, []testType{"these", "are", "test", "values"}, testTypes)
}
