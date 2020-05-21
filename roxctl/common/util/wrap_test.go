package util

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRunENoArgs(t *testing.T) {
	dummyFunc := func(*cobra.Command) error { return nil }
	for _, testCase := range []struct {
		name string
		args []string
		err  bool
	}{
		{"empty", []string{}, false},
		{"one arg", []string{""}, true},
		{"two args", []string{"a", "b"}, true},
		{"three args", []string{"a", "b", "c"}, true},
	} {
		c := testCase
		t.Run(c.name, func(t *testing.T) {
			runE := RunENoArgs(dummyFunc)
			if c.err {
				assert.Error(t, runE(nil, c.args))
			} else {
				assert.NoError(t, runE(nil, c.args))
			}
		})
	}
}
