package main

import (
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
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
	AddMissingDefaultsToFlagUsage(root)

	type testCase struct {
		args    []string
		command string
	}
	for _, c := range []testCase{
		{
			[]string{"central", "--insecure", "whoami", "-e", "test"},
			"central whoami --endpoint ... --insecure true",
		},
		{
			[]string{"declarative-config", "create", "auth-provider", "iap", "--audience", "test"},
			"declarative-config create auth-provider iap --audience ...",
		},
	} {
		var command []string
		once := sync.Once{}
		common.PatchPersistentPreRunHooks(root, func(cmd *cobra.Command, args []string) {
			once.Do(func() {
				command = reconstructCommand(cmd)
			})
			// Do not actually run the command:
			cmd.Run = mockRun
			cmd.RunE = mockRunE
		})

		output, err := executeCommand(root, c.args...)
		assert.NoError(t, err)
		assert.Equal(t, "", output)
		assert.Equal(t, c.command, strings.Join(command, " "))
	}
}
