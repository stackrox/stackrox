package resolvers

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search/predicate"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	componentPredicateFactory = predicate.NewFactory("component", &storage.EmbeddedImageScanComponent{})
)

func init() {
	schema := getBuilder()
	utils.Must(
		// NOTE: This list is and should remain alphabetically ordered
		schema.AddType("EmbeddedImageScanComponent", []string{
			"activeState(query: String): ActiveState",
			"deploymentCount(query: String, scopeQuery: String): Int!",
			"deployments(query: String, scopeQuery: String, pagination: Pagination): [Deployment!]!",
			"fixedIn: String!",
			"id: ID!",
			"imageCount(query: String, scopeQuery: String): Int!",
			"images(query: String, scopeQuery: String, pagination: Pagination): [Image!]!",
			"lastScanned: Time",
			"layerIndex: Int",
			"license: License",
			"location(query: String): String!",
			"name: String!",
			"nodeCount(query: String, scopeQuery: String): Int!",
			"nodes(query: String, scopeQuery: String, pagination: Pagination): [Node!]!",
			"priority: Int!",
			"riskScore: Float!",
			"source: String!",
			"topVuln: EmbeddedVulnerability",
			"version: String!",
			"vulnCount(query: String, scopeQuery: String): Int!",
			"vulnCounter(query: String): VulnerabilityCounter!",
			"vulns(query: String, scopeQuery: String, pagination: Pagination): [EmbeddedVulnerability]!",
		}),
		schema.AddExtraResolver("EmbeddedImageScanComponent", `unusedVarSink(query: String): Int`),
	)
}
