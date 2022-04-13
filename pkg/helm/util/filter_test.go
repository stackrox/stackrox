package util

import (
	"fmt"
	"testing"

	"github.com/stackrox/stackrox/pkg/helm/util/internal/ignore"
	"github.com/stackrox/stackrox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func TestFilterFiles_Helmignore(t *testing.T) {
	r := ignore.Empty()
	r.AddDefaults()

	require.True(t, r.Ignore("templates/.dotfile.yaml", fakeFileInfo("templates/.dotfile.yaml")))
	fileList := []string{
		"Chart.yaml",
		"values.yaml",
		"templates/NOTES.txt",
		"templates/_helpers.tpl",
		"templates/deployment.yaml",
		"templates/.dotfile.yaml",
		"secrets/values.yaml",
		"misc/some/secret/values.yaml",
		"misc/something/else",
		"files/some",
	}

	cases := []struct {
		rules           string
		expectedIgnored []string
	}{
		{
			rules:           "", // no rules - apply defaults
			expectedIgnored: []string{"templates/.dotfile.yaml"},
		},
		{
			rules: "# empty ignore file",
		},
		{
			rules:           "*.yaml",
			expectedIgnored: []string{"Chart.yaml", "values.yaml", "templates/deployment.yaml", "templates/.dotfile.yaml", "secrets/values.yaml", "misc/some/secret/values.yaml"},
		},
		{
			rules:           "values.yaml",
			expectedIgnored: []string{"values.yaml", "secrets/values.yaml", "misc/some/secret/values.yaml"},
		},
		{
			rules:           "/values.yaml",
			expectedIgnored: []string{"values.yaml"},
		},
		{
			rules:           "some",
			expectedIgnored: []string{"misc/some/secret/values.yaml", "files/some"},
		},
		{
			rules:           "some/",
			expectedIgnored: []string{"misc/some/secret/values.yaml"},
		},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("rule %q", c.rules), func(t *testing.T) {
			var files []*loader.BufferedFile
			for _, name := range fileList {
				files = append(files, &loader.BufferedFile{
					Name: name,
					Data: []byte(name),
				})
			}
			if c.rules != "" {
				files = append(files, &loader.BufferedFile{
					Name: ignore.HelmIgnore,
					Data: []byte(c.rules + "\n"),
				})
			}

			ignoredFiles := set.NewStringSet()
			for _, f := range files {
				ignoredFiles.Add(f.Name)
			}

			filtered, err := FilterFiles(files)
			require.NoError(t, err)

			for _, f := range filtered {
				expectedData := f.Name
				if f.Name == ignore.HelmIgnore {
					expectedData = c.rules + "\n"
				}
				assert.Equal(t, expectedData, string(f.Data))
				assert.Truef(t, ignoredFiles.Remove(f.Name), "unknown or duplicated file %q in filtered output", f.Name)
			}

			assert.ElementsMatch(t, c.expectedIgnored, ignoredFiles.AsSlice())
		})
	}
}
