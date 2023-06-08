package resolvers

import (
	"context"

	protoTypes "github.com/gogo/protobuf/types"
	"github.com/graph-gophers/graphql-go"
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
		schema.AddExtraResolver("EmbeddedNodeScanComponent", `unusedVarSink(query: String): Int`),
	)
}

func (resolver *nodeScanResolver) Components(_ context.Context, args PaginatedQuery) ([]*EmbeddedNodeScanComponentResolver, error) {
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

	return paginate(pagination, vulns, err)
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
	os          string
	root        *Resolver
	lastScanned *protoTypes.Timestamp
	data        *storage.EmbeddedNodeScanComponent
}

// UnusedVarSink represents a query sink
func (encr *EmbeddedNodeScanComponentResolver) UnusedVarSink(_ context.Context, _ RawQuery) *int32 {
	return nil
}

// ID returns a unique identifier for the component.
func (encr *EmbeddedNodeScanComponentResolver) ID(_ context.Context) graphql.ID {
	return graphql.ID(scancomponent.ComponentID(encr.data.GetName(), encr.data.GetVersion(), encr.os))
}

// Name returns the name of the component.
func (encr *EmbeddedNodeScanComponentResolver) Name(_ context.Context) string {
	return encr.data.GetName()
}

// Version gives the version of the node component.
func (encr *EmbeddedNodeScanComponentResolver) Version(_ context.Context) string {
	return encr.data.GetVersion()
}

// Priority returns the priority of the component.
func (encr *EmbeddedNodeScanComponentResolver) Priority(_ context.Context) int32 {
	return int32(encr.data.GetPriority())
}

// RiskScore returns the risk score of the component.
func (encr *EmbeddedNodeScanComponentResolver) RiskScore(_ context.Context) float64 {
	return float64(encr.data.GetRiskScore())
}

// LastScanned is the last time the component was scanned in an node.
func (encr *EmbeddedNodeScanComponentResolver) LastScanned(_ context.Context) (*graphql.Time, error) {
	return timestamp(encr.lastScanned)
}

// TopVuln returns the first vulnerability with the top CVSS score.
func (encr *EmbeddedNodeScanComponentResolver) TopVuln(_ context.Context) (*EmbeddedVulnerabilityResolver, error) {
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
func (encr *EmbeddedNodeScanComponentResolver) Vulns(_ context.Context, args PaginatedQuery) ([]*EmbeddedVulnerabilityResolver, error) {
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

	vulns, err = paginate(query.GetPagination(), vulns, nil)
	if err != nil {
		return nil, err
	}
	return vulns, nil
}

// VulnCount resolves the number of vulnerabilities contained in the node component.
func (encr *EmbeddedNodeScanComponentResolver) VulnCount(_ context.Context, _ RawQuery) (int32, error) {
	return int32(len(encr.data.GetVulns())), nil
}

// VulnCounter resolves the number of different types of vulnerabilities contained in an node component.
func (encr *EmbeddedNodeScanComponentResolver) VulnCounter(_ context.Context, _ RawQuery) (*VulnerabilityCounterResolver, error) {
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
			thisComponentID := scancomponent.ComponentID(component.GetName(), component.GetVersion(), node.GetScan().GetOperatingSystem())
			if _, exists := idToComponent[thisComponentID]; !exists {
				idToComponent[thisComponentID] = &EmbeddedNodeScanComponentResolver{
					os:   node.GetScan().GetOperatingSystem(),
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
