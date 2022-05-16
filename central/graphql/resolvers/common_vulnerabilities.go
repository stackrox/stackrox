package resolvers

import (
	"context"

	"github.com/graph-gophers/graphql-go"
)

var commonVulnerabilitySubResolvers = []string{ // note: alphabetically ordered
	"activeState(query: String): ActiveState",
	"componentCount(query: String): Int!",
	"components(query: String, pagination: Pagination): [EmbeddedImageScanComponent!]!",
	"createdAt: Time", // Discovered At System
	"cve: String!",
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
	"vulnerabilityState: String!",
}

// CommonVulnerabilityResolver represents the supported API on image vulnerabilities
type CommonVulnerabilityResolver interface { // note: alphabetically ordered
	ActiveState(ctx context.Context, args RawQuery) (*activeStateResolver, error)
	ComponentCount(ctx context.Context, args RawQuery) (int32, error)
	Components(ctx context.Context, args PaginatedQuery) ([]ComponentResolver, error)
	CreatedAt(ctx context.Context) (*graphql.Time, error)
	CVE(ctx context.Context) string
	Cvss(ctx context.Context) float64
	EnvImpact(ctx context.Context) (float64, error)
	FixedByVersion(ctx context.Context) (string, error)
	ID(ctx context.Context) graphql.ID
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
	VulnerabilityState(ctx context.Context) string
}
