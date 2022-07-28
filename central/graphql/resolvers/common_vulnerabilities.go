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
	"cveBaseInfo: CVEInfo",
	"cvss: Float!",
	"envImpact: Float!",
	"fixedByVersion: String!",
	"id: ID!",
	"impactScore: Float!",
	"isFixable(query: String): Boolean!",
	"lastScanned: Time",
	"severity: String!",
	"snoozeExpiry: Time",
	"snoozeStart: Time",
	"snoozed: Boolean!",
	"unusedVarSink(query: String): Int",
	"vectors: EmbeddedVulnerabilityVectors",

	// deprecated fields
	"suppressActivation: Time @deprecated(reason: \"use 'snoozeStart'\")",
	"suppressExpiry: Time @deprecated(reason: \"use 'snoozeExpiry'\")",
	"suppressed: Boolean! @deprecated(reason: \"use 'snoozed'\")",

	// CVEInfo fields
	"createdAt: Time @deprecated(reason: \"use 'cveBaseInfo'\")",
	"cve: String! @deprecated(reason: \"use 'cveBaseInfo'\")",
	"lastModified: Time @deprecated(reason: \"use 'cveBaseInfo'\")",
	"link: String! @deprecated(reason: \"use 'cveBaseInfo'\")",
	"publishedOn: Time @deprecated(reason: \"use 'cveBaseInfo'\")",
	"scoreVersion: String! @deprecated(reason: \"use 'cveBaseInfo'\")",
	"summary: String! @deprecated(reason: \"use 'cveBaseInfo'\")",
}

// CommonVulnerabilityResolver represents the supported API on all vulnerabilities
//  NOTE: This list is and should remain alphabetically ordered
type CommonVulnerabilityResolver interface {
	CveBaseInfo(ctx context.Context) (*cVEInfoResolver, error)
	Cvss(ctx context.Context) float64
	EnvImpact(ctx context.Context) (float64, error)
	FixedByVersion(ctx context.Context) (string, error)
	Id(ctx context.Context) graphql.ID
	ImpactScore(ctx context.Context) float64
	IsFixable(ctx context.Context, args RawQuery) (bool, error)
	LastScanned(ctx context.Context) (*graphql.Time, error)
	Severity(ctx context.Context) string
	SnoozeStart(ctx context.Context) (*graphql.Time, error)
	SnoozeExpiry(ctx context.Context) (*graphql.Time, error)
	Snoozed(ctx context.Context) bool
	UnusedVarSink(ctx context.Context, args RawQuery) *int32
	Vectors() *EmbeddedVulnerabilityVectorsResolver

	// deprecated functions

	ID(ctx context.Context) graphql.ID
	SuppressActivation(ctx context.Context) (*graphql.Time, error)
	SuppressExpiry(ctx context.Context) (*graphql.Time, error)
	Suppressed(ctx context.Context) bool

	// CVEInfo functions

	CreatedAt(ctx context.Context) (*graphql.Time, error)
	CVE(ctx context.Context) string
	LastModified(ctx context.Context) (*graphql.Time, error)
	Link(ctx context.Context) string
	PublishedOn(ctx context.Context) (*graphql.Time, error)
	ScoreVersion(ctx context.Context) string
	Summary(ctx context.Context) string
}
