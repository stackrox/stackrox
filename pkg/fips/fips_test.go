package fips

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireEnabled(t *testing.T) {
	if os.Getenv("ROX_TEST_FIPS_SUBPROCESS") == "1" {
		RequireEnabled()
		return
	}

	for name, tc := range map[string]struct {
		godebug string
		wantErr bool
	}{
		"default":  {},
		"enabled":  {godebug: "fips140=on"},
		"disabled": {godebug: "fips140=off", wantErr: true},
	} {
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestRequireEnabled$")
			cmd.Env = append(os.Environ(), "ROX_TEST_FIPS_SUBPROCESS=1", "GODEBUG="+tc.godebug)
			output, err := cmd.CombinedOutput()
			if tc.wantErr {
				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr)
				assert.Equal(t, 1, exitErr.ExitCode())
				assert.Contains(t, string(output), "FIPS 140-3 mode is required")
			} else {
				require.NoError(t, err, "%s", output)
			}
		})
	}
}
