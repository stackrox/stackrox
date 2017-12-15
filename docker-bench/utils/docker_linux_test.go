// +build linux

package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPID(t *testing.T) {
	if val := os.Getenv("CIRCLECI"); len(val) != 0 {
		t.Skip("This test cannot run in CircleCI Docker-in-Docker")
	}

	pid, err := getPID("init")
	assert.Nil(t, err)
	assert.Equal(t, 1, pid)

	pid, err = getPID("howdy")
	assert.NotNil(t, err)
}

func TestGetProcessPID(t *testing.T) {
	if val := os.Getenv("CIRCLECI"); len(val) != 0 {
		t.Skip("This test cannot run in CircleCI Docker-in-Docker")
	}

	processes := []string{"howdy", "init", "blah"}
	pid, name, err := getProcessPID(processes)
	assert.Nil(t, err)
	assert.Equal(t, 1, pid)
	assert.Equal(t, "init", name)

	processes = []string{"howdy"}
	_, _, err = getProcessPID(processes)
	assert.NotNil(t, err)
}

func TestGetCommandLine(t *testing.T) {
	if val := os.Getenv("CIRCLECI"); len(val) != 0 {
		t.Skip("This test cannot run in CircleCI Docker-in-Docker")
	}

	cmdline, err := getCommandLine(1)
	require.Nil(t, err)
	assert.Contains(t, cmdline, "init")
}
