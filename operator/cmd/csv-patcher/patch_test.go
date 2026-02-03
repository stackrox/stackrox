package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatchCSV(t *testing.T) {
	// Set up test environment variables
	os.Setenv("RELATED_IMAGE_MAIN", "quay.io/rhacs-eng/main:4.0.0")
	os.Setenv("RELATED_IMAGE_SCANNER", "quay.io/rhacs-eng/scanner:4.0.0")
	defer func() {
		os.Unsetenv("RELATED_IMAGE_MAIN")
		os.Unsetenv("RELATED_IMAGE_SCANNER")
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
