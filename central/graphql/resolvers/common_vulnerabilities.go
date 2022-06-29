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
	"createdAt: Time", // Discovered At System
	"cve: String!",
	"envImpact: Float!",
	"fixedByVersion: String!",
	"isFixable(query: String): Boolean!",
	"lastModified: Time",
	"lastScanned: Time",
	"link: String!",
	"publishedOn: Time",
	"scoreVersion: String!",
	"summary: String!",
	"suppressActivation: Time",
	"suppressExpiry: Time",
	"suppressed: Boolean!",
	"unusedVarSink(query: String): Int",
	"vectors: EmbeddedVulnerabilityVectors",
	"vulnerabilityState: String!",
}

// CommonVulnerabilityResolver represents the supported API on all vulnerabilities
//  NOTE: This list is and should remain alphabetically ordered
type CommonVulnerabilityResolver interface {

	/*
	 * The following functions are auto generated based off of defined proto objects
	 */
	CveBaseInfo(ctx context.Context) (*cVEInfoResolver, error)
	Cvss(ctx context.Context) float64
	Id(ctx context.Context) graphql.ID
	ImpactScore(ctx context.Context) float64
	OperatingSystem(ctx context.Context) string
	Severity(ctx context.Context) string
	SnoozeStart(ctx context.Context) (*graphql.Time, error)
	SnoozeExpiry(ctx context.Context) (*graphql.Time, error)
	Snoozed(ctx context.Context) bool

	/*
	 * The following functions are to allow UI time to transition to new definitions
	 */
	CreatedAt(ctx context.Context) (*graphql.Time, error)
	CVE(ctx context.Context) string
	LastModified(ctx context.Context) (*graphql.Time, error)
	Link(ctx context.Context) string
	PublishedOn(ctx context.Context) (*graphql.Time, error)
	ScoreVersion(ctx context.Context) string
	Summary(ctx context.Context) string
	SuppressActivation(ctx context.Context) (*graphql.Time, error)
	SuppressExpiry(ctx context.Context) (*graphql.Time, error)
	Suppressed(ctx context.Context) bool

	EnvImpact(ctx context.Context) (float64, error)
	FixedByVersion(ctx context.Context) (string, error)
	IsFixable(ctx context.Context, args RawQuery) (bool, error)
	LastScanned(ctx context.Context) (*graphql.Time, error)
	UnusedVarSink(ctx context.Context, args RawQuery) *int32
	Vectors() *EmbeddedVulnerabilityVectorsResolver
	VulnerabilityState(ctx context.Context) string
}
