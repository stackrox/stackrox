package storagetoeffectiveaccessscope

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClusters(t *testing.T) {
	for name, tc := range map[string]struct {
		input []*storage.Cluster
	}{
		"nil input": {
			input: nil,
		},
		"empty input": {
			input: []*storage.Cluster{},
		},
		"single cluster": {
			input: []*storage.Cluster{
				{Id: "cluster1", Name: "cluster1"},
			},
		},
		"multiple clusters": {
			input: []*storage.Cluster{
				{Id: "cluster1", Name: "Cluster 1"},
				{Id: "cluster2", Name: "Cluster 2", Labels: map[string]string{}},
				{Id: "cluster3", Name: "Cluster 3", Labels: map[string]string{"Key": "Value"}},
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			out := Clusters(tc.input)
			require.Equal(it, len(tc.input), len(out))

			if tc.input == nil {
				assert.Nil(it, out)
			} else {
				for ix := range tc.input {
					inc := tc.input[ix]
					outc := out[ix]
					assert.Equal(it, inc.GetId(), outc.GetId())
					assert.Equal(it, inc.GetName(), outc.GetName())
					assert.Equal(it, inc.GetLabels(), outc.GetLabels())
				}
			}
		})
	}
}
