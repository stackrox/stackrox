import io.grpc.StatusRuntimeException
import orchestratormanager.OrchestratorType
import services.BaseService
import services.ClusterService
import stackrox.generated.AlertServiceGrpc
import stackrox.generated.AlertServiceOuterClass
import stackrox.generated.AlertServiceOuterClass.GetAlertsCountsRequest
import stackrox.generated.AlertServiceOuterClass.GetAlertsCountsResponse
import stackrox.generated.AlertServiceOuterClass.GetAlertsGroupResponse
import stackrox.generated.AlertServiceOuterClass.GetAlertTimeseriesResponse
import stackrox.generated.AlertServiceOuterClass.ListAlert
import stackrox.generated.Common
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
import stackrox.generated.PolicyServiceOuterClass
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
import stackrox.generated.SensorEventIservice
import v1.NetworkPolicyServiceGrpc
import v1.SecretServiceGrpc
import v1.NetworkPolicyServiceOuterClass

class Services extends BaseService {

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

    static GetAlertsCountsResponse getAlertCounts(
            GetAlertsCountsRequest request = GetAlertsCountsRequest.newBuilder().build()) {
        return getAlertClient().getAlertsCounts(request)
      }

    static GetAlertsGroupResponse getAlertGroups(ListAlertsRequest request = ListAlertsRequest.newBuilder().build()) {
        return getAlertClient().getAlertsGroup(request)
      }

    static GetAlertTimeseriesResponse getAlertTimeseries(
            ListAlertsRequest request = ListAlertsRequest.newBuilder().build()) {
        return getAlertClient().getAlertTimeseries(request)
      }

    static int getAlertEnforcementCount(String deploymentName, String policyName) {
        def violations = getViolations(ListAlertsRequest.newBuilder()
                .setQuery("Deployment:${deploymentName}+Policy:${policyName}").build())
        return violations.get(0)?.enforcementCount
    }

    static Alert getViolation(String id) {
        return getAlertClient().getAlert(getResourceByID(id))
    }

    static resolveAlert(String alertID) {
        return getAlertClient().resolveAlert(
            AlertServiceOuterClass.ResolveAlertRequest.newBuilder().setId(alertID).build())
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
            def sec = getSecretServiceClient().getSecret(ResourceByID.newBuilder().setId(id).build())
            if (sec != null) {
                return sec.id
                  }
            sleep(intervalSeconds * 1000)
            }
        println "Failed to add secret " + id + " after waiting " + waitTime * intervalSeconds + " seconds"
        return null
      }

    static waitForViolation(String deploymentName, String policyName, int timeoutSeconds = 30) {
        def violations = getViolationsWithTimeout(deploymentName, policyName, timeoutSeconds)
        return violations != null && violations.size() > 0
      }

    private static getViolationsHelper(String query, String policyName, int timeoutSeconds) {
        int intervalSeconds = 1
        int waitTime
        for (waitTime = 0; waitTime < timeoutSeconds / intervalSeconds; waitTime++) {
            def violations = getViolations(ListAlertsRequest.newBuilder()
                    .setQuery(query).build())
            if (violations.size() > 0) {
                println "violation size is: " + violations.size()
                println policyName + " triggered after waiting " + waitTime * intervalSeconds + " seconds"
                return violations
            }
            sleep(intervalSeconds * 1000)
        }

        println "Failed to trigger " + policyName + " after waiting " + waitTime * intervalSeconds + " seconds"
        return []
    }

    static getViolationsWithTimeout(String deploymentName, String policyName, int timeoutSeconds) {
        return getViolationsHelper("Deployment:${deploymentName}+Policy:${policyName}", policyName, timeoutSeconds)
    }

    static getViolationsByDeploymentID(String deploymentID, String policyName, int timeoutSeconds) {
        return getViolationsHelper("Deployment Id:${deploymentID}+Policy:${policyName}", policyName, timeoutSeconds)
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

    static updatePolicy(Policy policyDef) {
        try {
            getPolicyClient().putPolicy(policyDef)
        } catch (Exception e) {
            println e.toString()
        }
    }

    static updatePolicyToWhitelistDeployment(String policyName, objects.Deployment deployment) {
        Policy policyMeta = getPolicyByName(policyName)

        def policyDef = Policy.
            newBuilder(policyMeta).
            addWhitelists(PolicyServiceOuterClass.Whitelist.newBuilder().
                setDeployment(PolicyServiceOuterClass.Whitelist.Deployment.newBuilder().
                    setName(deployment.getName()).
                    setScope(Common.Scope.newBuilder().
                        setNamespace(deployment.getNamespace())
                    ).
                    build()).
                build()).
            build()

        try {
            getPolicyClient().putPolicy(policyDef)
        } catch (Exception e) {
            println e.toString()
        }
        println "Updated ${policyName} to whitelist ${deployment.getName()}"
        return policyMeta
    }

    static updatePolicyLifecycleStage(String policyName, List<LifecycleStage> stages) {
        Policy policyMeta = getPolicyByName(policyName)

        def builder = Policy.newBuilder(policyMeta).clearLifecycleStages()
        if (stages != null && !stages.isEmpty()) {
            builder.addAllLifecycleStages(stages)
        }
        def policyDef = builder.build()

        try {
            getPolicyClient().putPolicy(policyDef)
        } catch (Exception e) {
            println e.toString()
            return []
        }
        println "Updated lifecycleStage of '${policyName}' to ${stages}"
        return policyMeta.getLifecycleStagesList()
    }

    static updatePolicyEnforcement(String policyName, List<EnforcementAction> enforcementActions) {
        Policy policyMeta = getPolicyByName(policyName)

        def builder = Policy.newBuilder(policyMeta).clearEnforcementActions()
        if (enforcementActions != null && !enforcementActions.isEmpty()) {
            builder.addAllEnforcementActions(enforcementActions)
        } else {
            builder.addAllEnforcementActions([])
        }
        def policyDef = builder.build()

        try {
            getPolicyClient().putPolicy(policyDef)
        } catch (Exception e) {
            println e.toString()
            return ["EXCEPTION"]
        }
        sleep(3000) // Sleep for a little bit to make sure the update propagates in Central.

        if (enforcementActions != null && !enforcementActions.isEmpty()) {
            println "Updated enforcement of '${policyName}' to ${enforcementActions}"
        } else {
            println "Updated enforcement of '${policyName}' to have no enforcement actions"
        }
        return policyMeta.getEnforcementActionsList()
    }

    static submitNetworkGraphSimulation(String yaml, String query = null) {
        println "Generating simulation using YAML:"
        println yaml
        try {
            NetworkPolicyServiceOuterClass.SimulateNetworkGraphRequest.Builder request =
                    NetworkPolicyServiceOuterClass.SimulateNetworkGraphRequest.newBuilder()
                            .setClusterId(ClusterService.getClusterId())
                            .setSimulationYaml(yaml)
            if (query != null) {
                request.setQuery(query)
            }
            return getNetworkPolicyClient().simulateNetworkGraph(request.build())
        } catch (Exception e) {
            println e.toString()
        }
    }

    static getNetworkGraph() {
        try {
            return getNetworkPolicyClient().getNetworkGraph(
                    NetworkPolicyServiceOuterClass.GetNetworkGraphRequest.newBuilder()
                            .setClusterId(ClusterService.getClusterId())
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
                            .setFrom("stackrox")
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

    static sendSimulationNotification(
            String notifierId,
            String yaml,
            String clusterId = ClusterService.getClusterId()) {
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

    static String addGcrRegistryAndScanner() {
        String serviceAccount = System.getenv("GKE_SERVICE_ACCOUNT")
        String gcrId = ""

        try {
            gcrId = getIntegrationClient().postImageIntegration(
                ImageIntegration.newBuilder()
                    .setName("gcr")
                    .setType("google")
                    .addCategories(ImageIntegrationServiceOuterClass.ImageIntegrationCategory.REGISTRY)
                    .addCategories(ImageIntegrationServiceOuterClass.ImageIntegrationCategory.SCANNER)
                        .setGoogle(
                        ImageIntegrationServiceOuterClass.GoogleConfig.newBuilder()
                                .setEndpoint("us.gcr.io")
                                .setProject("ultra-current-825")
                                .setServiceAccount(serviceAccount)
                                .build()
                ).build()
            )
            .getId()
        } catch (Exception e) {
            println e.toString()
        }

        return gcrId
    }

    static deleteGcrRegistryAndScanner(String gcrId) {
        try {
            getIntegrationClient().deleteImageIntegration(
                    ResourceByID.newBuilder()
                            .setId(gcrId)
                            .build()
            )
        } catch (Exception e) {
            println e.toString()
        }
    }

    static applyEnforcement(SensorEventIservice.SensorEnforcement.Builder builder) {
        try {
            return getEnforcementClient().applyEnforcement(
                    EnforcementServiceOuterClass.EnforcementRequest.newBuilder()
                            .setClusterId(ClusterService.getClusterId())
                            .setEnforcement(builder)
                    .build()
            )
        } catch (Exception e) {
            println e.toString()
        }
    }

    static applyKillEnforcement(String podId, String containerId) {
        SensorEventIservice.SensorEnforcement.Builder killEnforcementBuilder =
                SensorEventIservice.SensorEnforcement.newBuilder()
                        .setEnforcement(EnforcementAction.KILL_POD_ENFORCEMENT)
                        .setContainerInstance(SensorEventIservice.ContainerInstanceEnforcement.newBuilder()
                                .setContainerInstanceId(containerId)
                                .setPodId(podId)
                        )
        return applyEnforcement(killEnforcementBuilder)
    }

    static applyScaleDownEnforcement(objects.Deployment deployment) {
        SensorEventIservice.SensorEnforcement.Builder scaleDownEnforcementBuilder =
                SensorEventIservice.SensorEnforcement.newBuilder()
                        .setEnforcement(EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT)
                        .setDeployment(SensorEventIservice.DeploymentEnforcement.newBuilder()
                                .setDeploymentId(deployment.deploymentUid)
                                .setDeploymentName(deployment.name)
                                .setDeploymentType("Deployment")
                                .setNamespace(deployment.namespace)
                                .setAlertId("qa_automation")
                        )
        return applyEnforcement(scaleDownEnforcementBuilder)
    }

    static applyNodeConstraintEnforcement(objects.Deployment deployment) {
        SensorEventIservice.SensorEnforcement.Builder scaleDownEnforcementBuilder =
                SensorEventIservice.SensorEnforcement.newBuilder()
                        .setEnforcement(EnforcementAction.UNSATISFIABLE_NODE_CONSTRAINT_ENFORCEMENT)
                        .setDeployment(SensorEventIservice.DeploymentEnforcement.newBuilder()
                        .setDeploymentId(deployment.deploymentUid)
                        .setDeploymentName(deployment.name)
                        .setDeploymentType("Deployment")
                        .setNamespace(deployment.namespace)
                        .setAlertId("qa_automation")
                )
        return applyEnforcement(scaleDownEnforcementBuilder)
    }

    static waitForNetworkPolicy(String id, int timeoutSeconds = 30) {
        int intervalSeconds = 1
        int waitTime
        for (waitTime = 0; waitTime < timeoutSeconds / intervalSeconds; waitTime++) {
            try {
                getNetworkPolicyClient().getNetworkPolicy(ResourceByID.newBuilder().setId(id).build())
                return true
            } catch (Exception e) {
                println "Exception checking for NetworkPolicy in SR, retrying...:"
                println e.toString()
                sleep(intervalSeconds * 1000)
            }
        }

        println "SR did not detect the network policy"
        return false
    }

    static boolean roxDetectedDeployment(String deploymentID) {
        try {
            def deployment = getDeploymentClient().
                    getDeployment(ResourceByID.newBuilder().setId(deploymentID).build())
            if (deployment.getContainersList().size() == 0) {
                println("Deployment found but it had no containers...")
                return false
            }
            if (deployment.getContainers(0).getImage() == null) {
                println("Deployment found by SR, but images not correlated yet... ")
                return false
            }
            return true
        } catch (Exception e) {
            println "SR does not detect the deployment yet: " + e.toString()
            return false
        }
    }

    static waitForDeployment(objects.Deployment deployment, int timeoutSeconds = 30) {
        if (deployment.deploymentUid == null) {
            println "deploymentID for [${deployment.name}] is null, checking orchestrator directly for deployment ID"
            deployment.deploymentUid = OrchestratorType.orchestrator.getDeploymentId(deployment)
            if (deployment.deploymentUid == null) {
                println "deployment does not exist in orchestrator"
                return false
            }
        }
        int intervalSeconds = 1
        int waitTime
        def startTime = System.currentTimeMillis()
        for (waitTime = 0; waitTime < timeoutSeconds / intervalSeconds; waitTime++) {
            if (roxDetectedDeployment(deployment.deploymentUid)) {
                println "SR found deployment within ${(System.currentTimeMillis() - startTime) / 1000}s"
                return true
            }
            println "Retrying in ${intervalSeconds}..."
            sleep(intervalSeconds * 1000)
        }
        println "SR did not detect the deployment"
        return false
    }
}
