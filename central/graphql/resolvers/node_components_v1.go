package resolvers

import (
	"context"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/node/mappings"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stackrox/rox/pkg/search"
	utils "github.com/stackrox/rox/pkg/utils"
)

// Resolvers on Embedded Scan Object.
/////////////////////////////////////

func init() {
	schema := getBuilder()
	utils.Must(
		schema.AddType("EmbeddedNodeScanComponent", []string{
			"id: ID!",
			"name: String!",
			"version: String!",
			"topVuln: EmbeddedVulnerability",
			"vulns(query: String, pagination: Pagination): [EmbeddedVulnerability]!",
			"vulnCount(query: String): Int!",
			"vulnCounter(query: String): VulnerabilityCounter!",
			"lastScanned: Time",
			"priority: Int!",
			"riskScore: Float!",
		}),
		schema.AddExtraResolver("NodeScan", `components(query: String, pagination: Pagination): [EmbeddedNodeScanComponent!]!`),
		schema.AddExtraResolver("NodeScan", `componentCount(query: String): Int!`),
		schema.AddExtraResolver("EmbeddedNodeScanComponent", `unusedVarSink(query: String): Int`),
		schema.AddExtraResolver("EmbeddedNodeScanComponent", "plottedVulns(query: String): PlottedVulnerabilities!"),
	)
}

func (resolver *nodeScanResolver) Components(ctx context.Context, args PaginatedQuery) ([]*EmbeddedNodeScanComponentResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	pagination := query.GetPagination()
	query.Pagination = nil

	vulns, err := mapNodesToComponentResolvers(resolver.root, []*storage.Node{
		{
			Scan: resolver.data,
		},
	}, query)

	resolvers, err := paginationWrapper{
		pv: pagination,
	}.paginate(vulns, err)
	return resolvers.([]*EmbeddedNodeScanComponentResolver), err
}

func (resolver *nodeScanResolver) ComponentCount(ctx context.Context, args RawQuery) (int32, error) {
	resolvers, err := resolver.Components(ctx, PaginatedQuery{Query: args.Query})
	if err != nil {
		return 0, err
	}
	return int32(len(resolvers)), nil
}

// EmbeddedNodeScanComponentResolver resolves data about an node scan component.
type EmbeddedNodeScanComponentResolver struct {
	root        *Resolver
	lastScanned *protoTypes.Timestamp
	data        *storage.EmbeddedNodeScanComponent
}

// PlottedVulns returns the data required by top risky component scatter-plot on vuln mgmt dashboard
func (encr *EmbeddedNodeScanComponentResolver) PlottedVulns(ctx context.Context, args RawQuery) (*PlottedVulnerabilitiesResolver, error) {
	return nil, errors.New("not implemented")
}

// UnusedVarSink represents a query sink
func (encr *EmbeddedNodeScanComponentResolver) UnusedVarSink(ctx context.Context, args RawQuery) *int32 {
	return nil
}

// ID returns a unique identifier for the component.
func (encr *EmbeddedNodeScanComponentResolver) ID(ctx context.Context) graphql.ID {
	return graphql.ID(scancomponent.ComponentID(encr.data.GetName(), encr.data.GetVersion(), ""))
}

// Name returns the name of the component.
func (encr *EmbeddedNodeScanComponentResolver) Name(ctx context.Context) string {
	return encr.data.GetName()
}

// Version gives the version of the node component.
func (encr *EmbeddedNodeScanComponentResolver) Version(ctx context.Context) string {
	return encr.data.GetVersion()
}

// Priority returns the priority of the component.
func (encr *EmbeddedNodeScanComponentResolver) Priority(ctx context.Context) int32 {
	return int32(encr.data.GetPriority())
}

// RiskScore returns the risk score of the component.
func (encr *EmbeddedNodeScanComponentResolver) RiskScore(ctx context.Context) float64 {
	return float64(encr.data.GetRiskScore())
}

// LastScanned is the last time the component was scanned in an node.
func (encr *EmbeddedNodeScanComponentResolver) LastScanned(ctx context.Context) (*graphql.Time, error) {
	return timestamp(encr.lastScanned)
}

// TopVuln returns the first vulnerability with the top CVSS score.
func (encr *EmbeddedNodeScanComponentResolver) TopVuln(ctx context.Context) (VulnerabilityResolver, error) {
	var maxCvss *storage.EmbeddedVulnerability
	for _, vuln := range encr.data.GetVulns() {
		if maxCvss == nil || vuln.GetCvss() > maxCvss.GetCvss() {
			maxCvss = vuln
		}
	}
	if maxCvss == nil {
		return nil, nil
	}
	return encr.root.wrapEmbeddedVulnerability(maxCvss, nil)
}

// Vulns resolves the vulnerabilities contained in the node component.
func (encr *EmbeddedNodeScanComponentResolver) Vulns(ctx context.Context, args PaginatedQuery) ([]VulnerabilityResolver, error) {
	query, err := args.AsV1QueryOrEmpty()
	if err != nil {
		return nil, err
	}

	vulnQuery, _ := search.FilterQueryWithMap(query, mappings.VulnerabilityOptionsMap)
	vulnPred, err := vulnPredicateFactory.GeneratePredicate(vulnQuery)
	if err != nil {
		return nil, err
	}

	// Use the nodes to map CVEs to the nodes and components.
	vulns := make([]*EmbeddedVulnerabilityResolver, 0, len(encr.data.GetVulns()))
	for _, vuln := range encr.data.GetVulns() {
		if !vulnPred.Matches(vuln) {
			continue
		}
		vulns = append(vulns, &EmbeddedVulnerabilityResolver{
			data:        vuln,
			root:        encr.root,
			lastScanned: encr.lastScanned,
		})
	}

	resolvers, err := paginationWrapper{
		pv: query.GetPagination(),
	}.paginate(vulns, nil)
	if err != nil {
		return nil, err
	}
	paginatedVulns := resolvers.([]*EmbeddedVulnerabilityResolver)

	ret := make([]VulnerabilityResolver, 0, len(paginatedVulns))
	for _, resolver := range paginatedVulns {
		ret = append(ret, resolver)
	}
	return ret, err
}

// VulnCount resolves the number of vulnerabilities contained in the node component.
func (encr *EmbeddedNodeScanComponentResolver) VulnCount(ctx context.Context, args RawQuery) (int32, error) {
	return int32(len(encr.data.GetVulns())), nil
}

// VulnCounter resolves the number of different types of vulnerabilities contained in an node component.
func (encr *EmbeddedNodeScanComponentResolver) VulnCounter(ctx context.Context, args RawQuery) (*VulnerabilityCounterResolver, error) {
	return mapVulnsToVulnerabilityCounter(encr.data.GetVulns()), nil
}

// Static helpers.
//////////////////

// Map the nodes that matched a query to the node components it contains.
func mapNodesToComponentResolvers(root *Resolver, nodes []*storage.Node, query *v1.Query) ([]*EmbeddedNodeScanComponentResolver, error) {
	query, _ = search.FilterQueryWithMap(query, mappings.ComponentOptionsMap)
	componentPred, err := componentPredicateFactory.GeneratePredicate(query)
	if err != nil {
		return nil, err
	}

	// Use the nodes to map CVEs to the nodes and components.
	idToComponent := make(map[string]*EmbeddedNodeScanComponentResolver)
	for _, node := range nodes {
		for _, component := range node.GetScan().GetComponents() {
			if !componentPred.Matches(component) {
				continue
			}
			thisComponentID := scancomponent.ComponentID(component.GetName(), component.GetVersion(), "")
			if _, exists := idToComponent[thisComponentID]; !exists {
				idToComponent[thisComponentID] = &EmbeddedNodeScanComponentResolver{
					root: root,
					data: component,
				}
			}
			latestTime := idToComponent[thisComponentID].lastScanned
			if latestTime == nil || node.GetScan().GetScanTime().Compare(latestTime) > 0 {
				idToComponent[thisComponentID].lastScanned = node.GetScan().GetScanTime()
			}
		}
	}

	// Create the resolvers.
	resolvers := make([]*EmbeddedNodeScanComponentResolver, 0, len(idToComponent))
	for _, component := range idToComponent {
		resolvers = append(resolvers, component)
	}
	return resolvers, nil
}
