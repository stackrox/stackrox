import static Services.waitForViolation
import static io.stackrox.proto.storage.RoleOuterClass.Access.NO_ACCESS
import static io.stackrox.proto.storage.RoleOuterClass.Access.READ_ACCESS
import static io.stackrox.proto.storage.RoleOuterClass.Access.READ_WRITE_ACCESS
import static io.stackrox.proto.storage.RoleOuterClass.SimpleAccessScope.newBuilder
import static services.ClusterService.DEFAULT_CLUSTER_NAME
import static util.Helpers.withRetry

import orchestratormanager.OrchestratorTypes

import io.stackrox.proto.api.v1.ApiTokenService.GenerateTokenResponse
import io.stackrox.proto.api.v1.NamespaceServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass as SSOC
import io.stackrox.proto.storage.DeploymentOuterClass
import io.stackrox.proto.storage.RoleOuterClass

import objects.Deployment
import services.AlertService
import services.ApiTokenService
import services.BaseService
import services.ClusterService
import services.DeploymentService
import services.ImageService
import services.NamespaceService
import services.NetworkGraphService
import services.RoleService
import services.SearchService
import services.SecretService
import services.SummaryService
import util.Env
import util.NetworkGraphUtil

import org.junit.AssumptionViolatedException
import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Unroll

@Tag("BAT")
class SACTest extends BaseSpecification {
    static final private String IMAGE = "quay.io/rhacs-eng/qa-multi-arch:nginx-unprivileged-1.25.2"
    static final private String DEPLOYMENTNGINX_NAMESPACE_QA1 = "sac-deploymentnginx-qa1"
    static final private String NAMESPACE_QA1 = "qa-test1"
    static final private String DEPLOYMENTNGINX_NAMESPACE_QA2 = "sac-deploymentnginx-qa2"
    static final private String NAMESPACE_QA2 = "qa-test2"
    static final private String SECRETNAME = "sac-secret"
    static final protected String ALLACCESSTOKEN = "allAccessToken"
    static final protected String NOACCESSTOKEN = "noAccess"
    static final protected Deployment DEPLOYMENT_QA1 = new Deployment()
            .setName(DEPLOYMENTNGINX_NAMESPACE_QA1)
            .setImage(IMAGE)
            .addPort(22, "TCP")
            .addAnnotation("test", "annotation")
            .setEnv(["CLUSTER_NAME": "main"])
            .setNamespace(NAMESPACE_QA1)
            .addLabel("app", "test")
    static final protected Deployment DEPLOYMENT_QA2 = new Deployment()
            .setName(DEPLOYMENTNGINX_NAMESPACE_QA2)
            .setImage(IMAGE)
            .addPort(22, "TCP")
            .addAnnotation("test", "annotation")
            .setEnv(["CLUSTER_NAME": "main"])
            .setNamespace(NAMESPACE_QA2)
            .addLabel("app", "test")

    static final private List<Deployment> DEPLOYMENTS = [DEPLOYMENT_QA1, DEPLOYMENT_QA2,]

    static final private UNSTABLE_FLOWS = [
            // monitoring doesn't keep a persistent outgoing connection, so we might or might not see this flow.
            "stackrox/monitoring -> INTERNET",
    ] as Set

    // Increase the timeout conditionally based on whether we are running race-detection builds or within OpenShift
    // environments. Both take longer than the default values.
    static final private Integer WAIT_FOR_VIOLATION_TIMEOUT =
            isRaceBuild() ? 600 : ((Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT) ? 100 : 60)

    static final private Integer WAIT_FOR_RISK_RETRIES =
            isRaceBuild() ? 300 : ((Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT) ? 80 : 50)
    static final private String DENY_ALL = isPostgresRun() ?
        "ffffffff-ffff-fff4-f5ff-fffffffffffe" :
        'io.stackrox.authz.accessscope.denyall'

    @Shared
    private Map<String, RoleOuterClass.Access> allResourcesAccess

    @Shared
    private Map<String, List<String>> tokenToRoles

    def setup() {
        BaseService.useBasicAuth()
    }

    def setupSpec() {
        // Make sure we scan the image initially to make reprocessing faster.
        def img = ImageService.scanImage(TEST_IMAGE, false)
        assert img.hasScan()

        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
        }
        // Make sure each deployment has caused at least one alert
        assert waitForViolation(DEPLOYMENT_QA1.name, "Secure Shell (ssh) Port Exposed",
                WAIT_FOR_VIOLATION_TIMEOUT)
        assert waitForViolation(DEPLOYMENT_QA2.name, "Secure Shell (ssh) Port Exposed",
                WAIT_FOR_VIOLATION_TIMEOUT)

        // Make sure each deployment has a risk score.
        listDeployments().each { DeploymentOuterClass.ListDeployment dep ->
            try {
                withRetry(WAIT_FOR_RISK_RETRIES, 2) {
                    assert DeploymentService.getDeploymentWithRisk(dep.id).hasRisk()
                }
            } catch (Exception e) {
                throw new AssumptionViolatedException("Failed to retrieve risk from deployment ${dep.name}", e)
            }
        }

        allResourcesAccess = RoleService.resources.resourcesList.collectEntries { [it, READ_WRITE_ACCESS] }

        def remoteQaTest1 = createAccessScope(DEFAULT_CLUSTER_NAME, "qa-test1")
        def remoteQaTest2 = createAccessScope(DEFAULT_CLUSTER_NAME, "qa-test2")

        def noaccess = createRole(DENY_ALL, allResourcesAccess)

        tokenToRoles = [
                (NOACCESSTOKEN)                   : [noaccess],
                (ALLACCESSTOKEN)                  : [createRole(UNRESTRICTED_SCOPE_ID, allResourcesAccess)],
                "deployments-access-token"        : [createRole(remoteQaTest2.id,
                        ["Deployment": READ_ACCESS, "DeploymentExtension": READ_ACCESS])],
                "getSummaryCountsToken"           : [createRole(remoteQaTest1.id, allResourcesAccess)],
                "listSecretsToken"                : [createRole(UNRESTRICTED_SCOPE_ID, ["Secret": READ_ACCESS])],
                "searchAlertsToken"               : [createRole(remoteQaTest1.id, ["Alert": READ_ACCESS]), noaccess],
                "searchDeploymentsToken"          : [createRole(remoteQaTest1.id,
                        ["Deployment": READ_ACCESS]), noaccess],
                "searchImagesToken"               : [createRole(remoteQaTest1.id, ["Image": READ_ACCESS]), noaccess],
                "searchNamespacesToken"           : [createRole(remoteQaTest1.id,
                        ["Namespace": READ_ACCESS]), noaccess],
                "searchDeploymentsImagesToken"    : [createRole(remoteQaTest1.id,
                        ["Deployment": READ_ACCESS, "Image": READ_ACCESS]), noaccess],
                "stackroxNetFlowsToken"           : [createRole(createAccessScope(DEFAULT_CLUSTER_NAME, "stackrox").id,
                        ["Deployment": READ_ACCESS, "NetworkGraph": READ_ACCESS]),
                                                     createRole(UNRESTRICTED_SCOPE_ID, ["Cluster": READ_ACCESS]),
                                                     noaccess],
                "kubeSystemDeploymentsImagesToken": [createRole(createAccessScope(
                        DEFAULT_CLUSTER_NAME, "kube-system").id, ["Deployment": READ_ACCESS, "Image": READ_ACCESS]),
                                                     noaccess],
                "aggregatedToken"                 : [createRole(remoteQaTest2.id, ["Deployment": READ_ACCESS]),
                                                     createRole(remoteQaTest1.id, ["Deployment": NO_ACCESS]),
                                                     noaccess],
                "getClusterToken"                 : [createRole(remoteQaTest1.id, ["Cluster": READ_ACCESS]),
                                                     noaccess],
        ]
    }

    def cleanupSpec() {
        BaseService.useBasicAuth()
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
        [NAMESPACE_QA1, NAMESPACE_QA2].forEach {
            ns ->
                orchestrator.deleteNamespace(ns)
                orchestrator.waitForNamespaceDeletion(ns)
        }
        cleanupRole(*(tokenToRoles.values().flatten().unique()))
    }

    static getAlertCount() {
        return AlertService.getViolations().size()
    }

    static getImageCount() {
        return ImageService.getImages().size()
    }

    static getDeploymentCount() {
        return DeploymentService.listDeploymentsSearch().deploymentsCount
    }

    static getNamespaceCount() {
        return NamespaceService.getNamespaces().size()
    }

    GenerateTokenResponse useToken(String tokenName) {
        GenerateTokenResponse token = ApiTokenService.generateToken(tokenName, *(tokenToRoles.get(tokenName)))
        BaseService.useApiToken(token.token)
        token
    }

    static getSpecificQuery(String category) {
        def queryString = category + ":*"
        def query = SSOC.RawSearchRequest.newBuilder()
                .setQuery(queryString)
                .build()
        return query
    }

    def createSecret(String namespace) {
        String secID = orchestrator.createSecret(SECRETNAME, namespace)
        SecretService.waitForSecret(secID, 10)
    }

    def deleteSecret(String namespace) {
        orchestrator.deleteSecret(SECRETNAME, namespace)
    }

    def cleanupRole(String... roleName) {
        roleName.each {
            try {
                def role = RoleService.getRole(it)
                RoleService.deleteRole(role.name)
                RoleService.deleteAccessScope(role.accessScopeId)
            } catch (Exception e) {
                log.error("Error deleting role ${it} or associated access scope: " + e)
            }
        }
    }

    String createRole(String sacId, Map<String, RoleOuterClass.Access> resources) {
        RoleService.createRoleWithScopeAndPermissionSet("SACv2 Test Automation Role " + UUID.randomUUID(),
                sacId, resources
        ).name
    }

    def createAccessScope(String clusterName, String namespaceName) {
        RoleService.createAccessScope(newBuilder()
                .setName(UUID.randomUUID().toString())
                .setRules(RoleOuterClass.SimpleAccessScope.Rules.newBuilder()
                        .addIncludedNamespaces(RoleOuterClass.SimpleAccessScope.Rules.Namespace.newBuilder()
                                .setClusterName(clusterName)
                                .setNamespaceName(namespaceName)))
                .build())
    }

    @Unroll
    def "Verify that only namespace #sacResource is visible when using SAC"() {
        when:
        "Create test API token with a built-in role"
        useToken("deployments-access-token")
        then:
        "Call API and verify data returned is within scoped access"
        def result = DeploymentService.listDeployments()
        log.info result.toString()
        assert result.size() == 1
        assert DeploymentService.getDeploymentWithRisk(result.first().id).hasRisk()
        def resourceNotAllowed = result.find { it.namespace != sacResource }
        assert resourceNotAllowed == null

        where:
        "Data inputs are: "
        sacResource   | _
        NAMESPACE_QA2 | _
    }

    def "Verify GetSummaryCounts using a token without access receives no results"() {
        when:
        "GetSummaryCounts is called using a token without access"
        createSecret(DEPLOYMENT_QA1.namespace)
        useToken(NOACCESSTOKEN)
        def result = SummaryService.getCounts()
        then:
        "Verify GetSumamryCounts returns no results"
        assert result.getNumDeployments() == 0
        assert result.getNumSecrets() == 0
        assert result.getNumNodes() == 0
        assert result.getNumClusters() == 0
        assert result.getNumImages() == 0
        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
        deleteSecret(DEPLOYMENT_QA1.namespace)
    }

    def "Verify GetSummaryCounts using a token with partial access receives partial results"() {
        when:
        "GetSummaryCounts is called using a token with restricted access"
        createSecret(DEPLOYMENT_QA1.namespace)
        createSecret(DEPLOYMENT_QA2.namespace)
        useToken("getSummaryCountsToken")
        def result = SummaryService.getCounts()
        then:
        "Verify correct counts are returned by GetSummaryCounts"
        assert result.getNumDeployments() == 1
        assert result.getNumSecrets() == orchestrator.getSecretCount(DEPLOYMENT_QA1.namespace)
        assert result.getNumImages() == 1
        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
        deleteSecret(DEPLOYMENT_QA1.namespace)
        deleteSecret(DEPLOYMENT_QA2.namespace)
    }

    def "Verify GetSummaryCounts using a token with all access receives all results"() {
        when:
        "GetSummaryCounts is called using a token with all access"
        createSecret(DEPLOYMENT_QA1.namespace)
        createSecret(DEPLOYMENT_QA2.namespace)
        useToken(ALLACCESSTOKEN)
        def result = SummaryService.getCounts()
        then:
        "Verify results are returned in each category"
        assert result.getNumDeployments() >= 2
        // These may be created by other tests so it's hard to know the exact number.
        assert result.getNumSecrets() >= 2
        assert result.getNumNodes() > 0
        assert result.getNumClusters() >= 1
        assert result.getNumImages() >= 1
        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
        deleteSecret(DEPLOYMENT_QA1.namespace)
        deleteSecret(DEPLOYMENT_QA2.namespace)
    }

    @Unroll
    def "Verify alerts count is scoped"() {
        given:
        def query = SSOC.RawQuery.newBuilder().setQuery(
                "Deployment:${DEPLOYMENT_QA1.name},${DEPLOYMENT_QA2.name}"
        ).build()

        when:
        def alertsCount = { String tokenName ->
            BaseService.useBasicAuth()
            useToken(tokenName)
            AlertService.alertClient.countAlerts(query).count
        }

        then:
        assert alertsCount(NOACCESSTOKEN) == 0
        // getSummaryCountsToken has access only to QA1 deployment while
        // ALLACCESSTOKEN has access to QA1 and QA2. Since deployments are identical
        // number of alerts for ALLACCESSTOKEN should be twice of getSummaryCountsToken.
        assert 2 * alertsCount("getSummaryCountsToken") == alertsCount(ALLACCESSTOKEN)
    }

    def "Verify ListSecrets using a token without access receives no results"() {
        when:
        "ListSecrets is called using a token without view access to Secrets"
        createSecret(DEPLOYMENT_QA1.namespace)
        useToken(NOACCESSTOKEN)
        def result = SecretService.listSecrets()
        then:
        "Verify no secrets are returned by ListSecrets"
        assert result.secretsCount == 0
        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
        deleteSecret(DEPLOYMENT_QA1.namespace)
    }

    def "Verify ListSecrets using a token with access receives some results"() {
        when:
        "ListSecrets is called using a token with view access to Secrets"
        createSecret(DEPLOYMENT_QA1.namespace)
        createSecret(DEPLOYMENT_QA2.namespace)
        useToken("listSecretsToken")
        def result = SecretService.listSecrets()

        then:
        "Verify no secrets are returned by ListSecrets"
        assert result.secretsCount > 0

        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
        deleteSecret(DEPLOYMENT_QA1.namespace)
        deleteSecret(DEPLOYMENT_QA2.namespace)
    }

    @Unroll
    def "Verify Search on #category resources using the #tokenName token returns #numResults results"() {
        when:
        "A search is performed using the given token"
        def query = getSpecificQuery(category)
        useToken(tokenName)
        def result = SearchService.search(query)

        then:
        "Verify the specified number of results are returned"
        assert result.resultsCount == numResults

        where:
        "Data inputs are: "
        tokenName                | category     | numResults
        NOACCESSTOKEN            | "Cluster"    | 0
        "searchDeploymentsToken" | "Deployment" | 1
        "searchImagesToken"      | "Image"      | 1
    }

    @Unroll
    def "Verify Search on #category resources using the #tokenName token returns >= #minReturned results"() {
        when:
        "A search is performed using the given token"
        def query = getSpecificQuery(category)
        useToken(tokenName)
        def result = SearchService.search(query)

        then:
        "Verify >= the specified number of results are returned"
        assert result.resultsCount >= minReturned

        where:
        "Data inputs are: "
        tokenName           | category     | minReturned
        "searchAlertsToken" | "Deployment" | 1
    }

    def "Verify Search using the allAccessToken returns results for all search categories"() {
        when:
        "A search is performed using the allAccessToken"
        createSecret(DEPLOYMENT_QA1.namespace)
        def query = getSpecificQuery("Cluster")
        useToken(ALLACCESSTOKEN)
        def result = SearchService.search(query)
        then:
        "Verify something was returned for every search category"
        for (SSOC.SearchResponse.Count numResults : result.countsList) {
            // Policies are globally scoped so our cluster-scoped query won't return any
            if (numResults.category == SSOC.SearchCategory.POLICIES ||
                numResults.category == SSOC.SearchCategory.POLICY_CATEGORIES ||
                numResults.category == SSOC.SearchCategory.IMAGE_INTEGRATIONS) {
                continue
            }
            assert numResults.count > 0
        }
        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
        deleteSecret(DEPLOYMENT_QA1.namespace)
    }

    @Unroll
    def "Verify Autocomplete on #category resources using the #tokenName token returns #numResults results"() {
        when:
        "Search is called using a token without view access to Deployments"
        def query = getSpecificQuery(category)
        useToken(tokenName)
        def result = SearchService.autocomplete(query)
        then:
        "Verify no results are returned by Search"
        assert result.getValuesCount() == numResults

        where:
        "Data inputs are: "
        tokenName                | category     | numResults
        NOACCESSTOKEN            | "Deployment" | 0
        NOACCESSTOKEN            | "Image"      | 0
        "searchDeploymentsToken" | "Deployment" | 1
        "searchImagesToken"      | "Image"      | 1
        "searchNamespacesToken"  | "Namespace"  | 1
    }

    @Unroll
    def "Verify Autocomplete on #category resources using the #tokenName token returns >= to #minReturned results"() {
        when:
        "Autocomplete is called using the given token"
        def query = getSpecificQuery(category)
        useToken(tokenName)
        def result = SearchService.autocomplete(query)
        then:
        "Verify exactly the expected number of results are returned"
        assert result.getValuesCount() >= minReturned

        where:
        "Data inputs are: "
        tokenName      | category     | minReturned
        ALLACCESSTOKEN | "Deployment" | 2
        ALLACCESSTOKEN | "Image"      | 1
    }

    @Unroll
    def "Verify using the #tokenName token with the #service service returns #numReturned results"() {
        when:
        "The service under test is called using the given token"
        useToken(tokenName)
        def result = resultCountFunc()
        then:
        "Verify exactly the expected number of results are returned"
        assert result == numReturned

        where:
        "Data inputs are: "
        tokenName                | numReturned | resultCountFunc          | service
        NOACCESSTOKEN            | 0           | this.&getDeploymentCount | "Deployment"
        "searchDeploymentsToken" | 1           | this.&getDeploymentCount | "Deployment"
        NOACCESSTOKEN            | 0           | this.&getAlertCount      | "Alert"
        NOACCESSTOKEN            | 0           | this.&getImageCount      | "Image"
        NOACCESSTOKEN            | 0           | this.&getNamespaceCount  | "Namespace"
        "searchNamespacesToken"  | 1           | this.&getNamespaceCount  | "Namespace"
        "searchImagesToken"      | 1           | this.&getImageCount      | "Image"
    }

    @Unroll
    def "Verify using the #tokenName token with the #service service returns >= to #minNumReturned results"() {
        when:
        "The service under test is called using the given token"
        useToken(tokenName)
        def result = resultCountFunc()
        then:
        "Verify greater than or equal to the expected number of results are returned"
        assert result >= minNumReturned

        where:
        "Data inputs are: "
        tokenName           | minNumReturned | resultCountFunc          | service
        ALLACCESSTOKEN      | 1              | this.&getAlertCount      | "Alert"
        "searchAlertsToken" | 1              | this.&getAlertCount      | "Alert"
        ALLACCESSTOKEN      | 1              | this.&getImageCount      | "Image"
        ALLACCESSTOKEN      | 2              | this.&getDeploymentCount | "Deployment"
        ALLACCESSTOKEN      | 2              | this.&getNamespaceCount  | "Namespace"
    }

    static getNamespaceId(String name) {
        def namespaces = NamespaceService.getNamespaces()
        for (NamespaceServiceOuterClass.Namespace namespace : namespaces) {
            if (namespace.getMetadata().name == name) {
                return namespace.getMetadata().id
            }
        }
        return null
    }

    @Unroll
    def "Verify Namespace service SAC is enforced properly when using the #tokenName token"() {
        when:
        "We try to get one namespace we have access to and one namespace we don't have access to "
        def qa1NamespaceId = getNamespaceId(DEPLOYMENT_QA1.namespace)
        def qa2NamespaceId = getNamespaceId(DEPLOYMENT_QA2.namespace)
        useToken(tokenName)
        def qa1 = NamespaceService.getNamespace(qa1NamespaceId)
        def qa2 = NamespaceService.getNamespace(qa2NamespaceId)
        then:
        "We should get results for the namespace we have access to and null for the namespace we don't have access to"
        // Either the value should be null and it is, else the value is not null
        assert qa1Null && qa1 == null || qa1 != null
        assert qa2Null && qa2 == null || qa2 != null

        where:
        "Data inputs are: "
        tokenName               | qa1Null | qa2Null
        NOACCESSTOKEN           | true    | true
        "searchNamespacesToken" | false   | true
        ALLACCESSTOKEN          | false   | false
    }

    @Unroll
    def "Verify search with SAC and token #tokenName yields the same number of results as restricted search"() {
        when:
        "Searching for categories ${categories} in namespace ${namespace} with basic auth"
        def restrictedQuery = SSOC.RawSearchRequest.newBuilder()
                .addAllCategories(categories)
                .setQuery("Cluster:${DEFAULT_CLUSTER_NAME}+Namespace:${namespace}")
                .build()
        def restrictedWithBasicAuthCount = SearchService.search(restrictedQuery).resultsCount

        and:
        "Searching for categories ${categories} in namespace ${namespace} with a token with all access"
        useToken(ALLACCESSTOKEN)
        def restrictedWithAllAccessCount = SearchService.search(restrictedQuery).resultsCount

        and:
        "Searching for categories ${categories} in all NS with token ${tokenName} restricted to namespace ${namespace}"
        useToken(tokenName)
        def unrestrictedQuery = SSOC.RawSearchRequest.newBuilder()
                .addAllCategories(categories)
                .setQuery("Cluster:${DEFAULT_CLUSTER_NAME}")
                .build()
        def unrestrictedWithSACCount = SearchService.search(unrestrictedQuery).resultsCount

        then:
        "The number of results should be the same for everything"

        log.info "With basic auth + restricted query: ${restrictedWithBasicAuthCount}"
        log.info "With all access token + restricted query: ${restrictedWithAllAccessCount}"
        log.info "With SAC restricted token + unrestricted query: ${unrestrictedWithSACCount}"

        assert restrictedWithBasicAuthCount == restrictedWithAllAccessCount
        assert restrictedWithAllAccessCount == unrestrictedWithSACCount

        where:
        "Data inputs are: "
        tokenName                          | namespace      | categories
        NOACCESSTOKEN                      | "non_existent" | [SSOC.SearchCategory.NAMESPACES,
                                                               SSOC.SearchCategory.IMAGES,
                                                               SSOC.SearchCategory.DEPLOYMENTS]
        "kubeSystemDeploymentsImagesToken" | "kube-system"  | [SSOC.SearchCategory.IMAGES]
        "searchNamespacesToken"            | NAMESPACE_QA1  | [SSOC.SearchCategory.NAMESPACES]
        "searchDeploymentsToken"           | NAMESPACE_QA1  | [SSOC.SearchCategory.DEPLOYMENTS]
        "searchDeploymentsImagesToken"     | NAMESPACE_QA1  | [SSOC.SearchCategory.IMAGES]
    }

    def "Verify that SAC has the same effect as query restriction for network flows"() {
        when:
        "Obtaining the network graph for the StackRox namespace with all access"
        def networkGraphWithAllAccess = NetworkGraphService.getNetworkGraph(null, "Namespace:stackrox")
        def allAccessFlows = NetworkGraphUtil.flowStrings(networkGraphWithAllAccess)
        allAccessFlows.removeAll(UNSTABLE_FLOWS)
        log.info "${allAccessFlows}"

        def allAccessFlowsWithoutNeighbors = allAccessFlows.findAll {
            it.matches("(stackrox/.*|INTERNET) -> (stackrox/.*|INTERNET)")
        }
        log.info "${allAccessFlowsWithoutNeighbors}"

        and:
        "Obtaining the network graph for the StackRox namespace with a SAC restricted token"
        useToken("stackroxNetFlowsToken")
        def networkGraphWithSAC = NetworkGraphService.getNetworkGraph(null, "Namespace:stackrox")
        def sacFlows = NetworkGraphUtil.flowStrings(networkGraphWithSAC)
        sacFlows.removeAll(UNSTABLE_FLOWS)
        log.info "${sacFlows}"

        and:
        "Obtaining the network graph for the StackRox namespace with a SAC restricted token and no query"
        def networkGraphWithSACNoQuery = NetworkGraphService.getNetworkGraph()
        def sacFlowsNoQuery = NetworkGraphUtil.flowStrings(networkGraphWithSACNoQuery)
        sacFlowsNoQuery.removeAll(UNSTABLE_FLOWS)
        log.info "${sacFlowsNoQuery}"

        then:
        "Query-restricted and non-restricted flows should be equal under SAC"
        assert sacFlows == sacFlowsNoQuery

        and:
        "The flows should be equal to the flows obtained with all access after removing masked endpoints"
        Set<String> sacFlowsFiltered = sacFlows.findAll { !it.contains("masked deployment") }
        Set<String> sacFlowsNoQueryFiltered = sacFlowsNoQuery.findAll { !it.contains("masked deployment") }

        assert allAccessFlowsWithoutNeighbors == sacFlowsFiltered
        assert allAccessFlowsWithoutNeighbors == sacFlowsNoQueryFiltered

        and:
        "The flows obtained with SAC should contain some masked deployments"
        assert sacFlowsFiltered.size() < sacFlows.size()
        assert sacFlowsNoQueryFiltered.size() < sacFlowsNoQuery.size()

        and:
        "The masked deployments should be external to stackrox namespace"
        assert sacFlows.intersect(sacFlowsFiltered) ==
                allAccessFlows.intersect(allAccessFlowsWithoutNeighbors)
        assert sacFlowsNoQuery.intersect(sacFlowsNoQueryFiltered) ==
                allAccessFlows.intersect(allAccessFlowsWithoutNeighbors)
    }

    def "test role aggregation should not combine permissions sets"() {
        when:
        useToken("aggregatedToken")

        then:
        def result = DeploymentService.listDeployments()
        assert result.find { it.name == DEPLOYMENT_QA2.name }
        assert !result.find { it.name == DEPLOYMENT_QA1.name }
    }

    @Unroll
    def "Verify using the #tokenName token gets #numResults results when retrieving the current cluster"() {
        when:
        useToken(tokenName)
        def clusters = ClusterService.getClusters()
        def count = 0
        clusters.forEach {
            cluster ->
                if (cluster.getName() == DEFAULT_CLUSTER_NAME) { count++ }
        }

        then:
        "The number of valid results should be the expected one"
        assert count == numResults

        where:
        "Data inputs are: "
        tokenName         | numResults
        NOACCESSTOKEN     | 0
        "getClusterToken" | 1
        ALLACCESSTOKEN    | 1
    }

    private static List<DeploymentOuterClass.ListDeployment> listDeployments() {
        return Services.getDeployments(
                SSOC.RawQuery.newBuilder().setQuery("Namespace:" + NAMESPACE_QA1).build()
        ) + Services.getDeployments(
                SSOC.RawQuery.newBuilder().setQuery("Namespace:" + NAMESPACE_QA2).build()
        )
    }
}
