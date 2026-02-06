package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatchCSV(t *testing.T) {
	// Set up test environment variables
	require.NoError(t, os.Setenv("RELATED_IMAGE_MAIN", "quay.io/rhacs-eng/main:4.0.0"))
	require.NoError(t, os.Setenv("RELATED_IMAGE_SCANNER", "quay.io/rhacs-eng/scanner:4.0.0"))
	defer func() {
		require.NoError(t, os.Unsetenv("RELATED_IMAGE_MAIN"))
		require.NoError(t, os.Unsetenv("RELATED_IMAGE_SCANNER"))
	}()

	tests := []struct {
		name       string
		input      map[string]interface{}
		opts       PatchOptions
		wantErr    bool
		assertions func(t *testing.T, result map[string]interface{})
	}{
		{
			name: "basic version patching",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "rhacs-operator.v0.0.1",
					"annotations": map[string]interface{}{
						"containerImage": "quay.io/stackrox-io/stackrox-operator:0.0.1",
						"createdAt":      "",
					},
				},
				"spec": map[string]interface{}{
					"version": "0.0.1",
					"customresourcedefinitions": map[string]interface{}{
						"owned": []interface{}{},
					},
				},
			},
			opts: PatchOptions{
				Version:           "4.0.0",
				OperatorImage:     "quay.io/stackrox-io/stackrox-operator:4.0.0",
				FirstVersion:      "3.62.0",
				RelatedImagesMode: "omit",
			},
			assertions: func(t *testing.T, result map[string]interface{}) {
				metadata := result["metadata"].(map[string]interface{})
				assert.Equal(t, "rhacs-operator.v4.0.0", metadata["name"])

				annotations := metadata["annotations"].(map[string]interface{})
				assert.Equal(t, "quay.io/stackrox-io/stackrox-operator:4.0.0", annotations["containerImage"])
				assert.NotEmpty(t, annotations["createdAt"])

				spec := result["spec"].(map[string]interface{})
				assert.Equal(t, "4.0.0", spec["version"])
			},
		},
		{
			name: "replaces version calculation",
			input: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "rhacs-operator.v0.0.1",
					"annotations": map[string]interface{}{
						"containerImage": "quay.io/stackrox-io/stackrox-operator:0.0.1",
					},
				},
				"spec": map[string]interface{}{
					"version": "0.0.1",
					"customresourcedefinitions": map[string]interface{}{
						"owned": []interface{}{},
					},
				},
			},
			opts: PatchOptions{
				Version:           "4.0.1",
				OperatorImage:     "quay.io/stackrox-io/stackrox-operator:4.0.1",
				FirstVersion:      "4.0.0",
				RelatedImagesMode: "omit",
			},
			assertions: func(t *testing.T, result map[string]interface{}) {
				spec := result["spec"].(map[string]interface{})
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
	// Set up environment variable
	require.NoError(t, os.Setenv("RELATED_IMAGE_MAIN", "quay.io/rhacs-eng/main:4.5.0"))
	defer func() {
		require.NoError(t, os.Unsetenv("RELATED_IMAGE_MAIN"))
	}()

	spec := map[string]interface{}{
		"install": map[string]interface{}{
			"spec": map[string]interface{}{
				"deployments": []interface{}{
					map[string]interface{}{
						"spec": map[string]interface{}{
							"template": map[string]interface{}{
								"spec": map[string]interface{}{
									"containers": []interface{}{
										map[string]interface{}{
											"env": []interface{}{
												map[string]interface{}{
													"name": "RELATED_IMAGE_MAIN",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	err := injectRelatedImageEnvVars(spec)
	require.NoError(t, err)

	// Verify the value was injected
	deployments := spec["install"].(map[string]interface{})["spec"].(map[string]interface{})["deployments"].([]interface{})
	deployment := deployments[0].(map[string]interface{})
	containers := deployment["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["containers"].([]interface{})
	container := containers[0].(map[string]interface{})
	env := container["env"].([]interface{})
	envVar := env[0].(map[string]interface{})

	assert.Equal(t, "RELATED_IMAGE_MAIN", envVar["name"])
	assert.Equal(t, "quay.io/rhacs-eng/main:4.5.0", envVar["value"])
}

func TestInjectRelatedImageEnvVars_MultipleNested(t *testing.T) {
	// Set up multiple environment variables
	require.NoError(t, os.Setenv("RELATED_IMAGE_MAIN", "quay.io/rhacs-eng/main:4.5.0"))
	require.NoError(t, os.Setenv("RELATED_IMAGE_SCANNER", "quay.io/rhacs-eng/scanner:4.5.0"))
	require.NoError(t, os.Setenv("RELATED_IMAGE_SCANNER_DB", "quay.io/rhacs-eng/scanner-db:4.5.0"))
	defer func() {
		require.NoError(t, os.Unsetenv("RELATED_IMAGE_MAIN"))
		require.NoError(t, os.Unsetenv("RELATED_IMAGE_SCANNER"))
		require.NoError(t, os.Unsetenv("RELATED_IMAGE_SCANNER_DB"))
	}()

	spec := map[string]interface{}{
		"install": map[string]interface{}{
			"spec": map[string]interface{}{
				"deployments": []interface{}{
					map[string]interface{}{
						"spec": map[string]interface{}{
							"template": map[string]interface{}{
								"spec": map[string]interface{}{
									"containers": []interface{}{
										map[string]interface{}{
											"env": []interface{}{
												map[string]interface{}{
													"name": "RELATED_IMAGE_MAIN",
												},
												map[string]interface{}{
													"name": "RELATED_IMAGE_SCANNER",
												},
												map[string]interface{}{
													"name": "RELATED_IMAGE_SCANNER_DB",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	err := injectRelatedImageEnvVars(spec)
	require.NoError(t, err)

	// Verify all values were injected
	deployments := spec["install"].(map[string]interface{})["spec"].(map[string]interface{})["deployments"].([]interface{})
	deployment := deployments[0].(map[string]interface{})
	containers := deployment["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["containers"].([]interface{})
	container := containers[0].(map[string]interface{})
	env := container["env"].([]interface{})

	assert.Equal(t, "quay.io/rhacs-eng/main:4.5.0", env[0].(map[string]interface{})["value"])
	assert.Equal(t, "quay.io/rhacs-eng/scanner:4.5.0", env[1].(map[string]interface{})["value"])
	assert.Equal(t, "quay.io/rhacs-eng/scanner-db:4.5.0", env[2].(map[string]interface{})["value"])
}

func TestInjectRelatedImageEnvVars_MissingEnvVar(t *testing.T) {
	spec := map[string]interface{}{
		"install": map[string]interface{}{
			"spec": map[string]interface{}{
				"deployments": []interface{}{
					map[string]interface{}{
						"spec": map[string]interface{}{
							"template": map[string]interface{}{
								"spec": map[string]interface{}{
									"containers": []interface{}{
										map[string]interface{}{
											"env": []interface{}{
												map[string]interface{}{
													"name": "RELATED_IMAGE_NONEXISTENT",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	err := injectRelatedImageEnvVars(spec)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RELATED_IMAGE_NONEXISTENT")
	assert.Contains(t, err.Error(), "not set")
}

func TestInjectRelatedImageEnvVars_NoRelatedImages(t *testing.T) {
	spec := map[string]interface{}{
		"install": map[string]interface{}{
			"spec": map[string]interface{}{
				"deployments": []interface{}{
					map[string]interface{}{
						"spec": map[string]interface{}{
							"template": map[string]interface{}{
								"spec": map[string]interface{}{
									"containers": []interface{}{
										map[string]interface{}{
											"env": []interface{}{
												map[string]interface{}{
													"name":  "SOME_OTHER_VAR",
													"value": "some-value",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	err := injectRelatedImageEnvVars(spec)
	require.NoError(t, err)

	// Verify spec is unchanged
	deployments := spec["install"].(map[string]interface{})["spec"].(map[string]interface{})["deployments"].([]interface{})
	deployment := deployments[0].(map[string]interface{})
	containers := deployment["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["containers"].([]interface{})
	container := containers[0].(map[string]interface{})
	env := container["env"].([]interface{})
	envVar := env[0].(map[string]interface{})

	assert.Equal(t, "SOME_OTHER_VAR", envVar["name"])
	assert.Equal(t, "some-value", envVar["value"])
}

func TestConstructRelatedImages_MultipleEnvVars(t *testing.T) {
	// Set up environment variables
	require.NoError(t, os.Setenv("RELATED_IMAGE_MAIN", "quay.io/rhacs-eng/main:4.5.0"))
	require.NoError(t, os.Setenv("RELATED_IMAGE_SCANNER", "quay.io/rhacs-eng/scanner:4.5.0"))
	defer func() {
		require.NoError(t, os.Unsetenv("RELATED_IMAGE_MAIN"))
		require.NoError(t, os.Unsetenv("RELATED_IMAGE_SCANNER"))
	}()

	spec := map[string]interface{}{}
	managerImage := "quay.io/rhacs-eng/rhacs-operator:4.5.0"

	err := constructRelatedImages(spec, managerImage)
	require.NoError(t, err)

	// Verify relatedImages was created
	relatedImages, ok := spec["relatedImages"].([]map[string]interface{})
	require.True(t, ok, "relatedImages should be a []map[string]interface{}")
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
	// Ensure no RELATED_IMAGE_* env vars are set
	// Note: We can't unset all env vars, but we can verify behavior with none matching our pattern
	originalEnv := os.Environ()
	for _, envVar := range originalEnv {
		if len(envVar) > 14 && envVar[:14] == "RELATED_IMAGE_" {
			parts := strings.SplitN(envVar, "=", 2)
			os.Unsetenv(parts[0])
			defer os.Setenv(parts[0], parts[1])
		}
	}

	spec := map[string]interface{}{}
	managerImage := "quay.io/rhacs-eng/rhacs-operator:4.5.0"

	err := constructRelatedImages(spec, managerImage)
	require.NoError(t, err)

	// Verify only manager entry exists
	relatedImages, ok := spec["relatedImages"].([]map[string]interface{})
	require.True(t, ok)
	require.Equal(t, 1, len(relatedImages), "should only have manager entry")

	assert.Equal(t, "manager", relatedImages[0]["name"])
	assert.Equal(t, "quay.io/rhacs-eng/rhacs-operator:4.5.0", relatedImages[0]["image"])
}

func TestConstructRelatedImages_NameTransformation(t *testing.T) {
	// Set up environment variable with underscores
	require.NoError(t, os.Setenv("RELATED_IMAGE_SCANNER_DB_SLIM", "quay.io/rhacs-eng/scanner-db-slim:4.5.0"))
	defer func() {
		require.NoError(t, os.Unsetenv("RELATED_IMAGE_SCANNER_DB_SLIM"))
	}()

	spec := map[string]interface{}{}
	managerImage := "quay.io/rhacs-eng/rhacs-operator:4.5.0"

	err := constructRelatedImages(spec, managerImage)
	require.NoError(t, err)

	// Verify name transformation to lowercase
	relatedImages, ok := spec["relatedImages"].([]map[string]interface{})
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
