package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
)

/*
 * This represents a list of common resolvers between vulnerability GraphQL types.
 * It should be kept in sync with the interface definition below.
 *
 * NOTE: This list is and should remain alphabetically ordered
 */
var commonVulnerabilitySubResolvers = []string{
	"createdAt: Time",
	"cve: String!",
	"cveBaseInfo: CVEInfo",
	"cvss: Float!",
	"envImpact: Float!",
	"fixedByVersion: String!",
	"id: ID!",
	"impactScore: Float!",
	"isFixable(query: String): Boolean!",
	"lastModified: Time",
	"lastScanned: Time",
	"link: String!",
	"publishedOn: Time",
	"scoreVersion: String!",
	"severity: String!",
	"summary: String!",
	"suppressActivation: Time",
	"suppressExpiry: Time",
	"suppressed: Boolean!",
	"unusedVarSink(query: String): Int",
	"vectors: EmbeddedVulnerabilityVectors",
}

// CommonVulnerabilityResolver represents the supported API on all vulnerabilities
//
//	NOTE: This list is and should remain alphabetically ordered
type CommonVulnerabilityResolver interface {
	CreatedAt(ctx context.Context) (*graphql.Time, error)
	CVE(ctx context.Context) string
	CveBaseInfo(ctx context.Context) (*cVEInfoResolver, error)
	Cvss(ctx context.Context) float64
	EnvImpact(ctx context.Context) (float64, error)
	FixedByVersion(ctx context.Context) (string, error)
	Id(ctx context.Context) graphql.ID
	ImpactScore(ctx context.Context) float64
	IsFixable(ctx context.Context, args RawQuery) (bool, error)
	LastModified(ctx context.Context) (*graphql.Time, error)
	LastScanned(ctx context.Context) (*graphql.Time, error)
	Link(ctx context.Context) string
	PublishedOn(ctx context.Context) (*graphql.Time, error)
	ScoreVersion(ctx context.Context) string
	Severity(ctx context.Context) string
	Summary(ctx context.Context) string
	SuppressActivation(ctx context.Context) (*graphql.Time, error)
	SuppressExpiry(ctx context.Context) (*graphql.Time, error)
	Suppressed(ctx context.Context) bool
	UnusedVarSink(ctx context.Context, args RawQuery) *int32
	Vectors() *EmbeddedVulnerabilityVectorsResolver
}
