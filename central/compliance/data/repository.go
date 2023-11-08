package data

import (
	"bytes"
	"compress/gzip"
	"context"
	"math"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/mailru/easyjson"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/compliance/framework"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/compliance"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/compliance/compress"
	"github.com/stackrox/rox/pkg/compliance/data"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/postgres/pgutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

type repository struct {
	cluster     *storage.Cluster
	nodes       map[string]*storage.Node
	deployments map[string]*storage.Deployment

	unresolvedAlerts             []*storage.ListAlert
	networkPolicies              map[string]*storage.NetworkPolicy
	deploymentsToNetworkPolicies map[string][]*storage.NetworkPolicy
	policies                     map[string]*storage.Policy
	images                       []*storage.ListImage
	imageIntegrations            []*storage.ImageIntegration
	registries                   []framework.ImageMatcher
	scanners                     []framework.ImageMatcher
	sshProcessIndicators         []*storage.ProcessIndicator
	hasProcessIndicators         bool
	networkFlows                 []*storage.NetworkFlow
	notifiers                    []*storage.Notifier
	roles                        []*storage.K8SRole
	bindings                     []*storage.K8SRoleBinding
	cisDockerRunCheck            bool
	cisKubernetesRunCheck        bool
	categoryToPolicies           map[string]set.StringSet // maps categories to policy set

	complianceOperatorResults map[string][]*storage.ComplianceOperatorCheckResult

	hostScrape map[string]*compliance.ComplianceReturn

	nodeResults map[string]map[string]*compliance.ComplianceStandardResult
}

func (r *repository) Cluster() *storage.Cluster {
	return r.cluster
}

func (r *repository) Nodes() map[string]*storage.Node {
	return r.nodes
}

func (r *repository) Deployments() map[string]*storage.Deployment {
	return r.deployments
}

func (r *repository) NetworkPolicies() map[string]*storage.NetworkPolicy {
	return r.networkPolicies
}

func (r *repository) DeploymentsToNetworkPolicies() map[string][]*storage.NetworkPolicy {
	return r.deploymentsToNetworkPolicies
}

func (r *repository) Policies() map[string]*storage.Policy {
	return r.policies
}

func (r *repository) PolicyCategories() map[string]set.StringSet {
	return r.categoryToPolicies
}

func (r *repository) Images() []*storage.ListImage {
	return r.images
}

func (r *repository) ImageIntegrations() []*storage.ImageIntegration {
	return r.imageIntegrations
}

func (r *repository) SSHProcessIndicators() []*storage.ProcessIndicator {
	return r.sshProcessIndicators
}

func (r *repository) HasProcessIndicators() bool {
	return r.hasProcessIndicators
}

func (r *repository) NetworkFlows() []*storage.NetworkFlow {
	return r.networkFlows
}

func (r *repository) Notifiers() []*storage.Notifier {
	return r.notifiers
}

func (r *repository) K8sRoles() []*storage.K8SRole {
	return r.roles
}

func (r *repository) K8sRoleBindings() []*storage.K8SRoleBinding {
	return r.bindings
}

func (r *repository) UnresolvedAlerts() []*storage.ListAlert {
	return r.unresolvedAlerts
}

func (r *repository) HostScraped(node *storage.Node) *compliance.ComplianceReturn {
	return r.hostScrape[node.GetName()]
}

func (r *repository) NodeResults() map[string]map[string]*compliance.ComplianceStandardResult {
	return r.nodeResults
}

func (r *repository) CISKubernetesTriggered() bool {
	return r.cisKubernetesRunCheck
}

func (r *repository) RegistryIntegrations() []framework.ImageMatcher {
	return r.registries
}

func (r *repository) ScannerIntegrations() []framework.ImageMatcher {
	return r.scanners
}

func (r *repository) ComplianceOperatorResults() map[string][]*storage.ComplianceOperatorCheckResult {
	return r.complianceOperatorResults
}

func newRepository(ctx context.Context, domain framework.ComplianceDomain, scrapeResults map[string]*compliance.ComplianceReturn, factory *factory) (*repository, error) {
	r := &repository{}
	if err := r.init(ctx, domain, scrapeResults, factory); err != nil {
		return nil, err
	}
	return r, nil
}

func nodesByID(nodes []*storage.Node) map[string]*storage.Node {
	result := make(map[string]*storage.Node, len(nodes))
	for _, node := range nodes {
		result[node.GetId()] = node
	}
	return result
}

func deploymentsByID(deployments []*storage.Deployment) map[string]*storage.Deployment {
	result := make(map[string]*storage.Deployment, len(deployments))
	for _, deployment := range deployments {
		result[deployment.GetId()] = deployment
	}
	return result
}

func networkPoliciesByID(policies []*storage.NetworkPolicy) map[string]*storage.NetworkPolicy {
	result := make(map[string]*storage.NetworkPolicy, len(policies))
	for _, policy := range policies {
		result[policy.GetId()] = policy
	}
	return result
}

func policiesByName(policies []*storage.Policy) map[string]*storage.Policy {
	result := make(map[string]*storage.Policy, len(policies))
	for _, policy := range policies {
		result[policy.GetName()] = policy
	}
	return result
}

func policyCategories(policies []*storage.Policy) map[string]set.StringSet {
	result := make(map[string]set.StringSet, len(policies))
	for _, policy := range policies {
		if policy.Disabled {
			continue
		}
		for _, category := range policy.Categories {
			policySet, ok := result[category]
			if !ok {
				policySet = set.NewStringSet()
			}
			policySet.Add(policy.Name)
			result[category] = policySet
		}
	}
	return result
}

func (r *repository) init(ctx context.Context, domain framework.ComplianceDomain, scrapeResults map[string]*compliance.ComplianceReturn, f *factory) error {
	r.cluster = domain.Cluster().Cluster()
	r.nodes = nodesByID(framework.Nodes(domain))

	deployments := framework.Deployments(domain)
	r.deployments = deploymentsByID(deployments)

	clusterID := r.cluster.GetId()

	clusterQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	infPagination := &v1.QueryPagination{
		Limit: math.MaxInt32,
	}
	clusterQuery.Pagination = infPagination

	networkPolicies, err := f.networkPoliciesStore.GetNetworkPolicies(ctx, clusterID, "")
	if err != nil {
		return err
	}
	r.networkPolicies = networkPoliciesByID(networkPolicies)

	networkTree := f.netTreeMgr.GetNetworkTree(ctx, clusterID)
	if networkTree == nil {
		networkTree = f.netTreeMgr.CreateNetworkTree(ctx, clusterID)
	}

	r.deploymentsToNetworkPolicies = f.networkGraphEvaluator.GetApplyingPoliciesPerDeployment(deployments, networkTree, networkPolicies)

	policies, err := f.policyStore.GetAllPolicies(ctx)
	if err != nil {
		return err
	}

	r.policies = policiesByName(policies)
	r.categoryToPolicies = policyCategories(policies)

	r.images, err = f.imageStore.SearchListImages(ctx, clusterQuery)
	if err != nil {
		return err
	}

	r.imageIntegrations, err = f.imageIntegrationStore.GetImageIntegrations(ctx,
		&v1.GetImageIntegrationsRequest{},
	)
	if err != nil {
		return err
	}

	for _, registryIntegration := range f.imageIntegrationsSet.RegistrySet().GetAll() {
		r.registries = append(r.registries, registryIntegration)
	}
	for _, scannerIntegration := range f.imageIntegrationsSet.ScannerSet().GetAll() {
		r.scanners = append(r.scanners, scannerIntegration.GetScanner())
	}

	sshProcessQuery := search.NewQueryBuilder().AddRegexes(search.ProcessExecPath, ".*ssh.*").ProtoQuery()
	sshQuery := search.ConjunctionQuery(clusterQuery, sshProcessQuery)

	r.sshProcessIndicators, err = f.processIndicatorStore.SearchRawProcessIndicators(ctx, sshQuery)
	if err != nil {
		return err
	}

	hasIndicatorsQuery := clusterQuery.Clone()
	hasIndicatorsQuery.Pagination.Limit = 1
	result, err := f.processIndicatorStore.Search(ctx, hasIndicatorsQuery)
	if err != nil {
		return err
	}
	r.hasProcessIndicators = len(result) != 0

	flowStore, err := f.networkFlowDataStore.GetFlowStore(ctx, domain.Cluster().ID())
	if err != nil {
		return err
	} else if flowStore == nil {
		return errors.Errorf("no flows found for cluster %q", domain.Cluster().ID())
	}
	r.networkFlows, _, err = flowStore.GetAllFlows(ctx, nil)
	if err != nil {
		return err
	}

	r.notifiers, err = f.notifierDataStore.GetNotifiers(ctx)
	if err != nil {
		return err
	}

	r.roles, err = f.roleDataStore.SearchRawRoles(ctx, clusterQuery)
	if err != nil {
		return err
	}

	r.bindings, err = f.bindingDataStore.SearchRawRoleBindings(ctx, clusterQuery)
	if err != nil {
		return err
	}

	alertQuery := search.ConjunctionQuery(
		clusterQuery,
		search.NewQueryBuilder().AddExactMatches(search.ViolationState, storage.ViolationState_ACTIVE.String(), storage.ViolationState_SNOOZED.String()).ProtoQuery(),
	)
	alertQuery.Pagination = infPagination
	r.unresolvedAlerts, err = f.alertStore.SearchListAlerts(ctx, alertQuery)
	if err != nil {
		return err
	}

	r.complianceOperatorResults = make(map[string][]*storage.ComplianceOperatorCheckResult)
	walkFn := func() error {
		r.complianceOperatorResults = make(map[string][]*storage.ComplianceOperatorCheckResult)
		return f.complianceOperatorResultStore.Walk(ctx, func(c *storage.ComplianceOperatorCheckResult) error {
			if c.GetClusterId() != clusterID {
				return nil
			}
			rule := c.Annotations[v1alpha1.RuleIDAnnotationKey]
			if rule == "" {
				log.Errorf("Expected rule annotation for %+v", c)
				return nil
			}
			r.complianceOperatorResults[rule] = append(r.complianceOperatorResults[rule], c)
			return nil
		})
	}
	if err := pgutils.RetryIfPostgres(walkFn); err != nil {
		return err
	}

	// Flatten the files so we can do direct lookups on the nested values
	for _, n := range scrapeResults {
		totalNodeFiles := data.FlattenFileMap(n.GetFiles())
		n.Files = totalNodeFiles
	}

	r.hostScrape = scrapeResults

	r.nodeResults = getNodeResults(scrapeResults)

	cisKubernetesStandardID, err := f.standardsRepo.GetCISKubernetesStandardID()
	if err != nil {
		return err
	}

	kubeCISRunResults, err := f.complianceStore.GetLatestRunResults(ctx, clusterID, cisKubernetesStandardID, 0)
	if err == nil && kubeCISRunResults.LastSuccessfulResults != nil {
		r.cisKubernetesRunCheck = true
	}

	return nil
}

func getNodeResults(scrapeResults map[string]*compliance.ComplianceReturn) map[string]map[string]*compliance.ComplianceStandardResult {
	nodeResults := make(map[string]map[string]*compliance.ComplianceStandardResult, len(scrapeResults))
	for nodeName, n := range scrapeResults {
		if n.GetEvidence() == nil {
			continue
		}

		result, err := decompressNodeResults(n.GetEvidence())
		if err != nil {
			log.Error(errors.Wrapf(err, "unable to decompress compliance results from node %s", nodeName))
			continue
		}
		nodeResults[nodeName] = result
	}
	return nodeResults
}

func decompressNodeResults(chunk *compliance.GZIPDataChunk) (map[string]*compliance.ComplianceStandardResult, error) {
	reader := bytes.NewReader(chunk.GetGzip())
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(gzReader.Close)

	var wrappedRunResults compress.ResultWrapper
	if err := easyjson.UnmarshalFromReader(gzReader, &wrappedRunResults); err != nil {
		return nil, err
	}
	return wrappedRunResults.ResultMap, nil
}
