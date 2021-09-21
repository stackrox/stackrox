package completion

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompletionCommand_InvalidArgs(t *testing.T) {
	cases := map[string]struct {
		args []string
		err  error
	}{
		"no args given": {
			args: []string{},
			err:  errors.New("Missing argument. Use one of the following: [bash|zsh|fish|powershell]"),
		},
		"invalid args given": {
			args: []string{"oh-my-zsh"},
			err:  errInvalidArgs,
		},
		"more than 1 arg given": {
			args: []string{"zhs", "oh-my-zsh"},
			err:  errors.New("Missing argument. Use one of the following: [bash|zsh|fish|powershell]"),
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cmd := Command()
			cmd.SetArgs(c.args)
			err := cmd.Execute()
			assert.Equal(t, c.err, err, "expected %v to match %v", err, errInvalidArgs)
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
			cmd := Command()
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs(c.args)
			assert.NoErrorf(t, cmd.Execute(), "completion for %q failed", c.args[0])
		})
	}
}
