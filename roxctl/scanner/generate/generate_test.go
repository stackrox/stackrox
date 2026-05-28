package generate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScannerGenerateValidation(t *testing.T) {
	t.Run("validate succeeds with default params", func(t *testing.T) {
		cmd := scannerGenerateCommand{}
		require.NoError(t, cmd.validate())
	})
}
