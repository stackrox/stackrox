package search

import (
	"testing"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearchFieldsHaveProtobufTags validates that all FieldLabels defined in
// options.go that are used for a specific category have corresponding search tags
// in the protobuf definitions.
//
// This test catches bugs where:
// 1. A FieldLabel is defined in pkg/search/options.go
// 2. Code uses that FieldLabel to build queries (e.g., ForFieldLabel(search.AllowPrivilegeEscalation))
// 3. But the protobuf field is missing the search: tag
// 4. Result: Queries fail or return incorrect results
func TestSearchFieldsHaveProtobufTags(t *testing.T) {
	testCases := []struct {
		category                v1.SearchCategory
		protoType               interface{}
		categoryName            string
		expectedSearchTagFields map[string]bool
	}{
		{
			category:     v1.SearchCategory_DEPLOYMENTS,
			protoType:    (*storage.Deployment)(nil),
			categoryName: "Deployments",
			expectedSearchTagFields: map[string]bool{
				AllowPrivilegeEscalation.String(): true,
				Privileged.String():               true,
				ReadOnlyRootFilesystem.String():   true,
				AddCapabilities.String():          true,
				DropCapabilities.String():         true,

				HostNetwork.String():                  true,
				HostPID.String():                      true,
				HostIPC.String():                      true,
				AutomountServiceAccountToken.String(): true,

				Namespace.String():      true,
				Cluster.String():        true,
				DeploymentName.String(): true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.categoryName, func(t *testing.T) {
			// Build the OptionsMap by walking the protobuf structure
			// This is exactly what happens at runtime
			optionsMap := Walk(tc.category, tc.categoryName, tc.protoType)
			require.NotNil(t, optionsMap, "OptionsMap should not be nil")

			for fieldLabel, mustExist := range tc.expectedSearchTagFields {
				t.Run(fieldLabel, func(t *testing.T) {
					searchOptions, exists := optionsMap.Get(fieldLabel)

					if mustExist {
						assert.True(t, exists,
							"Field %q is missing from OptionsMap for %s. "+
								"This means the protobuf field is missing the search: tag. "+
								"Find the field in proto/storage/*.proto and add search:\"%s\" to the @gotags annotation. ",
							fieldLabel, tc.categoryName, fieldLabel, fieldLabel)

						if exists {
							assert.NotEmpty(t, searchOptions.GetFieldPath(),
								"Field %q has empty FieldPath in OptionsMap", fieldLabel)
							assert.Equal(t, tc.category, searchOptions.GetCategory(),
								"Field %q has wrong category", fieldLabel)
						}
					}
				})
			}
		})
	}
}
