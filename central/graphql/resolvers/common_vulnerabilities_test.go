package resolvers

import (
	"context"
	"fmt"
	"testing"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type vulnerabilityScoreTestResolver struct {
	image   *imageCVEV2Resolver
	node    *nodeCVEResolver
	cluster *clusterCVEResolver
}

func (r *vulnerabilityScoreTestResolver) Image() *imageCVEV2Resolver { return r.image }
func (r *vulnerabilityScoreTestResolver) Node() *nodeCVEResolver     { return r.node }
func (r *vulnerabilityScoreTestResolver) Cluster() *clusterCVEResolver {
	return r.cluster
}

func TestVulnerabilityScoresGraphQL(t *testing.T) {
	cases := map[string]struct {
		score float32
		want  string
	}{
		"unavailable": {want: "null"},
		"available":   {score: 7.5, want: "7.5"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			root := &vulnerabilityScoreTestResolver{
				image:   &imageCVEV2Resolver{data: &storage.ImageCVEV2{Cvss: tc.score, Nvdcvss: tc.score}},
				node:    &nodeCVEResolver{data: &storage.NodeCVE{Cvss: tc.score}},
				cluster: &clusterCVEResolver{data: &storage.ClusterCVE{Cvss: tc.score}},
			}
			schema, err := graphql.ParseSchema(`
				type Query { image: ImageCVEV2!, node: NodeCVE!, cluster: ClusterCVE! }
				type ImageCVEV2 { cvss: Float, nvdcvss: Float }
				type NodeCVE { cvss: Float }
				type ClusterCVE { cvss: Float }
			`, root)
			require.NoError(t, err)
			response := schema.Exec(context.Background(), `{ image { cvss nvdcvss } node { cvss } cluster { cvss } }`, "", nil)
			require.Empty(t, response.Errors)
			assert.JSONEq(t, fmt.Sprintf(`{"image":{"cvss":%s,"nvdcvss":%s},"node":{"cvss":%s},"cluster":{"cvss":%s}}`,
				tc.want, tc.want, tc.want, tc.want), string(response.Data))
		})
	}
}
