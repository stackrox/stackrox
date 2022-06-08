package tests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersions(t *testing.T) {
	t.Parallel()

	client := centralgrpc.HTTPClientForCentral(t)

	resp, err := client.Get("/debug/versions.json")
	require.NoError(t, err)
	defer utils.IgnoreError(resp.Body.Close)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var versions version.Versions
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&versions))

	kind := version.GetVersionKind(versions.MainVersion)
	require.NotEqualf(t, version.InvalidKind, kind, "invalid main version %s", versions.MainVersion)
	if kind == version.DevelopmentKind || kind == version.NightlyKind {
		t.Skip("nothing to be checked on development versions")
	}

	assert.Equal(t, kind, version.GetVersionKind(versions.CollectorVersion), "rc and release builds should reference a corresponding collector version")
	assert.Equal(t, kind, version.GetVersionKind(versions.ScannerVersion), "rc and release builds should reference a corresponding scanner version")
}
