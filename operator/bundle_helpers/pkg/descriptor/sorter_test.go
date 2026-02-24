package descriptor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFixCSVDescriptorsMap(t *testing.T) {
	tests := []struct {
		name       string
		input      map[string]any
		wantErr    bool
		errMessage string
		assertions func(t *testing.T, csvDoc map[string]any)
	}{
		{
			name: "sorts descriptors by parent path",
			input: map[string]any{
				"spec": map[string]any{
					"customresourcedefinitions": map[string]any{
						"owned": []any{
							map[string]any{
								"specDescriptors": []any{
									map[string]any{
										"path": "spec.foo.bar",
									},
									map[string]any{
										"path": "spec.foo",
									},
									map[string]any{
										"path": "spec",
									},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, csvDoc map[string]any) {
				spec := csvDoc["spec"].(map[string]any)
				crds := spec["customresourcedefinitions"].(map[string]any)
				owned := crds["owned"].([]any)
				crd := owned[0].(map[string]any)
				descriptors := crd["specDescriptors"].([]any)

				// Verify order: spec, spec.foo, spec.foo.bar
				assert.Equal(t, "spec", descriptors[0].(map[string]any)["path"])
				assert.Equal(t, "spec.foo", descriptors[1].(map[string]any)["path"])
				assert.Equal(t, "spec.foo.bar", descriptors[2].(map[string]any)["path"])
			},
		},
		{
			name: "converts relative field dependencies",
			input: map[string]any{
				"spec": map[string]any{
					"customresourcedefinitions": map[string]any{
						"owned": []any{
							map[string]any{
								"specDescriptors": []any{
									map[string]any{
										"path": "spec.parent.field",
										"x-descriptors": []any{
											"urn:alm:descriptor:com.tectonic.ui:fieldDependency:.sibling:value",
										},
									},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, csvDoc map[string]any) {
				spec := csvDoc["spec"].(map[string]any)
				crds := spec["customresourcedefinitions"].(map[string]any)
				owned := crds["owned"].([]any)
				crd := owned[0].(map[string]any)
				descriptors := crd["specDescriptors"].([]any)
				descriptor := descriptors[0].(map[string]any)
				xDescs := descriptor["x-descriptors"].([]any)

				assert.Equal(t, "urn:alm:descriptor:com.tectonic.ui:fieldDependency:spec.parent.sibling:value", xDescs[0])
			},
		},
		{
			name: "handles empty descriptors list",
			input: map[string]any{
				"spec": map[string]any{
					"customresourcedefinitions": map[string]any{
						"owned": []any{
							map[string]any{
								"specDescriptors": []any{},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, csvDoc map[string]any) {
				spec := csvDoc["spec"].(map[string]any)
				crds := spec["customresourcedefinitions"].(map[string]any)
				owned := crds["owned"].([]any)
				crd := owned[0].(map[string]any)
				descriptors := crd["specDescriptors"].([]any)

				assert.Empty(t, descriptors)
			},
		},
		{
			name: "handles missing specDescriptors",
			input: map[string]any{
				"spec": map[string]any{
					"customresourcedefinitions": map[string]any{
						"owned": []any{
							map[string]any{
								"name": "some-crd",
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, csvDoc map[string]any) {
				spec := csvDoc["spec"].(map[string]any)
				crds := spec["customresourcedefinitions"].(map[string]any)
				owned := crds["owned"].([]any)
				crd := owned[0].(map[string]any)

				_, exists := crd["specDescriptors"]
				assert.False(t, exists)
			},
		},
		{
			name: "processes multiple CRDs",
			input: map[string]any{
				"spec": map[string]any{
					"customresourcedefinitions": map[string]any{
						"owned": []any{
							map[string]any{
								"specDescriptors": []any{
									map[string]any{
										"path": "spec.b",
									},
									map[string]any{
										"path": "spec.a",
									},
								},
							},
							map[string]any{
								"specDescriptors": []any{
									map[string]any{
										"path": "spec.parent.child",
										"x-descriptors": []any{
											"urn:alm:descriptor:com.tectonic.ui:fieldDependency:.other:val",
										},
									},
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, csvDoc map[string]any) {
				spec := csvDoc["spec"].(map[string]any)
				crds := spec["customresourcedefinitions"].(map[string]any)
				owned := crds["owned"].([]any)

				// Check first CRD - stable sort keeps original order for same parent
				crd1 := owned[0].(map[string]any)
				descriptors1 := crd1["specDescriptors"].([]any)
				assert.Equal(t, "spec.b", descriptors1[0].(map[string]any)["path"])
				assert.Equal(t, "spec.a", descriptors1[1].(map[string]any)["path"])

				// Check second CRD path conversion
				crd2 := owned[1].(map[string]any)
				descriptors2 := crd2["specDescriptors"].([]any)
				descriptor := descriptors2[0].(map[string]any)
				xDescs := descriptor["x-descriptors"].([]any)
				assert.Equal(t, "urn:alm:descriptor:com.tectonic.ui:fieldDependency:spec.parent.other:val", xDescs[0])
			},
		},
		{
			name: "errors on invalid spec type",
			input: map[string]any{
				"spec": "invalid",
			},
			wantErr:    true,
			errMessage: "spec not found or not a map",
		},
		{
			name: "errors on invalid customresourcedefinitions type",
			input: map[string]any{
				"spec": map[string]any{
					"customresourcedefinitions": "invalid",
				},
			},
			wantErr:    true,
			errMessage: "customresourcedefinitions not found or not a map",
		},
		{
			name: "errors on invalid owned type",
			input: map[string]any{
				"spec": map[string]any{
					"customresourcedefinitions": map[string]any{
						"owned": map[string]any{
							"invalid": "structure",
						},
					},
				},
			},
			wantErr:    true,
			errMessage: "owned not found or not a list",
		},
		{
			name: "errors on invalid specDescriptors type",
			input: map[string]any{
				"spec": map[string]any{
					"customresourcedefinitions": map[string]any{
						"owned": []any{
							map[string]any{
								"specDescriptors": "invalid",
							},
						},
					},
				},
			},
			wantErr:    true,
			errMessage: "specDescriptors is not a list",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FixCSVDescriptorsMap(tt.input)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMessage)
				return
			}
			require.NoError(t, err)
			if tt.assertions != nil {
				tt.assertions(t, tt.input)
			}
		})
	}
}
