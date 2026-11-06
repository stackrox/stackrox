package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// Directly usable deployment YAMLs must advance with the canonical archive.
func TestDeploymentBundlePins(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	bundle, err := os.ReadFile(filepath.Join(root, "scanner/image/scanner/bundles/ci-minimal/vulnerabilities.zip"))
	require.NoError(t, err)
	expected := fmt.Sprintf("ci-minimal-%x", sha256.Sum256(bundle))
	for name, path := range map[string]string{
		"operator":          "operator/tests/common/central-cr.yaml",
		"operator disabled": "operator/tests/common/central-cr-without-scanner-v4.yaml",
		"operator enabled":  "operator/tests/common/central-cr-with-scanner-v4-enabled-explicitly.yaml",
		"operator misc":     "operator/tests/central/central-misc/010-central-cr.yaml",
		"local Helm":        "deploy/common/local-dev-values.yaml",
	} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, path))
			require.NoError(t, err)
			require.Equal(t, []string{expected}, regexp.MustCompile(`ci-minimal-[a-f0-9]{64}`).FindAllString(string(data), -1), "update deployment pins when regenerating the bundle")
		})
	}
}
