package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLabelsJSON(t *testing.T) {
	t.Parallel()

	lb, path, err := parseLabelsJSON("testdata-rhcos")
	require.NoError(t, err)
	assert.Contains(t, path, filepath.Join("usr", "share", "buildinfo", "labels.json"))
	assert.Equal(t, "openshift/ose-rhel-coreos-9", lb.Name)
	assert.Equal(t, "x86_64", lb.Architecture)
	assert.Equal(t, "cpe:/a:redhat:openshift:4.21::el9", lb.CPE)
	assert.False(t, lb.Created.IsZero())
}

func TestParseLabelsJSONNotFound(t *testing.T) {
	t.Parallel()

	_, _, err := parseLabelsJSON(t.TempDir())
	require.ErrorIs(t, err, errLabelsNotFound)
}

func TestParseLabelsJSONQuotedValues(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "usr", "share", "buildinfo")
	require.NoError(t, os.MkdirAll(path, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(path, "labels.json"), []byte(`{
		"name": "\"openshift/ose-rhel-coreos-9\"",
		"org.opencontainers.image.created": "2025-03-24T10:34:00Z",
		"cpe": "\"cpe:/a:redhat:openshift:4.21::el9\"",
		"architecture": "\"x86_64\""
	}`), 0o644))

	lb, _, err := parseLabelsJSON(dir)
	require.NoError(t, err)
	assert.Equal(t, "openshift/ose-rhel-coreos-9", lb.Name)
	assert.Equal(t, "x86_64", lb.Architecture)
	assert.Equal(t, "cpe:/a:redhat:openshift:4.21::el9", lb.CPE)
}

func TestResolveBuildLabelsFallsBackToOSReleasePath(t *testing.T) {
	t.Parallel()

	lb := resolveBuildLabels("testdata", "testdata-rhcos")
	assert.Equal(t, "openshift/ose-rhel-coreos-9", lb.Name)
}
