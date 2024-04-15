package testutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRunningInCI(t *testing.T) {
	for ci, expected := range map[string]bool{
		"abc":   true,
		"yes":   true,
		"no":    true,
		"true":  true,
		"":      true,
		"false": false,
		"False": false,
	} {
		t.Setenv("CI", ci)
		assert.Equal(t, expected, IsRunningInCI(), ci)
	}
}
