package storagetoeffectiveaccessscope

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespaces(t *testing.T) {
	for name, tc := range map[string]struct {
		input []*storage.NamespaceMetadata
	}{
		"nil input": {
			input: nil,
		},
		"single namespace": {
			input: []*storage.NamespaceMetadata{
				{
					Id:   "namespace1",
					Name: "namespace1",
				},
			},
		},
		"multiple clusters": {
			input: []*storage.NamespaceMetadata{
				{
					Id:          "namespace1",
					Name:        "Namespace 1",
					ClusterName: "Cluster 1",
				},
				{
					Id:          "namespace2",
					Name:        "Namespace 2",
					ClusterName: "Cluster 2",
					Labels:      map[string]string{},
				},
				{
					Id:          "namespace3",
					Name:        "Namespace 3",
					ClusterName: "Cluster 1",
					Labels:      map[string]string{"Key": "Value"},
				},
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			out := Namespaces(tc.input)
			require.Equal(it, len(tc.input), len(out))

			if tc.input == nil {
				assert.Nil(it, out)
			} else {
				for ix := range tc.input {
					inNS := tc.input[ix]
					outNS := out[ix]
					assert.Equal(it, inNS.GetId(), outNS.GetId())
					assert.Equal(it, inNS.GetName(), outNS.GetName())
					assert.Equal(it, inNS.GetClusterName(), outNS.GetClusterName())
					assert.Equal(it, inNS.GetLabels(), outNS.GetLabels())
				}
			}
		})
	}
}
