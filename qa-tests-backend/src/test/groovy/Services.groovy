import io.stackrox.proto.api.v1.DetectionServiceOuterClass.BuildDetectionRequest
import io.stackrox.proto.api.v1.NotifierServiceOuterClass
import io.stackrox.proto.storage.ImageIntegrationOuterClass.ImageIntegration
import orchestratormanager.OrchestratorType
import services.BaseService
import io.stackrox.proto.api.v1.AlertServiceGrpc
import io.stackrox.proto.api.v1.AlertServiceOuterClass
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsCountsRequest
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsCountsResponse
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertsGroupResponse
import io.stackrox.proto.api.v1.AlertServiceOuterClass.GetAlertTimeseriesResponse
import io.stackrox.proto.api.v1.Common.ResourceByID
import io.stackrox.proto.api.v1.DeploymentServiceGrpc
import io.stackrox.proto.api.v1.DetectionServiceGrpc
import io.stackrox.proto.api.v1.ImageIntegrationServiceGrpc
import io.stackrox.proto.api.v1.NotifierServiceGrpc
import io.stackrox.proto.api.v1.PolicyServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.api.v1.AlertServiceOuterClass.ListAlertsRequest
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.AlertOuterClass.Alert
import io.stackrox.proto.storage.AlertOuterClass.ListAlert
import io.stackrox.proto.storage.DeploymentOuterClass.ListDeployment
import io.stackrox.proto.storage.DeploymentOuterClass.Deployment
import io.stackrox.proto.storage.ImageIntegrationOuterClass
import io.stackrox.proto.storage.ImageOuterClass
import io.stackrox.proto.storage.NotifierOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.EnforcementAction
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.PolicyOuterClass.ListPolicy
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import io.stackrox.proto.storage.PolicyOuterClass.Whitelist
import io.stackrox.proto.storage.ScopeOuterClass
import util.Env

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

    static getNotifierClient() {
        return NotifierServiceGrpc.newBlockingStub(getChannel())
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
                                    .addCategories(ImageIntegrationOuterClass.ImageIntegrationCategory.REGISTRY)
                                    .setType("docker")
                                    .setDocker(
                                    ImageIntegrationOuterClass.DockerConfig.newBuilder()
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

    static String addDockerTrustedRegistry(boolean includeScanner = true) {
        ImageIntegration.Builder builder = ImageIntegration.newBuilder()
                .setName("dtr")
                .setType("dtr")
                .addCategories(ImageIntegrationOuterClass.ImageIntegrationCategory.REGISTRY)
                .setDtr(ImageIntegrationOuterClass.DTRConfig.newBuilder()
                        .setEndpoint("https://apollo-dtr.rox.systems/")
                        .setUsername("qa")
                        .setPassword("W3g9xOPKyLTkBBMj")
                        .setInsecure(false)
                        .build()
                )
        if (includeScanner) {
            builder.addCategories(ImageIntegrationOuterClass.ImageIntegrationCategory.SCANNER)
        }
        return getIntegrationClient().postImageIntegration(builder.build()).getId()
    }

    static String addClairifyScanner(String clairifyEndpoint) {
        def success = false
        ImageIntegration integration = null
        def start = System.currentTimeMillis()
        while (!success && System.currentTimeMillis() - start < 30000) {
            try {
                integration = getIntegrationClient().postImageIntegration(
                        ImageIntegration.newBuilder()
                                .setName("clairify")
                                .setType("clairify")
                                .addCategories(ImageIntegrationOuterClass.ImageIntegrationCategory.SCANNER)
                                .setClairify(ImageIntegrationOuterClass.ClairifyConfig.newBuilder()
                                        .setEndpoint(clairifyEndpoint)
                                        .build()
                                )
                                .build()
                        )
                success = true
            } catch (Exception e) {
                if (e.toString().contains("INTERNAL: notifying of update errors:")) {
                    def id = getIntegrationClient().getImageIntegrations().integrationsList.find {
                        it.name == "clairify"
                    }?.id
                    if (id) {
                        getIntegrationClient().deleteImageIntegration(
                                ResourceByID.newBuilder().setId(id).build()
                        )
                    }
                } else {
                    println "Failed to create integration: ${e.toString()}"
                }
                sleep 3000
            }
        }
        return integration?.id
      }

    static deleteClairifyScanner(String clairifyId) {
        try {
            getIntegrationClient().deleteImageIntegration(
                    ResourceByID.newBuilder()
                            .setId(clairifyId)
                            .build()
            )
        } catch (Exception e) {
            println "Failed to delete integration: ${e.toString()}"
            return false
        }
      }

    static deleteImageIntegration(String integrationId) {
        try {
            getIntegrationClient().deleteImageIntegration(
                    ResourceByID.newBuilder()
                            .setId(integrationId)
                            .build()
            )
        } catch (Exception e) {
            println "Failed to delete integration: ${e.toString()}"
            return false
        }
        try {
            ImageIntegration integration = getIntegrationClient().getImageIntegration(
                    ResourceByID.newBuilder().setId(integrationId).build()
            )
            while (integration) {
                integration = getIntegrationClient().getImageIntegration(
                        ResourceByID.newBuilder().setId(integrationId).build()
                )
                sleep 2000
            }
        } catch (Exception e) {
            return e.toString().contains("NOT_FOUND")
        }
    }

    static requestBuildImageScan(String registry, String remote, String tag) {
        return getDetectionClient().detectBuildTime(BuildDetectionRequest.newBuilder().setImage(
                ImageOuterClass.Image.newBuilder()
                        .setName(ImageOuterClass.ImageName.newBuilder()
                        .setRegistry(registry)
                        .setRemote(remote)
                        .setTag(tag)
                        .build()
                )
        ).build())
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
            addWhitelists(Whitelist.newBuilder().
                setDeployment(Whitelist.Deployment.newBuilder().
                    setName(deployment.getName()).
                    setScope(ScopeOuterClass.Scope.newBuilder().
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

    static addSlackNotifier(String name) {
        try {
            return getNotifierClient().postNotifier(
                    NotifierOuterClass.Notifier.newBuilder()
                            .setType("slack")
                            .setName(name)
                            .setLabelKey("#slack-test")
                            .setLabelDefault(
                                "https://hooks.slack.com/services/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"
                            )
                            .setEnabled(true)
                            .setUiEndpoint("https://" +
                                    Env.mustGetHostname() + ":" +
                                    Env.mustGetPort())
                    .build()
            )
        } catch (Exception e) {
            println e.toString()
        }
    }

    static addJiraNotifier(String name) {
        try {
            return getNotifierClient().postNotifier(
                    NotifierOuterClass.Notifier.newBuilder()
                            .setType("jira")
                            .setName(name)
                            .setLabelKey("AJIT")
                            .setLabelDefault("AJIT")
                            .setEnabled(true)
                            .setUiEndpoint("https://" +
                                    Env.mustGetHostname() + ":" +
                                    Env.mustGetPort())
                            .setJira(NotifierOuterClass.Jira.newBuilder()
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
            NotifierOuterClass.Notifier.Builder builder =
                    NotifierOuterClass.Notifier.newBuilder()
                            .setEmail(NotifierOuterClass.Email.newBuilder())
            builder
                    .setType("email")
                    .setName(name)
                    .setLabelKey("mailgun")
                    .setLabelDefault("to@example.com")
                    .setEnabled(true)
                    .setUiEndpoint("https://" +
                            Env.mustGetHostname() + ":" +
                            Env.mustGetPort())
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

    static testNotifier(NotifierOuterClass.Notifier notifier) {
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

    static String addGcrRegistryAndScanner() {
        String serviceAccount = System.getenv("GOOGLE_CREDENTIALS_GCR_SCANNER")
        String gcrId = ""

        try {
            gcrId = getIntegrationClient().postImageIntegration(
                ImageIntegrationOuterClass.ImageIntegration.newBuilder()
                    .setName("gcr")
                    .setType("google")
                    .addCategories(ImageIntegrationOuterClass.ImageIntegrationCategory.REGISTRY)
                    .addCategories(ImageIntegrationOuterClass.ImageIntegrationCategory.SCANNER)
                        .setGoogle(
                        ImageIntegrationOuterClass.GoogleConfig.newBuilder()
                                .setEndpoint("us.gcr.io")
                                .setProject("stackrox-ci")
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
