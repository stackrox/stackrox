package csv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chartutil"
)

func mustReadValues(t *testing.T, s string) chartutil.Values {
	t.Helper()
	v, err := chartutil.ReadValues([]byte(s))
	require.NoError(t, err)
	return v
}

func TestPatchCSV(t *testing.T) {
	t.Setenv("RELATED_IMAGE_MAIN", "quay.io/rhacs-eng/main:4.0.0")
	t.Setenv("RELATED_IMAGE_SCANNER", "quay.io/rhacs-eng/scanner:4.0.0")

	tests := []struct {
		name       string
		input      chartutil.Values
		opts       PatchOptions
		wantErr    bool
		assertions func(t *testing.T, result chartutil.Values)
	}{
		{
			name: "basic version patching",
			input: mustReadValues(t, `
metadata:
  name: rhacs-operator.v0.0.1
  annotations:
    containerImage: quay.io/stackrox-io/stackrox-operator:0.0.1
    createdAt: ""
spec:
  version: "0.0.1"
  customresourcedefinitions:
    owned: []
`),
			opts: PatchOptions{
				Version:           "4.0.0",
				OperatorImage:     "quay.io/stackrox-io/stackrox-operator:4.0.0",
				FirstVersion:      "3.62.0",
				RelatedImagesMode: "omit",
			},
			assertions: func(t *testing.T, result chartutil.Values) {
				metadata := result["metadata"].(map[string]any)
				assert.Equal(t, "rhacs-operator.v4.0.0", metadata["name"])

				annotations := metadata["annotations"].(map[string]any)
				assert.Equal(t, "quay.io/stackrox-io/stackrox-operator:4.0.0", annotations["containerImage"])
				assert.NotEmpty(t, annotations["createdAt"])

				spec := result["spec"].(map[string]any)
				assert.Equal(t, "4.0.0", spec["version"])
			},
		},
		{
			name: "replaces version calculation",
			input: mustReadValues(t, `
metadata:
  name: rhacs-operator.v0.0.1
  annotations:
    containerImage: quay.io/stackrox-io/stackrox-operator:0.0.1
spec:
  version: "0.0.1"
  customresourcedefinitions:
    owned: []
`),
			opts: PatchOptions{
				Version:           "4.0.1",
				OperatorImage:     "quay.io/stackrox-io/stackrox-operator:4.0.1",
				FirstVersion:      "4.0.0",
				RelatedImagesMode: "omit",
			},
			assertions: func(t *testing.T, result chartutil.Values) {
				spec := result["spec"].(map[string]any)
				assert.Equal(t, "rhacs-operator.v4.0.0", spec["replaces"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PatchCSV(tt.input, tt.opts)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tt.assertions != nil {
				tt.assertions(t, tt.input)
			}
		})
	}
}

func TestInjectRelatedImageEnvVars_SingleEnvVar(t *testing.T) {
	t.Setenv("RELATED_IMAGE_MAIN", "quay.io/rhacs-eng/main:4.5.0")

	spec := mustReadValues(t, `
install:
  spec:
    deployments:
    - spec:
        template:
          spec:
            containers:
            - env:
              - name: RELATED_IMAGE_MAIN
`)

	err := injectRelatedImageEnvVars(spec)
	require.NoError(t, err)

	// Verify the value was injected
	deployments := spec["install"].(map[string]any)["spec"].(map[string]any)["deployments"].([]any)
	deployment := deployments[0].(map[string]any)
	containers := deployment["spec"].(map[string]any)["template"].(map[string]any)["spec"].(map[string]any)["containers"].([]any)
	container := containers[0].(map[string]any)
	env := container["env"].([]any)
	envVar := env[0].(map[string]any)

	assert.Equal(t, "RELATED_IMAGE_MAIN", envVar["name"])
	assert.Equal(t, "quay.io/rhacs-eng/main:4.5.0", envVar["value"])
}

func TestInjectRelatedImageEnvVars_MultipleNested(t *testing.T) {
	t.Setenv("RELATED_IMAGE_MAIN", "quay.io/rhacs-eng/main:4.5.0")
	t.Setenv("RELATED_IMAGE_SCANNER", "quay.io/rhacs-eng/scanner:4.5.0")
	t.Setenv("RELATED_IMAGE_SCANNER_DB", "quay.io/rhacs-eng/scanner-db:4.5.0")

	spec := mustReadValues(t, `
install:
  spec:
    deployments:
    - spec:
        template:
          spec:
            containers:
            - env:
              - name: RELATED_IMAGE_MAIN
              - name: RELATED_IMAGE_SCANNER
              - name: RELATED_IMAGE_SCANNER_DB
`)

	err := injectRelatedImageEnvVars(spec)
	require.NoError(t, err)

	// Verify all values were injected
	deployments := spec["install"].(map[string]any)["spec"].(map[string]any)["deployments"].([]any)
	deployment := deployments[0].(map[string]any)
	containers := deployment["spec"].(map[string]any)["template"].(map[string]any)["spec"].(map[string]any)["containers"].([]any)
	container := containers[0].(map[string]any)
	env := container["env"].([]any)

	assert.Equal(t, "quay.io/rhacs-eng/main:4.5.0", env[0].(map[string]any)["value"])
	assert.Equal(t, "quay.io/rhacs-eng/scanner:4.5.0", env[1].(map[string]any)["value"])
	assert.Equal(t, "quay.io/rhacs-eng/scanner-db:4.5.0", env[2].(map[string]any)["value"])
}

func TestInjectRelatedImageEnvVars_MissingEnvVar(t *testing.T) {
	spec := mustReadValues(t, `
install:
  spec:
    deployments:
    - spec:
        template:
          spec:
            containers:
            - env:
              - name: RELATED_IMAGE_NONEXISTENT
`)

	err := injectRelatedImageEnvVars(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RELATED_IMAGE_NONEXISTENT")
	assert.Contains(t, err.Error(), "not set")
}

func TestInjectRelatedImageEnvVars_NoRelatedImages(t *testing.T) {
	spec := mustReadValues(t, `
install:
  spec:
    deployments:
    - spec:
        template:
          spec:
            containers:
            - env:
              - name: SOME_OTHER_VAR
                value: some-value
`)

	err := injectRelatedImageEnvVars(spec)
	require.NoError(t, err)

	// Verify spec is unchanged
	deployments := spec["install"].(map[string]any)["spec"].(map[string]any)["deployments"].([]any)
	deployment := deployments[0].(map[string]any)
	containers := deployment["spec"].(map[string]any)["template"].(map[string]any)["spec"].(map[string]any)["containers"].([]any)
	container := containers[0].(map[string]any)
	env := container["env"].([]any)
	envVar := env[0].(map[string]any)

	assert.Equal(t, "SOME_OTHER_VAR", envVar["name"])
	assert.Equal(t, "some-value", envVar["value"])
}

func TestConstructRelatedImages_MultipleEnvVars(t *testing.T) {
	t.Setenv("RELATED_IMAGE_MAIN", "quay.io/rhacs-eng/main:4.5.0")
	t.Setenv("RELATED_IMAGE_SCANNER", "quay.io/rhacs-eng/scanner:4.5.0")

	spec := map[string]any{}
	managerImage := "quay.io/rhacs-eng/rhacs-operator:4.5.0"

	err := constructRelatedImages(spec, managerImage)
	require.NoError(t, err)

	// Verify relatedImages was created
	relatedImages, ok := spec["relatedImages"].([]map[string]any)
	require.True(t, ok, "relatedImages should be a []map[string]any")
	require.GreaterOrEqual(t, len(relatedImages), 3, "should have at least 3 entries (main, scanner, manager)")

	// Find entries by name
	imagesByName := make(map[string]string)
	for _, img := range relatedImages {
		name := img["name"].(string)
		image := img["image"].(string)
		imagesByName[name] = image
	}

	// Verify entries
	assert.Equal(t, "quay.io/rhacs-eng/main:4.5.0", imagesByName["main"])
	assert.Equal(t, "quay.io/rhacs-eng/scanner:4.5.0", imagesByName["scanner"])
	assert.Equal(t, "quay.io/rhacs-eng/rhacs-operator:4.5.0", imagesByName["manager"])
}

func TestConstructRelatedImages_NoEnvVars(t *testing.T) {
	spec := map[string]any{}
	managerImage := "quay.io/rhacs-eng/rhacs-operator:4.5.0"

	err := constructRelatedImages(spec, managerImage)
	require.NoError(t, err)

	// Verify only manager entry exists
	relatedImages, ok := spec["relatedImages"].([]map[string]any)
	require.True(t, ok)
	require.Equal(t, 1, len(relatedImages), "should only have manager entry")

	assert.Equal(t, "manager", relatedImages[0]["name"])
	assert.Equal(t, "quay.io/rhacs-eng/rhacs-operator:4.5.0", relatedImages[0]["image"])
}

func TestConstructRelatedImages_NameTransformation(t *testing.T) {
	t.Setenv("RELATED_IMAGE_SCANNER_DB_SLIM", "quay.io/rhacs-eng/scanner-db-slim:4.5.0")

	spec := map[string]any{}
	managerImage := "quay.io/rhacs-eng/rhacs-operator:4.5.0"

	err := constructRelatedImages(spec, managerImage)
	require.NoError(t, err)

	// Verify name transformation to lowercase
	relatedImages, ok := spec["relatedImages"].([]map[string]any)
	require.True(t, ok)

	// Find the scanner_db_slim entry
	found := false
	for _, img := range relatedImages {
		if img["name"].(string) == "scanner_db_slim" {
			found = true
			assert.Equal(t, "quay.io/rhacs-eng/scanner-db-slim:4.5.0", img["image"])
			break
		}
	}
	assert.True(t, found, "should have scanner_db_slim entry with lowercase name")
}
