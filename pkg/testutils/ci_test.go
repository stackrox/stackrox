package testutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsRunningInCI(t *testing.T) {
	_, set := os.LookupEnv("CI")
	assert.Equal(t, set, IsRunningInCI()) // False in local, True in CI.

	for _, ci := range []string{"abc", "yes", "no", "true", "", "false", "False"} {
		t.Setenv("CI", ci)
		assert.True(t, IsRunningInCI(), ci)
	}
}
