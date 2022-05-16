package resolvers

import "github.com/stackrox/rox/pkg/utils"

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("EmbeddedNodeScanComponent", []string{
			"license: License",
			"id: ID!",
			"name: String!",
			"version: String!",
			"topVuln: EmbeddedVulnerability",
			"vulns(query: String, scopeQuery: String, pagination: Pagination): [EmbeddedVulnerability]!",
			"vulnCount(query: String, scopeQuery: String): Int!",
			"vulnCounter(query: String): VulnerabilityCounter!",
			"plottedVulns(query: String): PlottedVulnerabilities!",
			"lastScanned: Time",
			"nodes(query: String, scopeQuery: String, pagination: Pagination): [Node!]!",
			"nodeCount(query: String, scopeQuery: String): Int!",
			"priority: Int!",
			"source: String!",
			"location(query: String): String!",
			"riskScore: Float!",
			"fixedIn: String!",
			"unusedVarSink(query: String): Int",
		}),
		schema.AddQuery("component(id: ID): NodeScanComponent"),
		schema.AddQuery("components(query: String, scopeQuery: String, pagination: Pagination): [NodeScanComponent!]!"),
		schema.AddQuery("componentCount(query: String): Int!"),
	)
}
