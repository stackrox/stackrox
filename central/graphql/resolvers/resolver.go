package resolvers

//go:generate go run ./gen

import (
	"context"
	"fmt"
	"reflect"

	activeComponent "github.com/stackrox/rox/central/activecomponent/datastore"
	violationsDatastore "github.com/stackrox/rox/central/alert/datastore"
	"github.com/stackrox/rox/central/apitoken/backend"
	"github.com/stackrox/rox/central/audit"
	clusterDatastore "github.com/stackrox/rox/central/cluster/datastore"
	clusterCVEEdgeDataStore "github.com/stackrox/rox/central/clustercveedge/datastore"
	"github.com/stackrox/rox/central/compliance/aggregation"
	complianceDS "github.com/stackrox/rox/central/compliance/datastore"
	complianceManager "github.com/stackrox/rox/central/compliance/manager"
	"github.com/stackrox/rox/central/compliance/manager/service"
	complianceService "github.com/stackrox/rox/central/compliance/service"
	complianceStandards "github.com/stackrox/rox/central/compliance/standards"
	complianceOperatorManager "github.com/stackrox/rox/central/complianceoperator/manager"
	componentCVEEdgeDataStore "github.com/stackrox/rox/central/componentcveedge/datastore"
	clusterCVEDataStore "github.com/stackrox/rox/central/cve/cluster/datastore"
	legacyImageCVEDataStore "github.com/stackrox/rox/central/cve/datastore"
	"github.com/stackrox/rox/central/cve/fetcher"
	imageCVEDataStore "github.com/stackrox/rox/central/cve/image/datastore"
	cveMatcher "github.com/stackrox/rox/central/cve/matcher"
	nodeCVEDataStore "github.com/stackrox/rox/central/cve/node/datastore"
	deploymentDatastore "github.com/stackrox/rox/central/deployment/datastore"
	groupDataStore "github.com/stackrox/rox/central/group/datastore"
	imageDatastore "github.com/stackrox/rox/central/image/datastore"
	imageComponentDataStore "github.com/stackrox/rox/central/imagecomponent/datastore"
	imageComponentEdgeDataStore "github.com/stackrox/rox/central/imagecomponentedge/datastore"
	imageCVEEdgeDataStore "github.com/stackrox/rox/central/imagecveedge/datastore"
	mitreDataStore "github.com/stackrox/rox/central/mitre/datastore"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/datastore"
	nfDS "github.com/stackrox/rox/central/networkgraph/flow/datastore"
	npDS "github.com/stackrox/rox/central/networkpolicies/datastore"
	nodeDataStore "github.com/stackrox/rox/central/node/globaldatastore"
	nodeComponentDataStore "github.com/stackrox/rox/central/nodecomponent/datastore"
	nodeComponentCVEEdgeDataStore "github.com/stackrox/rox/central/nodecomponentcveedge/datastore"
	notifierDataStore "github.com/stackrox/rox/central/notifier/datastore"
	"github.com/stackrox/rox/central/notifier/processor"
	podDatastore "github.com/stackrox/rox/central/pod/datastore"
	policyDatastore "github.com/stackrox/rox/central/policy/datastore"
	baselineStore "github.com/stackrox/rox/central/processbaseline/datastore"
	processIndicatorStore "github.com/stackrox/rox/central/processindicator/datastore"
	k8sroleStore "github.com/stackrox/rox/central/rbac/k8srole/datastore"
	k8srolebindingStore "github.com/stackrox/rox/central/rbac/k8srolebinding/datastore"
	riskDataStore "github.com/stackrox/rox/central/risk/datastore"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/central/role/resources"
	secretDataStore "github.com/stackrox/rox/central/secret/datastore"
	serviceAccountDataStore "github.com/stackrox/rox/central/serviceaccount/datastore"
	vulnReqDataStore "github.com/stackrox/rox/central/vulnerabilityrequest/datastore"
	"github.com/stackrox/rox/central/vulnerabilityrequest/manager/querymgr"
	"github.com/stackrox/rox/central/vulnerabilityrequest/manager/requestmgr"
	watchedImageDataStore "github.com/stackrox/rox/central/watchedimage/datastore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	auditPkg "github.com/stackrox/rox/pkg/audit"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
)

// Resolver is the root GraphQL resolver
type Resolver struct {
	ActiveComponent               activeComponent.DataStore
	ComplianceAggregator          aggregation.Aggregator
	APITokenBackend               backend.Backend
	ClusterDataStore              clusterDatastore.DataStore
	ClusterCVEDataStore           clusterCVEDataStore.DataStore
	ComplianceDataStore           complianceDS.DataStore
	ComplianceStandardStore       complianceStandards.Repository
	ComplianceService             v1.ComplianceServiceServer
	ComplianceManagementService   v1.ComplianceManagementServiceServer
	ComplianceManager             complianceManager.ComplianceManager
	clusterCVEEdgeDataStore       clusterCVEEdgeDataStore.DataStore
	ComponentCVEEdgeDataStore     componentCVEEdgeDataStore.DataStore
	CVEDataStore                  legacyImageCVEDataStore.DataStore
	ImageCVEDataStore             imageCVEDataStore.DataStore
	NodeCVEDataStore              nodeCVEDataStore.DataStore
	DeploymentDataStore           deploymentDatastore.DataStore
	PodDataStore                  podDatastore.DataStore
	ImageDataStore                imageDatastore.DataStore
	ImageComponentDataStore       imageComponentDataStore.DataStore
	NodeComponentDataStore        nodeComponentDataStore.DataStore
	NodeComponentCVEEdgeDataStore nodeComponentCVEEdgeDataStore.DataStore
	ImageComponentEdgeDataStore   imageComponentEdgeDataStore.DataStore
	ImageCVEEdgeDataStore         imageCVEEdgeDataStore.DataStore
	GroupDataStore                groupDataStore.DataStore
	NamespaceDataStore            namespaceDataStore.DataStore
	NetworkFlowDataStore          nfDS.ClusterDataStore
	NetworkPoliciesStore          npDS.DataStore
	NodeGlobalDataStore           nodeDataStore.GlobalDataStore
	NotifierStore                 notifierDataStore.DataStore
	PolicyDataStore               policyDatastore.DataStore
	ProcessIndicatorStore         processIndicatorStore.DataStore
	K8sRoleStore                  k8sroleStore.DataStore
	K8sRoleBindingStore           k8srolebindingStore.DataStore
	RoleDataStore                 roleDataStore.DataStore
	RiskDataStore                 riskDataStore.DataStore
	SecretsDataStore              secretDataStore.DataStore
	ServiceAccountsDataStore      serviceAccountDataStore.DataStore
	ViolationsDataStore           violationsDatastore.DataStore
	BaselineDataStore             baselineStore.DataStore
	WatchedImageDataStore         watchedImageDataStore.DataStore
	orchestratorIstioCVEManager   fetcher.OrchestratorIstioCVEManager
	cveMatcher                    *cveMatcher.CVEMatcher
	manager                       complianceOperatorManager.Manager
	mitreStore                    mitreDataStore.MitreAttackReadOnlyDataStore
	vulnReqMgr                    requestmgr.Manager
	vulnReqQueryMgr               querymgr.VulnReqQueryManager
	vulnReqStore                  vulnReqDataStore.DataStore
	AuditLogger                   auditPkg.Auditor
}

// New returns a Resolver wired into the relevant data stores
func New() *Resolver {
	resolver := &Resolver{
		ActiveComponent:             activeComponent.Singleton(),
		ComplianceAggregator:        aggregation.Singleton(),
		APITokenBackend:             backend.Singleton(),
		ComplianceDataStore:         complianceDS.Singleton(),
		ComplianceStandardStore:     complianceStandards.RegistrySingleton(),
		ComplianceManagementService: service.Singleton(),
		ComplianceManager:           complianceManager.Singleton(),
		ComplianceService:           complianceService.Singleton(),
		ClusterDataStore:            clusterDatastore.Singleton(),
		clusterCVEEdgeDataStore:     clusterCVEEdgeDataStore.Singleton(),
		ComponentCVEEdgeDataStore:   componentCVEEdgeDataStore.Singleton(),
		DeploymentDataStore:         deploymentDatastore.Singleton(),
		PodDataStore:                podDatastore.Singleton(),
		ImageDataStore:              imageDatastore.Singleton(),
		ImageComponentDataStore:     imageComponentDataStore.Singleton(),
		ImageComponentEdgeDataStore: imageComponentEdgeDataStore.Singleton(),
		ImageCVEEdgeDataStore:       imageCVEEdgeDataStore.Singleton(),
		GroupDataStore:              groupDataStore.Singleton(),
		NamespaceDataStore:          namespaceDataStore.Singleton(),
		NetworkPoliciesStore:        npDS.Singleton(),
		NetworkFlowDataStore:        nfDS.Singleton(),
		NodeGlobalDataStore:         nodeDataStore.Singleton(),
		NotifierStore:               notifierDataStore.Singleton(),
		PolicyDataStore:             policyDatastore.Singleton(),
		ProcessIndicatorStore:       processIndicatorStore.Singleton(),
		K8sRoleStore:                k8sroleStore.Singleton(),
		K8sRoleBindingStore:         k8srolebindingStore.Singleton(),
		RiskDataStore:               riskDataStore.Singleton(),
		RoleDataStore:               roleDataStore.Singleton(),
		SecretsDataStore:            secretDataStore.Singleton(),
		ServiceAccountsDataStore:    serviceAccountDataStore.Singleton(),
		ViolationsDataStore:         violationsDatastore.Singleton(),
		BaselineDataStore:           baselineStore.Singleton(),
		WatchedImageDataStore:       watchedImageDataStore.Singleton(),
		orchestratorIstioCVEManager: fetcher.SingletonManager(),
		cveMatcher:                  cveMatcher.Singleton(),
		manager:                     complianceOperatorManager.Singleton(),
		mitreStore:                  mitreDataStore.Singleton(),
		vulnReqMgr:                  requestmgr.Singleton(),
		vulnReqQueryMgr:             querymgr.Singleton(),
		vulnReqStore:                vulnReqDataStore.Singleton(),
		AuditLogger:                 audit.New(processor.Singleton()),
	}
	if features.PostgresDatastore.Enabled() {
		resolver.ClusterCVEDataStore = clusterCVEDataStore.Singleton()
		resolver.ImageCVEDataStore = imageCVEDataStore.Singleton()
		resolver.NodeCVEDataStore = nodeCVEDataStore.Singleton()
		resolver.NodeComponentCVEEdgeDataStore = nodeComponentCVEEdgeDataStore.Singleton()
		resolver.NodeComponentDataStore = nodeComponentDataStore.Singleton()
	} else {
		resolver.CVEDataStore = legacyImageCVEDataStore.Singleton()
	}
	return resolver
}

//lint:file-ignore U1000 It's okay for some of the variables below to be unused.
var (
	readAlerts                           = readAuth(resources.Alert)
	readTokens                           = readAuth(resources.APIToken)
	readClusters                         = readAuth(resources.Cluster)
	readCompliance                       = readAuth(resources.Compliance)
	readComplianceRuns                   = readAuth(resources.ComplianceRuns)
	readComplianceRunSchedule            = readAuth(resources.ComplianceRunSchedule)
	readCVEs                             = readAuth(resources.CVE)
	readDeployments                      = readAuth(resources.Deployment)
	readGroups                           = readAuth(resources.Group)
	readImages                           = readAuth(resources.Image)
	readIndicators                       = readAuth(resources.Indicator)
	readNamespaces                       = readAuth(resources.Namespace)
	readNetPolicies                      = readAuth(resources.NetworkPolicy)
	readNodes                            = readAuth(resources.Node)
	readNotifiers                        = readAuth(resources.Notifier)
	readPolicies                         = readAuth(resources.Policy)
	readK8sRoles                         = readAuth(resources.K8sRole)
	readK8sRoleBindings                  = readAuth(resources.K8sRoleBinding)
	readK8sSubjects                      = readAuth(resources.K8sSubject)
	readRisks                            = readAuth(resources.Risk)
	readRoles                            = readAuth(resources.Role)
	readSecrets                          = readAuth(resources.Secret)
	readServiceAccounts                  = readAuth(resources.ServiceAccount)
	readBaselines                        = readAuth(resources.ProcessWhitelist)
	readVulnerabilityRequestsOrApprovals = anyReadAuth(resources.VulnerabilityManagementRequests, resources.VulnerabilityManagementApprovals)

	writeAlerts                           = writeAuth(resources.Alert)
	writeCompliance                       = writeAuth(resources.Compliance)
	writeComplianceRuns                   = writeAuth(resources.ComplianceRuns)
	writeComplianceRunSchedule            = writeAuth(resources.ComplianceRunSchedule)
	writeIndicators                       = writeAuth(resources.Indicator)
	writeVulnerabilityRequests            = writeAuth(resources.VulnerabilityManagementRequests)
	writeVulnerabilityApprovals           = writeAuth(resources.VulnerabilityManagementApprovals)
	writeVulnerabilityRequestsOrApprovals = anyWriteAuth(resources.VulnerabilityManagementRequests, resources.VulnerabilityManagementApprovals)
)

type authorizerOverride struct{}

// SetAuthorizerOverride returns a context that will override the default permissions checking with custom
// logic. This is for testing only. It also feels pretty dangerous.
func SetAuthorizerOverride(ctx context.Context, authorizer authz.Authorizer) context.Context {
	return context.WithValue(ctx, authorizerOverride{}, authorizer)
}

func applyAuthorizer(authorizer authz.Authorizer) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		override := ctx.Value(authorizerOverride{})
		if override != nil {
			return override.(authz.Authorizer).Authorized(ctx, "graphql")
		}
		return authorizer.Authorized(ctx, "graphql")
	}
}

func readAuth(resource permissions.ResourceMetadata) func(ctx context.Context) error {
	return applyAuthorizer(user.With(permissions.View(resource)))
}

func anyReadAuth(resources ...permissions.ResourceMetadata) func(ctx context.Context) error {
	authorizers := make([]authz.Authorizer, 0, len(resources))
	for _, res := range resources {
		authorizers = append(authorizers, user.With(permissions.View(res)))
	}
	return applyAuthorizer(or.Or(authorizers...))
}

func writeAuth(resource permissions.ResourceMetadata) func(ctx context.Context) error {
	return applyAuthorizer(user.With(permissions.Modify(resource)))
}

func anyWriteAuth(resources ...permissions.ResourceMetadata) func(ctx context.Context) error {
	authorizers := make([]authz.Authorizer, 0, len(resources))
	for _, res := range resources {
		authorizers = append(authorizers, user.With(permissions.Modify(res)))
	}
	return applyAuthorizer(or.Or(authorizers...))
}

func stringSlice(inputSlice interface{}) []string {
	r := reflect.ValueOf(inputSlice)
	output := make([]string, r.Len())
	for i := 0; i < r.Len(); i++ {
		output[i] = fmt.Sprint(r.Index(i).Interface())
	}
	return output
}

// NewMock returns an empty Resolver for use in testing whether the GraphQL schema can be compiled
func NewMock() *Resolver {
	return &Resolver{}
}
