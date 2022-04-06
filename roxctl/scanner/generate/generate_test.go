package generate

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/apiparams"
	"github.com/stackrox/rox/pkg/istioutils"
	"github.com/stretchr/testify/require"
)

func TestScannerGenerateValidation(t *testing.T) {
	t.Run("not supported Istio version", func(t *testing.T) {
		cmdBadIstio := scannerGenerateCommand{apiParams: apiparams.Scanner{IstioVersion: "0.1.0"}}

		expectedErrorStr := fmt.Sprintf(
			"invalid arguments: unsupported Istio version %q used for argument %q. Use one of the following: [%s]",
			"0.1.0", "--"+istioSupportArg, strings.Join(istioutils.ListKnownIstioVersions(), "|"),
		)

		require.EqualError(t, cmdBadIstio.validate(), expectedErrorStr)
	})

	t.Run("supported Istio version", func(t *testing.T) {
		cmd := scannerGenerateCommand{apiParams: apiparams.Scanner{IstioVersion: istioutils.ListKnownIstioVersions()[0]}}

		require.Nil(t, cmd.validate())
	})

	t.Run("not provided Istio version", func(t *testing.T) {
		cmd := scannerGenerateCommand{}

		require.Nil(t, cmd.validate())
	})
}
