package main

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/maincommand"
	"github.com/stretchr/testify/assert"
)

func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
	buf := new(bytes.Buffer)
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	err = root.Execute()
	return buf.String(), err
}

func mockRun(_ *cobra.Command, _ []string)        {}
func mockRunE(_ *cobra.Command, _ []string) error { return nil }

func TestCommandReconstruction(t *testing.T) {
	root := maincommand.Command()

	type testCase struct {
		args    []string
		command string
	}
	for _, c := range []testCase{
		{
			[]string{"central", "--insecure", "whoami", "-e", "test"},
			"central whoami",
		},
		{
			[]string{"declarative-config", "create", "auth-provider", "iap", "--audience", "test"},
			"declarative-config create auth-provider iap",
		},
		{
			[]string{"central", "--insecure=false", "whoami"},
			// endpoint is always 'changed' in the common/flags/endpoint.go.
			"central whoami",
		},
		{
			[]string{"central", "db", "restore", "positional_argument"},
			"central db restore",
		},
	} {
		var command string
		once := sync.Once{}
		common.PatchPersistentPreRunHooks(root, func(cmd *cobra.Command, _ []string) {
			once.Do(func() {
				command = getCommandPath(cmd)
			})
			// Do not actually run the command:
			cmd.Run = mockRun
			cmd.RunE = mockRunE
		})

		output, err := executeCommand(root, c.args...)
		assert.NoError(t, err)
		assert.Equal(t, "", output)
		assert.Equal(t, c.command, command)
	}
}
