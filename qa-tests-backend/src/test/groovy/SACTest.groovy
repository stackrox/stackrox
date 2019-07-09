import static Services.waitForViolation

import io.stackrox.proto.api.v1.ApiTokenService.GenerateTokenResponse
import io.stackrox.proto.api.v1.NamespaceServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass as SSOC
import objects.Deployment
import services.AlertService
import services.ClusterService
import services.DeploymentService
import services.ImageService
import services.NamespaceService
import services.SACService
import services.ApiTokenService
import services.BaseService
import services.SearchService
import services.SecretService
import services.SummaryService
import spock.lang.Shared
import spock.lang.Unroll

class SACTest extends BaseSpecification {
    static final private String DEPLOYMENTNGINX_NAMESPACE_QA1 = "sac-deploymentnginx-qa1"
    static final private String DEPLOYMENTNGINX_NAMESPACE_QA2 = "sac-deploymentnginx-qa2"
    static final private String NONE = "None"
    static final private String SECRETNAME = "sac-secret"
    static final private Deployment DEPLOYMENT_QA1 = new Deployment()
            .setName(DEPLOYMENTNGINX_NAMESPACE_QA1)
            .setImage("nginx:1.7.9")
            .addPort(22, "TCP")
            .addAnnotation("test", "annotation")
            .setEnv(["CLUSTER_NAME": "main"])
            .setNamespace("test-qa1")
            .addLabel("app", "test")
    static final private Deployment DEPLOYMENT_QA2 = new Deployment()
            .setName(DEPLOYMENTNGINX_NAMESPACE_QA2)
            .setImage("nginx:1.7.9")
            .addPort(22, "TCP")
            .addAnnotation("test", "annotation")
            .setEnv(["CLUSTER_NAME": "main"])
            .setNamespace("test-qa2")
            .addLabel("app", "test")

    static final private List<Deployment> DEPLOYMENTS = [DEPLOYMENT_QA1, DEPLOYMENT_QA2,]
    @Shared
    private String pluginConfigID

    def setupSpec() {
        BaseService.useBasicAuth()
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
        }
        def response = SACService.addAuthPlugin()
        pluginConfigID = response.getId()
        println response.toString()
        // Make sure each deployment has caused at least one alert
        assert waitForViolation(DEPLOYMENT_QA1.name, "Secure Shell (ssh) Port Exposed", 60)
        assert waitForViolation(DEPLOYMENT_QA2.name, "Secure Shell (ssh) Port Exposed", 60)
    }

    def cleanupSpec() {
        BaseService.useBasicAuth()
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
        if (pluginConfigID != null) {
            SACService.deleteAuthPluginConfig(pluginConfigID)
        }
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

    static useToken(String tokenName) {
        BaseService.useBasicAuth()
        GenerateTokenResponse token = ApiTokenService.generateToken(tokenName, NONE)
        BaseService.useApiToken(token.token)
    }

    static getAllQuery() {
        def queryString = "Cluster: " + ClusterService.getCluster().name
        return SSOC.RawSearchRequest.newBuilder()
                .setQuery(queryString)
                .build()
    }

    static getSpecificQuery(SSOC.SearchCategory category) {
        def queryString = "Namespace: " + [DEPLOYMENT_QA1.namespace, DEPLOYMENT_QA2.namespace].join(", ")
        return SSOC.RawSearchRequest.newBuilder()
                .addCategories(category)
                .setQuery(queryString)
                .build()
    }

    def createSecret(String namespace) {
        String secID = orchestrator.createSecret(SECRETNAME, namespace)
        SecretService.waitForSecret(secID, 10)
    }

    def deleteSecret(String namespace) {
        orchestrator.deleteSecret(SECRETNAME, namespace)
    }

    @Unroll
    def "Verify that only namespace #sacResource is visible when using SAC"() {
        when:
        "Create test API token with a built-in role"
        useToken("deployments-access-token")
        then:
        "Call API and verify data returned is within scoped access"
        def result = DeploymentService.listDeployments()
        println result.toString()
        assert result.size() == 1
        def resourceNotAllowed = result.find { it.namespace != sacResource }
        assert resourceNotAllowed == null
        cleanup:
        BaseService.useBasicAuth()
        where:
        "Data inputs are: "
        sacResource | _
        "test-qa2"  | _
    }

    def "Verify GetSummaryCounts using a token without access receives no results"() {
        when:
        "GetSummaryCounts is called using a token without access"
        createSecret(DEPLOYMENT_QA1.namespace)
        useToken("noAccess")
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
        assert result.getNumSecrets() == 1
        assert result.getNumNodes() == 0
        assert result.getNumClusters() == 0
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
        useToken("allAccessToken")
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

    def "Verify ListSecrets using a token without access receives no results"() {
        when:
        "ListSecrets is called using a token without view access to Secrets"
        BaseService.useBasicAuth()
        createSecret(DEPLOYMENT_QA1.namespace)
        useToken("noAccess")
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
        BaseService.useBasicAuth()
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
    def "Verify Search on #category resources using the given token returns #numResults results"() {
        when:
        "A search is performed using the given token"
        def query = getAllQuery()
        useToken(tokenName)
        def result = SearchService.search(query)
        then:
        "Verify the specified number of results are returned"
        assert result.resultsCount == numResults
        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
        where:
        "Data inputs are: "
        tokenName                | category                        | numResults
        "noAccess"               | SSOC.SearchCategory.DEPLOYMENTS | 0
        "noAccess"               | SSOC.SearchCategory.ALERTS      | 0
        "noAccess"               | SSOC.SearchCategory.IMAGES      | 0
        "searchDeploymentsToken" | SSOC.SearchCategory.DEPLOYMENTS | 1
        "searchImagesToken"      | SSOC.SearchCategory.IMAGES      | 1
    }

    @Unroll
    def "Verify Search on #category resources using the #tokenName token returns >= #minReturned results"() {
        when:
        "A search is performed using the given token"
        def query = getAllQuery()
        useToken(tokenName)
        def result = SearchService.search(query)
        then:
        "Verify >= the specified number of results are returned"
        assert result.resultsCount >= minReturned
        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
        where:
        "Data inputs are: "
        tokenName           | category                   | minReturned
        "searchAlertsToken" | SSOC.SearchCategory.ALERTS | 1
    }

    def "Verify Search using the allAccessToken returns results for all search categories"() {
        when:
        "A search is performed using the allAccessToken"
        createSecret(DEPLOYMENT_QA1.namespace)
        def query = getAllQuery()
        useToken("allAccessToken")
        def result = SearchService.search(query)
        then:
        "Verify something was returned for every search category"
        for (SSOC.SearchResponse.Count numResults : result.countsList) {
            // Policies are globally scoped so our cluster-scoped query won't return any
            if (numResults.category == SSOC.SearchCategory.POLICIES) {
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
        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
        where:
        "Data inputs are: "
        tokenName                | category                        | numResults
        "noAccess"               | SSOC.SearchCategory.DEPLOYMENTS | 0
        "noAccess"               | SSOC.SearchCategory.ALERTS      | 0
        "noAccess"               | SSOC.SearchCategory.IMAGES      | 0
        "searchDeploymentsToken" | SSOC.SearchCategory.DEPLOYMENTS | 1
        "searchImagesToken"      | SSOC.SearchCategory.IMAGES      | 1
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
        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
        where:
        "Data inputs are: "
        tokenName           | category                        | minReturned
        "allAccessToken"    | SSOC.SearchCategory.DEPLOYMENTS | 2
        "allAccessToken"    | SSOC.SearchCategory.ALERTS      | 1
        "allAccessToken"    | SSOC.SearchCategory.IMAGES      | 1
        "searchAlertsToken" | SSOC.SearchCategory.ALERTS      | 1
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
        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
        where:
        "Data iputs are: "
        tokenName                | numReturned | resultCountFunc          | service
        "noAccess"               | 0           | this.&getDeploymentCount | "Deployment"
        "searchDeploymentsToken" | 1           | this.&getDeploymentCount | "Deployment"
        "noAccess"               | 0           | this.&getAlertCount      | "Alert"
        "noAccess"               | 0           | this.&getImageCount      | "Image"
        "noAccess"               | 0           | this.&getNamespaceCount  | "Namespace"
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
        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
        where:
        "Data iputs are: "
        tokenName           | minNumReturned | resultCountFunc          | service
        "allAccessToken"    | 1              | this.&getAlertCount      | "Alert"
        "searchAlertsToken" | 1              | this.&getAlertCount      | "Alert"
        "allAccessToken"    | 1              | this.&getImageCount      | "Image"
        "allAccessToken"    | 2              | this.&getDeploymentCount | "Deployment"
        "allAccessToken"    | 2              | this.&getNamespaceCount  | "Namespace"
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
        cleanup:
        "Cleanup"
        BaseService.useBasicAuth()
        where:
        "Data inputs are: "
        tokenName               | qa1Null | qa2Null
        "noAccess"              | true    | true
        "searchNamespacesToken" | false   | true
        "allAccessToken"        | false   | false
    }
}
