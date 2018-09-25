import io.grpc.StatusRuntimeException
import io.grpc.netty.GrpcSslContexts
import io.grpc.netty.NegotiationType
import io.grpc.netty.NettyChannelBuilder
import io.netty.handler.ssl.SslContext
import io.netty.handler.ssl.util.InsecureTrustManagerFactory
import stackrox.generated.AlertServiceGrpc
import stackrox.generated.AlertServiceOuterClass.ListAlert
import stackrox.generated.ClusterService
import stackrox.generated.ClustersServiceGrpc
import stackrox.generated.DeploymentServiceGrpc
import stackrox.generated.DetectionServiceGrpc
import stackrox.generated.EnforcementServiceGrpc
import stackrox.generated.EnforcementServiceOuterClass
import stackrox.generated.ImageIntegrationServiceGrpc
import stackrox.generated.ImageIntegrationServiceOuterClass
import stackrox.generated.ImageIntegrationServiceOuterClass.ImageIntegration
import stackrox.generated.ImageServiceOuterClass
import stackrox.generated.ImageServiceOuterClass.Image
import stackrox.generated.NotifierServiceGrpc
import stackrox.generated.NotifierServiceOuterClass
import stackrox.generated.PolicyServiceGrpc
import stackrox.generated.PolicyServiceOuterClass.EnforcementAction
import stackrox.generated.PolicyServiceOuterClass.LifecycleStage
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
import stackrox.generated.SensorEventServiceOuterClass
import v1.SecretServiceGrpc
import v1.NetworkPolicyServiceGrpc
import v1.NetworkPolicyServiceOuterClass

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

    static getDetectionClient() {
        return DetectionServiceGrpc.newBlockingStub(getChannel())
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

    static getSecretServiceClient() {
        return SecretServiceGrpc.newBlockingStub(getChannel())
    }

    static getNetworkPolicyClient() {
        return NetworkPolicyServiceGrpc.newBlockingStub(getChannel())
    }

    static getClusterServiceClient() {
        return ClustersServiceGrpc.newBlockingStub(getChannel())
    }

    static getNotifierClient() {
        return NotifierServiceGrpc.newBlockingStub(getChannel())
    }

    static getEnforcementClient() {
        return EnforcementServiceGrpc.newBlockingStub(getChannel())
    }

    static List<ListPolicy> getPolicies(RawQuery query = RawQuery.newBuilder().build()) {
        return getPolicyClient().listPolicies(query).policiesList
    }

    static Policy getPolicyByName(String policyName) {
        return getPolicy(
                getPolicies().find { it.name == policyName }.id
        )
    }

    static Policy getPolicy(String id) {
        return getPolicyClient().getPolicy(getResourceByID(id))
    }

    static deletePolicy(String policyID) {
        getPolicyClient().deletePolicy(
            ResourceByID.newBuilder()
                .setId(policyID)
                .build()
        )
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

    static SearchServiceOuterClass.SearchResponse getSearchResponse(
       String query, List<SearchServiceOuterClass.SearchCategory> categories) {
        def rawSearchRequest = SearchServiceOuterClass.RawSearchRequest.newBuilder()
            .addAllCategories(categories)
            .setQuery(query)
            .build()
        return getSearchServiceClient().search(rawSearchRequest)
    }

    static String getSecret(String id) {
        int intervalSeconds = 1
        int waitTime
        for (waitTime = 0; waitTime < 50000 / intervalSeconds; waitTime++) {
            def sec= getSecretServiceClient().getSecret(ResourceByID.newBuilder().setId(id).build())
            if (sec != null) {
                return sec.id
            }
            sleep(intervalSeconds * 1000)
       }
        println "Failed to add secret " + id + " after waiting " + waitTime * intervalSeconds + " seconds"
        return null
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

    static requestBuildImageScan(String registry, String remote, String tag) {
        return getDetectionClient().detectBuildTime(
                Image.newBuilder()
                        .setName(ImageServiceOuterClass.ImageName.newBuilder()
                            .setRegistry(registry)
                            .setRemote(remote)
                            .setTag(tag)
                            .build()
                        )
                        .build()
        )
    }

    static updatePolicyLifecycleStage(String policyName, LifecycleStage stage) {
        Policy policyMeta = getPolicyByName(policyName)
        def policyDef = Policy.newBuilder(policyMeta)
                .setLifecycleStage(stage)
                .build()
        try {
            getPolicyClient().putPolicy(policyDef)
        } catch (Exception e) {
            return ""
        }
        println "Updated lifecycleStage of '${policyName}' to ${stage}"
        return policyMeta.lifecycleStage
    }

    static updatePolicyEnforcement(String policyName, EnforcementAction enforcementAction) {
        Policy policyMeta = getPolicyByName(policyName)
        def policyDef = Policy.newBuilder(policyMeta)
                .setEnforcement(enforcementAction)
                .build()
        try {
            getPolicyClient().putPolicy(policyDef)
        } catch (Exception e) {
            return ""
        }
        println "Updated enforcement of '${policyName}' to ${enforcementAction}"
        return policyMeta.enforcement
    }

    static getClusterId(String name = "remote") {
        ClusterService.Cluster cluster = getClusterServiceClient().getClusters().clustersList.find { it.name == name }

        if (cluster == null) {
            def firstClusterName = getClusterServiceClient().getClusters().clustersList.get(0).name
            println "Could not find id for cluster name ${name}"
            println "Will return id for first cluster: ${firstClusterName}"
        }

        return getClusterServiceClient().getClusters().clustersList.get(0).id
    }

    static submitNetworkGraphSimulation(String yaml) {
        println "Generating simulation using YAML:"
        println yaml
        try {
            return getNetworkPolicyClient().getNetworkGraph(
                    NetworkPolicyServiceOuterClass.GetNetworkGraphRequest.newBuilder()
                            .setClusterId(getClusterId())
                            .setSimulationYaml(yaml)
                            .build()
            )
        } catch (Exception e) {
            println e.toString()
        }
    }

    static getNetworkGraph() {
        try {
            return getNetworkPolicyClient().getNetworkGraph(
                    NetworkPolicyServiceOuterClass.GetNetworkGraphRequest.newBuilder()
                            .setClusterId(getClusterId())
                            .build()
            )
        } catch (Exception e) {
            println e.toString()
        }
    }

    static addSlackNotifier(String name) {
        try {
            return getNotifierClient().postNotifier(
                    NotifierServiceOuterClass.Notifier.newBuilder()
                            .setType("slack")
                            .setName(name)
                            .setLabelKey("#slack-test")
                            .setLabelDefault(
                                "https://hooks.slack.com/services/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"
                            )
                            .setEnabled(true)
                            .setUiEndpoint("https://" +
                                    System.getenv("HOSTNAME") +
                                    ":" + System.getenv("PORT"))
                    .build()
            )
        } catch (Exception e) {
            println e.toString()
        }
    }

    static addJiraNotifier(String name) {
        try {
            return getNotifierClient().postNotifier(
                    NotifierServiceOuterClass.Notifier.newBuilder()
                            .setType("jira")
                            .setName(name)
                            .setLabelKey("AJIT")
                            .setLabelDefault("AJIT")
                            .setEnabled(true)
                            .setUiEndpoint("https://" +
                                    System.getenv("HOSTNAME") +
                                    ":" + System.getenv("PORT"))
                            .setJira(NotifierServiceOuterClass.Jira.newBuilder()
                                    .setUsername("k+automation@stackrox.com")
                                    .setPassword("D7wU97n9CFYuesHt")
                                    .setUrl("https://stack-rox.atlassian.net")
                                    .setIssueType("Task")
                            )
                            .build()
            )
        } catch (Exception e) {
            println e.toString()
        }
    }

    static addEmailNotifier(String name, Boolean disableTLS = false, startTLS = false, Integer port = null) {
        try {
            NotifierServiceOuterClass.Notifier.Builder builder =
                    NotifierServiceOuterClass.Notifier.newBuilder()
                            .setEmail(NotifierServiceOuterClass.Email.newBuilder())
            builder
                    .setType("email")
                    .setName(name)
                    .setLabelKey("mailgun")
                    .setLabelDefault("to@example.com")
                    .setEnabled(true)
                    .setUiEndpoint("https://" +
                            System.getenv("HOSTNAME") +
                            ":" + System.getenv("PORT"))
                    .setEmail(builder.getEmailBuilder()
                            .setUsername("postmaster@sandboxa91803d176f944229a601fc109e20250.mailgun.org")
                            .setPassword("5da76fea807449ea105a77d4fa05420f-7bbbcb78-b8136e8b")
                            .setSender("from@example.com")
                            .setDisableTLS(disableTLS)
                            .setUseSTARTTLS(startTLS)
                    )
            port == null ?
                    builder.getEmailBuilder().setServer("smtp.mailgun.org") :
                    builder.getEmailBuilder().setServer("smtp.mailgun.org:" + port)
            return getNotifierClient().postNotifier(builder.build())
        } catch (Exception e) {
            println e.toString()
        }
    }

    static testNotifier(NotifierServiceOuterClass.Notifier notifier) {
        try {
            return getNotifierClient().testNotifier(notifier)
        } catch (Exception e) {
            println e.toString()
            return e
        }
    }

    static deleteNotifier(String id) {
        try {
            getNotifierClient().deleteNotifier(
                    NotifierServiceOuterClass.DeleteNotifierRequest.newBuilder()
                            .setId(id)
                            .setForce(true)
                            .build()
            )
        } catch (Exception e) {
            println e.toString()
        }
    }

    static sendSimulationNotification(String notifierId, String yaml, String clusterId = getClusterId()) {
        try {
            NetworkPolicyServiceOuterClass.SendNetworkPolicyYamlRequest.Builder request =
                    NetworkPolicyServiceOuterClass.SendNetworkPolicyYamlRequest.newBuilder()
            notifierId == null ?: request.setNotifierId(notifierId)
            clusterId == null ?: request.setClusterId(clusterId)
            yaml == null ?: request.setYaml(yaml)
            return getNetworkPolicyClient().sendNetworkPolicyYAML(request.build())
        } catch (Exception e) {
            println e.toString()
            assert e instanceof StatusRuntimeException
        }
    }

    static applyEnforcement(SensorEventServiceOuterClass.SensorEnforcement.Builder builder) {
        try {
            return getEnforcementClient().applyEnforcement(
                    EnforcementServiceOuterClass.EnforcementRequest.newBuilder()
                            .setClusterId(getClusterId())
                            .setEnforcement(builder)
                    .build()
            )
        } catch (Exception e) {
            println e.toString()
        }
    }

    static applyKillEnforcement(String podId, String namespace, String containerId) {
        SensorEventServiceOuterClass.SensorEnforcement.Builder killEnforcemetBuilder =
                SensorEventServiceOuterClass.SensorEnforcement.newBuilder()
                        .setEnforcement(EnforcementAction.KILL_POD_ENFORCEMENT)
                        .setContainerInstance(SensorEventServiceOuterClass.ContainerInstanceEnforcement.newBuilder()
                                .setContainerInstanceId(containerId)
                                .setPodId(podId)
                                .setNamespace(namespace)
                        )
        return applyEnforcement(killEnforcemetBuilder)
    }

    static applyScaleDownEnforcement(objects.Deployment deployment) {
        SensorEventServiceOuterClass.SensorEnforcement.Builder scaleDownEnforcementBuilder =
                SensorEventServiceOuterClass.SensorEnforcement.newBuilder()
                        .setEnforcement(EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT)
                        .setDeployment(SensorEventServiceOuterClass.DeploymentEnforcement.newBuilder()
                                .setDeploymentId(deployment.deploymentUid)
                                .setDeploymentName(deployment.name)
                                .setDeploymentType("Deployment")
                                .setNamespace(deployment.namespace)
                                .setAlertId("qa_automation")
                        )
        return applyEnforcement(scaleDownEnforcementBuilder)
    }

    static applyNodeConstraintEnforcement(objects.Deployment deployment) {
        SensorEventServiceOuterClass.SensorEnforcement.Builder scaleDownEnforcementBuilder =
                SensorEventServiceOuterClass.SensorEnforcement.newBuilder()
                        .setEnforcement(EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT)
                        .setDeployment(SensorEventServiceOuterClass.DeploymentEnforcement.newBuilder()
                        .setDeploymentId(deployment.deploymentUid)
                        .setDeploymentName(deployment.name)
                        .setDeploymentType("Deployment")
                        .setNamespace(deployment.namespace)
                        .setAlertId("qa_automation")
                )
        return applyEnforcement(scaleDownEnforcementBuilder)
    }
}
