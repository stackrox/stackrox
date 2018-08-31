import io.grpc.netty.GrpcSslContexts
import io.grpc.netty.NegotiationType
import io.grpc.netty.NettyChannelBuilder
import io.netty.handler.ssl.SslContext
import io.netty.handler.ssl.util.InsecureTrustManagerFactory
import stackrox.generated.AlertServiceGrpc
import stackrox.generated.AlertServiceOuterClass.ListAlert
import stackrox.generated.DeploymentServiceGrpc
import stackrox.generated.ImageIntegrationServiceGrpc
import stackrox.generated.ImageIntegrationServiceOuterClass
import stackrox.generated.ImageIntegrationServiceOuterClass.ImageIntegration
import stackrox.generated.PolicyServiceGrpc
import stackrox.generated.PolicyServiceOuterClass.ListPolicy
import stackrox.generated.PolicyServiceOuterClass.Policy
import stackrox.generated.SearchServiceGrpc
import stackrox.generated.SearchServiceOuterClass.RawQuery
import stackrox.generated.AlertServiceOuterClass.Alert
import stackrox.generated.AlertServiceOuterClass.ListAlertsRequest
import stackrox.generated.DeploymentServiceOuterClass.ListDeployment
import stackrox.generated.DeploymentServiceOuterClass.Deployment
import stackrox.generated.Common.ResourceByID
import stackrox.generated.SearchServiceOuterClass
import stackrox.generated.SearchServiceOuterClass.SearchResult

class Services {

    static getChannel() {
        SslContext sslContext = GrpcSslContexts
                .forClient()
                .trustManager(InsecureTrustManagerFactory.INSTANCE)
                .build()

        int port = Integer.parseInt(System.getenv("PORT"))

        def channel = NettyChannelBuilder
                .forAddress(System.getenv("HOSTNAME"), port)
                .negotiationType(NegotiationType.TLS)
                .sslContext(sslContext)
                .build()
        return channel
    }

    static ResourceByID getResourceByID(String id) {
        return ResourceByID.newBuilder().setId(id).build()
    }

    static getIntegrationClient() {
        return ImageIntegrationServiceGrpc.newBlockingStub(getChannel())
    }

    static getPolicyClient() {
        return PolicyServiceGrpc.newBlockingStub(getChannel())
    }

    static getAlertClient() {
        return AlertServiceGrpc.newBlockingStub(getChannel())
    }

    static getDeploymentClient() {
        return DeploymentServiceGrpc.newBlockingStub(getChannel())
    }

    static getSearchServiceClient() {
        return SearchServiceGrpc.newBlockingStub(getChannel())
    }

    static List<ListPolicy> getPolicies(RawQuery query = RawQuery.newBuilder().build()) {
        return getPolicyClient().listPolicies(query).policiesList
    }

    static Policy getPolicy(String id) {
        return getPolicyClient().getPolicy(getResourceByID(id))
    }

    static List<ListAlert> getViolations(ListAlertsRequest request = ListAlertsRequest.newBuilder().build()) {
        return getAlertClient().listAlerts(request).alertsList
    }

    static Alert getViolaton(String id) {
        return getAlertClient().getAlert(getResourceByID(id))
    }

    static List<ListDeployment> getDeployments(RawQuery query = RawQuery.newBuilder().build()) {
        return getDeploymentClient().listDeployments(query).deploymentsList
    }

    static Deployment getDeployment(String id) {
        return getDeploymentClient().getDeployment(getResourceByID(id))
    }

    static List<SearchResult> getSearchResults(SearchServiceOuterClass.RawSearchRequest query) {
        return getSearchServiceClient().search(query).resultsList
    }

    static int getSearch(String value, SearchServiceOuterClass.SearchCategory category) {
        def result = getSearchResults(SearchServiceOuterClass.RawSearchRequest.newBuilder()
                .addCategories(category)
                .setQuery(value)
                .build())
        return result.size()
    }

    static waitForViolation(String deploymentName, String policyName, int timeoutSeconds) {
        int intervalSeconds = 1
        int waitTime
        for (waitTime = 0; waitTime < timeoutSeconds / intervalSeconds; waitTime++) {
            def violations = getViolations(ListAlertsRequest.newBuilder()
                    .setQuery("Deployment:${deploymentName}+Policy:${policyName}").build())
            if (violations.size() > 0) {
                println "violation size is: " + violations.size()
                println policyName + " triggered after waiting " + waitTime * intervalSeconds + " seconds"
                return true
            }
            sleep(intervalSeconds * 1000)
        }

        println "Failed to trigger " + policyName + " after waiting " + waitTime * intervalSeconds + " seconds"
        return false
    }

    static String addGenericDockerRegistry() {
        return getIntegrationClient().postImageIntegration(
            ImageIntegration.newBuilder()
                .setName("dockerhub")
                .addCategories(ImageIntegrationServiceOuterClass.ImageIntegrationCategory.REGISTRY)
                .setType("docker")
                .setDocker(
                    ImageIntegrationServiceOuterClass.DockerConfig.newBuilder()
                    .setUsername("")
                    .setPassword("")
                    .setEndpoint("registry-1.docker.io")
                    .setInsecure(false)
                    .build()
                )
                .build()
        )
        .getId()
    }

    static deleteGenericDockerRegistry(String gdrId) {
        getIntegrationClient().deleteImageIntegration(
                ResourceByID.newBuilder()
                        .setId(gdrId)
                        .build()
        )
    }

    static String addDockerTrustedRegistry() {
        return getIntegrationClient().postImageIntegration(
            ImageIntegration.newBuilder()
                .setName("dtr")
                .setType("dtr")
                .addCategories(ImageIntegrationServiceOuterClass.ImageIntegrationCategory.REGISTRY)
                .addCategories(ImageIntegrationServiceOuterClass.ImageIntegrationCategory.SCANNER)
                .setDtr(ImageIntegrationServiceOuterClass.DTRConfig.newBuilder()
                    .setEndpoint("https://apollo-dtr.rox.systems/")
                    .setUsername("qa")
                    .setPassword("W3g9xOPKyLTkBBMj")
                    .setInsecure(false)
                    .build()
                )
            .build()
        )
        .getId()
    }

    static deleteDockerTrustedRegistry(String dtrId) {
        getIntegrationClient().deleteImageIntegration(
                ResourceByID.newBuilder()
                        .setId(dtrId)
                        .build()
        )
    }

    static String addClairifyScanner(String clairifyEndpoint) {
        return getIntegrationClient().postImageIntegration(
            ImageIntegration.newBuilder()
                .setName("clairify")
                .setType("clairify")
                .addCategories(ImageIntegrationServiceOuterClass.ImageIntegrationCategory.SCANNER)
                .setClairify(ImageIntegrationServiceOuterClass.ClairifyConfig.newBuilder()
                    .setEndpoint(clairifyEndpoint)
                    .build()
                )
            .build()
        )
        .getId()
    }

    static deleteClairifyScanner(String clairifyId) {
        getIntegrationClient().deleteImageIntegration(
                ResourceByID.newBuilder()
                        .setId(clairifyId)
                        .build()
        )
    }
}
