// +build linux

package utils

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go exec.CommandContext(ctx, "/bin/sleep", "5").Run()
	time.Sleep(1 * time.Second)
	pid, err := getPID("sleep")
	assert.Nil(t, err)
	assert.NotEqual(t, -1, pid)
	cancel()

	pid, err = getPID("howdy")
	assert.NotNil(t, err)
}

func TestGetProcessPID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go exec.CommandContext(ctx, "/bin/sleep", "5").Run()
	time.Sleep(1 * time.Second)
	processes := []string{"howdy", "sleep", "blah"}
	pid, name, err := getProcessPID(processes)
	assert.Nil(t, err)
	assert.NotEqual(t, -1, pid)
	assert.Equal(t, "sleep", name)
	cancel()
	processes = []string{"howdy"}
	_, _, err = getProcessPID(processes)
	assert.NotNil(t, err)
}

func TestGetCommandLine(t *testing.T) {
	cmdline, err := getCommandLine(1)
	require.Nil(t, err)
	assert.NotEqual(t, 0, len(cmdline))
}
