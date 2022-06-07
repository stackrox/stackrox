package completion

import (
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
)

func TestCompletionCommand_InvalidArgs(t *testing.T) {
	cases := map[string]struct {
		args []string
		err  error
	}{
		"no args given": {
			args: []string{},
			err:  errox.InvalidArgs,
		},
		"invalid args given": {
			args: []string{"oh-my-zsh"},
			err:  errInvalidArgs,
		},
		"more than 1 arg given": {
			args: []string{"zhs", "oh-my-zsh"},
			err:  errox.InvalidArgs,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			io, _, _, _ := common.TestIO()
			cmd := Command(common.NewTestCLIEnvironment(t, io, nil))
			cmd.SetArgs(c.args)
			err := cmd.Execute()
			assert.Error(t, err)
			assert.ErrorIs(t, err, c.err)
		})
	}
}

func TestCompletionCommand_Success(t *testing.T) {
	cases := map[string]struct {
		args []string
	}{
		"bash completion": {
			args: []string{"bash"},
		},
		"zsh completion": {
			args: []string{"zsh"},
		},
		"fish completion": {
			args: []string{"fish"},
		},
		"powershell completion": {
			args: []string{"powershell"},
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			io, _, _, _ := common.TestIO()
			cmd := Command(common.NewTestCLIEnvironment(t, io, printer.DefaultColorPrinter()))
			cmd.SetArgs(c.args)
			assert.NoErrorf(t, cmd.Execute(), "completion for %q failed", c.args[0])
		})
	}
}
