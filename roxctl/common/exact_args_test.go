//go:build test_all

package common

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestExactArgsWithCustomErrMessage(t *testing.T) {
	dummyCmd := &cobra.Command{
		Use: "dummy",
	}

	tests := map[string]struct {
		errMsg       string
		args         []string
		expectedArgs int
		shouldFail   bool
	}{
		"exact args without error": {
			expectedArgs: 1,
			args:         []string{"one"},
			errMsg:       "",
			shouldFail:   false,
		},
		"non matching args with custom error": {
			expectedArgs: 2,
			args:         []string{"one"},
			errMsg:       "custom err",
			shouldFail:   true,
		},
	}

	for name, data := range tests {
		fn := ExactArgsWithCustomErrMessage(data.expectedArgs, data.errMsg)
		err := fn(dummyCmd, data.args)
		if data.shouldFail {
			assert.Errorf(t, err, "expected an error for test case %q", name)
			if err != nil {
				assert.Equalf(t, data.errMsg, err.Error(), "expected error message to match for test case %q", name)
			}
		} else {
			assert.NoErrorf(t, err, "expected no error for test case %q", name)
		}
	}
}
