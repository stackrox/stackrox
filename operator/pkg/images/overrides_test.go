package images

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
)

func TestToValues(t *testing.T) {
	testOverrides := Overrides{
		Main:      "central.image.fullRef",
		Scanner:   "scanner.image.fullRef",
		ScannerDB: "scanner.dbImage.fullRef",
	}

	t.Setenv(Main.EnvVar(), "override-main")
	t.Setenv(Scanner.EnvVar(), "")
	t.Setenv(ScannerDB.EnvVar(), "")

	vals, err := testOverrides.ToValues()
	require.NoError(t, err)

	expectedVals := chartutil.Values{
		"central": map[string]interface{}{
			"image": map[string]interface{}{
				"fullRef": "override-main",
			},
		},
	}

	assert.Equal(t, expectedVals, vals)
}
