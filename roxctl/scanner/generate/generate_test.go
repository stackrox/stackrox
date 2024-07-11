package generate

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScannerGenerateValidation(t *testing.T) {
	t.Run("not supported Istio version", func(t *testing.T) {
		cmdBadIstio := scannerGenerateCommand{apiParams: apiparams.Scanner{IstioVersion: "0.1.0"}}

		actualErr := cmdBadIstio.validate()
		require.Error(t, actualErr)
		assert.ErrorIs(t, actualErr, errox.InvalidArgs)

		expectedListOfIstioVersions := strings.Join(istioutils.ListKnownIstioVersions(), "|")
		assert.Contains(t, actualErr.Error(), expectedListOfIstioVersions)
	})

	t.Run("supported Istio version", func(t *testing.T) {
		cmd := scannerGenerateCommand{apiParams: apiparams.Scanner{IstioVersion: istioutils.ListKnownIstioVersions()[0]}}

		require.NoError(t, cmd.validate())
	})

	t.Run("not provided Istio version", func(t *testing.T) {
		cmd := scannerGenerateCommand{}

		require.NoError(t, cmd.validate())
	})
}
