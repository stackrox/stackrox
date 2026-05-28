package resolvers

import (
	"context"
	"time"

	"github.com/graph-gophers/graphql-go"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/scandata/types"
	"github.com/stackrox/rox/generated/storage"
	pkgMetrics "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/utils"
)

func init() {
	schema := getBuilder()
	utils.Must(
		// ProtoCVEListItem represents one row in the CVE list page
		schema.AddType("ProtoCVEListItem", []string{
			"cveName: String!",
			"severity: String!",
			"cvss: Float!",
			"imageCount: Int!",
			"fixable: Boolean!",
			"firstSeen: Time",
		}),
		// ProtoAdvisory represents an advisory (scanner-specific CVE finding)
		schema.AddType("ProtoAdvisory", []string{
			"id: String!",
			"advisoryId: String!",
			"cveName: String!",
			"severity: String!",
			"cvss: Float",
			"source: String!",
			"fixable: Boolean!",
			"fixedBy: String!",
			"description: String!",
			"publishedDate: Time",
		}),
		// Query resolvers
		schema.AddQuery("protoCVEList(limit: Int, offset: Int): [ProtoCVEListItem!]!"),
		schema.AddQuery("protoCVEDetail(cveName: String!): [ProtoAdvisory!]!"),
	)
}

// ProtoCVEListItem wraps a CVEListRow for GraphQL
type protoCVEListItemResolver struct {
	data *types.CVEListRow
}

func (r *protoCVEListItemResolver) CveName() string {
	return r.data.CVEName
}

func (r *protoCVEListItemResolver) Severity() string {
	return severityToString(r.data.Severity)
}

func (r *protoCVEListItemResolver) Cvss() float64 {
	return float64(r.data.CVSS)
}

func (r *protoCVEListItemResolver) ImageCount() int32 {
	return int32(r.data.ImageCount)
}

func (r *protoCVEListItemResolver) Fixable() bool {
	return r.data.Fixable
}

func (r *protoCVEListItemResolver) FirstSeen() *graphql.Time {
	if r.data.FirstSeen == nil {
		return nil
	}
	return &graphql.Time{Time: *r.data.FirstSeen}
}

// ProtoAdvisory wraps a ScanFinding for the advisory view
type protoAdvisoryResolver struct {
	data *storage.ScanFinding
}

func (r *protoAdvisoryResolver) ID() graphql.ID {
	return graphql.ID(r.data.GetId())
}

func (r *protoAdvisoryResolver) AdvisoryId() string {
	return r.data.GetAdvisoryId()
}

func (r *protoAdvisoryResolver) CveName() string {
	return r.data.GetCveName()
}

func (r *protoAdvisoryResolver) Severity() string {
	return r.data.GetSeverity().String()
}

func (r *protoAdvisoryResolver) Cvss() *float64 {
	cvss := float64(r.data.GetCvss())
	if cvss == 0 {
		return nil
	}
	return &cvss
}

func (r *protoAdvisoryResolver) Source() string {
	return r.data.GetSourceName()
}

func (r *protoAdvisoryResolver) Fixable() bool {
	return r.data.GetIsFixable()
}

func (r *protoAdvisoryResolver) FixedBy() string {
	return r.data.GetFixedBy()
}

func (r *protoAdvisoryResolver) Description() string {
	return r.data.GetDescription()
}

func (r *protoAdvisoryResolver) PublishedDate() *graphql.Time {
	ts := r.data.GetPublishedDate()
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &graphql.Time{Time: t}
}

// ProtoCVEList returns the CVE list page data
func (resolver *Resolver) ProtoCVEList(ctx context.Context, args struct {
	Limit  *int32
	Offset *int32
}) ([]*protoCVEListItemResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ProtoCVEList")

	// For prototype, skip SAC checks
	limit := 100
	if args.Limit != nil {
		limit = int(*args.Limit)
	}
	offset := 0
	if args.Offset != nil {
		offset = int(*args.Offset)
	}

	rows, _, err := resolver.ScanDataStore.ListCVEs(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	resolvers := make([]*protoCVEListItemResolver, 0, len(rows))
	for _, row := range rows {
		resolvers = append(resolvers, &protoCVEListItemResolver{data: row})
	}
	return resolvers, nil
}

// ProtoCVEDetail returns all advisories for a specific CVE
func (resolver *Resolver) ProtoCVEDetail(ctx context.Context, args struct{ CveName string }) ([]*protoAdvisoryResolver, error) {
	defer metrics.SetGraphQLOperationDurationTime(time.Now(), pkgMetrics.Root, "ProtoCVEDetail")

	// For prototype, skip SAC checks
	findings, err := resolver.ScanDataStore.GetFindingsByCVE(ctx, args.CveName)
	if err != nil {
		return nil, err
	}

	resolvers := make([]*protoAdvisoryResolver, 0, len(findings))
	for _, finding := range findings {
		resolvers = append(resolvers, &protoAdvisoryResolver{data: finding})
	}
	return resolvers, nil
}

// severityToString converts severity int to string (simplified for prototype)
func severityToString(severity int32) string {
	switch storage.VulnerabilitySeverity(severity) {
	case storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY:
		return "LOW"
	case storage.VulnerabilitySeverity_MODERATE_VULNERABILITY_SEVERITY:
		return "MODERATE"
	case storage.VulnerabilitySeverity_IMPORTANT_VULNERABILITY_SEVERITY:
		return "IMPORTANT"
	case storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}
