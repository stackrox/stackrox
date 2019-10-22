import io.stackrox.proto.api.v1.ImageIntegrationServiceOuterClass
import io.stackrox.proto.api.v1.ImageServiceGrpc
import io.stackrox.proto.api.v1.ImageServiceOuterClass
import io.stackrox.proto.api.v1.RiskServiceGrpc
import io.stackrox.proto.api.v1.RiskServiceOuterClass
import io.stackrox.proto.api.v1.DetectionServiceOuterClass.BuildDetectionRequest
import io.stackrox.proto.api.v1.NotifierServiceOuterClass
import io.stackrox.proto.storage.Common
import io.stackrox.proto.storage.ImageIntegrationOuterClass.ImageIntegration
import io.stackrox.proto.storage.NotifierOuterClass.Notifier
import io.stackrox.proto.storage.DeploymentOuterClass.ContainerImage
import io.stackrox.proto.storage.RiskOuterClass
import objects.NetworkPolicy
import orchestratormanager.OrchestratorType
import services.AlertService
import services.BaseService
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
import services.ImageService
import services.NetworkPolicyService
import util.Env
import util.Timer

class Services extends BaseService {

    static ResourceByID getResourceByID(String id) {
        return ResourceByID.newBuilder().setId(id).build()
      }

    static getImageClient() {
        return ImageServiceGrpc.newBlockingStub(getChannel())
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

    static getDeploymentClient() {
        return DeploymentServiceGrpc.newBlockingStub(getChannel())
      }

    static getRiskClient() {
        return RiskServiceGrpc.newBlockingStub(getChannel())
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

    static int getAlertEnforcementCount(String deploymentName, String policyName) {
        def violations = AlertService.getViolations(ListAlertsRequest.newBuilder()
                .setQuery("Deployment:${deploymentName}+Policy:${policyName}").build())
        return violations.get(0)?.enforcementCount
    }

    static List<ListDeployment> getDeployments(RawQuery query = RawQuery.newBuilder().build()) {
        return getDeploymentClient().listDeployments(query).deploymentsList
      }

    static Deployment getDeployment(String id) {
        return getDeploymentClient().getDeployment(getResourceByID(id))
      }

    static RiskOuterClass.Risk getRisk(String subjectID, RiskOuterClass.RiskSubjectType subjectType) {
        RiskServiceOuterClass.GetRiskRequest request =
                RiskServiceOuterClass.GetRiskRequest.newBuilder()
                        .setSubjectID(subjectID)
                        .setSubjectType(subjectType.toString()).build()
        return getRiskClient().getRisk(request)
    }

    static SearchServiceOuterClass.SearchResponse getSearchResponse(
                  String query, List<SearchServiceOuterClass.SearchCategory> categories) {
        def rawSearchRequest = SearchServiceOuterClass.RawSearchRequest.newBuilder()
                        .addAllCategories(categories)
                        .setQuery(query)
                        .build()
        return getSearchServiceClient().search(rawSearchRequest)
      }

    static waitForSuspiciousProcessInRiskIndicators(String deploymentId, int timeoutSeconds = 30) {
        int intervalSeconds = 3
        int iterations = timeoutSeconds / intervalSeconds
        Timer t = new Timer(iterations, intervalSeconds)
        while (t.IsValid()) {
            RiskOuterClass.Risk risk = Services.getRisk(deploymentId, RiskOuterClass.RiskSubjectType.DEPLOYMENT)
            RiskOuterClass.Risk.Result result = risk.resultsList
                    .find { it.name == "Suspicious Process Executions" }
            if (result != null) {
                return result
            }
        }
        println "No suspicious process executions found in risk indicator after waiting ${t.SecondsSince()} seconds"
        return null
    }

    static waitForViolation(String deploymentName, String policyName, int timeoutSeconds = 30) {
        def violations = getViolationsWithTimeout(deploymentName, policyName, timeoutSeconds)
        return violations != null && violations.size() > 0
      }

    private static getViolationsHelper(String query, String policyName, int timeoutSeconds) {
        int intervalSeconds = 3
        int iterations = timeoutSeconds / intervalSeconds

        Timer t = new Timer(iterations, intervalSeconds)
        while (t.IsValid()) {
            def violations = AlertService.getViolations(ListAlertsRequest.newBuilder()
                    .setQuery(query).build())
            if (violations.size() > 0) {
                println "violation size is: ${violations.size()}"
                println "${policyName} triggered after waiting ${t.SecondsSince()} seconds"
                return violations
            }
        }
        println "Failed to trigger ${policyName} after waiting ${t.SecondsSince()} seconds"
        return []
    }

    static getViolationsWithTimeout(String deploymentName, String policyName, int timeoutSeconds) {
        return getViolationsHelper("Deployment:${deploymentName}+Policy:${policyName}", policyName, timeoutSeconds)
    }

    static getViolationsByDeploymentID(String deploymentID, String policyName, int timeoutSeconds) {
        return getViolationsHelper("Deployment Id:${deploymentID}+Policy:${policyName}", policyName, timeoutSeconds)
    }

    static createImageIntegration(ImageIntegration integration) {
        Timer t = new Timer(15, 3)
        while (t.IsValid()) {
            try {
                ImageIntegration createdIntegration = getIntegrationClient().postImageIntegration(integration)
                return createdIntegration.getId()
            } catch (Exception e) {
                println "Unable to create image integration ${integration.name}: ${e.message}"
            }
        }
        println ("Unable to create image integration")
        return ""
    }

    static final private String AUTO_REGISTERED_SCANNER_INTEGRATION = "Stackrox Scanner"

    // For now, we delete the auto registered StackRox Scanner integration
    // since the QA tests are not stable when we run with them.
    // This function returns whether or not the integration was deleted.
    static boolean deleteAutoRegisteredStackRoxScannerIntegrationIfExists() {
        try {
            // The Stackrox Scanner integration is auto-added by the product,
            // so we first check whether it already exists.
            def scannerIntegrations = getIntegrationClient().getImageIntegrations(
                ImageIntegrationServiceOuterClass.GetImageIntegrationsRequest.
                    newBuilder().
                    setName(AUTO_REGISTERED_SCANNER_INTEGRATION).
                    build()
            )
            if (scannerIntegrations.getIntegrationsCount() > 1) {
                throw new RuntimeException("UNEXPECTED: Got more than one scanner integration: ${scannerIntegrations}")
            }
            if (scannerIntegrations.getIntegrationsCount() == 0) {
                return false
            }
            def id = scannerIntegrations.getIntegrations(0).id
            // Delete
            getIntegrationClient().deleteImageIntegration(ResourceByID.newBuilder().setId(id).build())
            return true
        } catch (Exception e) {
            println "Unable to delete existing Stackrox scanner integration: ${e.message}"
        }
    }

    // This function adds the Stackrox scanner integration.
    // Currently, we do this if it was deleted by the test on setup,
    // so that the test leaves things in the same state after cleanup.
    static String addStackroxScannerIntegration() {
        try {
            def integration = getIntegrationClient().postImageIntegration(ImageIntegration.newBuilder().
                setName(AUTO_REGISTERED_SCANNER_INTEGRATION).
                setType("clairify").
                addCategories(ImageIntegrationOuterClass.ImageIntegrationCategory.SCANNER).
                setClairify(ImageIntegrationOuterClass.ClairifyConfig.newBuilder().
                    setEndpoint("https://scanner.stackrox:8080").
                    build()
                ).
                build()
            )
            return integration.id
        } catch (Exception e) {
            println "Unable to ensure scanner integrations: ${e.message}"
        }
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
        return createImageIntegration(builder.build())
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

    static scanImage(String image) {
        return getImageClient().scanImage(
                ImageServiceOuterClass.ScanImageRequest.newBuilder()
                         .setImageName(image).build()
        )
    }

    static requestBuildImageScan(String registry, String remote, String tag) {
        println "${registry}/${remote}:${tag}"
        return getDetectionClient().detectBuildTime(BuildDetectionRequest.newBuilder().setImage(
               ContainerImage.newBuilder()
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

    static updatePolicyImageWhitelist(String policyName, List<String> images) {
        Policy policyMeta = getPolicyByName(policyName)

        def builder = Policy.newBuilder(policyMeta).clearWhitelists()
        for (String image: images) {
            builder.addWhitelists(
                    Whitelist.newBuilder()
                            .setImage(
                                Whitelist.Image.newBuilder()
                                        .setName(image)
                                        .build()
                            ).build())
        }
        def policyDef = builder.build()

        try {
            getPolicyClient().putPolicy(policyDef)
        } catch (Exception e) {
            println e.toString()
            return []
        }
        println "Updated whitelists of '${policyName}' to ${images}"
        return images
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

    static addTeamsNotifier(String name) {
        try {
            return getNotifierClient().postNotifier(
                    NotifierOuterClass.Notifier.newBuilder()
                            .setType("teams")
                            .setName(name)
                            .setLabelKey("#teams-test")
                            .setLabelDefault(
                                    "https://outlook.office.com/webhook/8a021ef7-9845-449a-a0c0-7bf85eab3955@" +
                                            "6aec22ae-2b26-45bd-b17f-d60e89828e89/IncomingWebhook/9bb3b3574ea2" +
                                            "4655b6482116848bf175/6de97827-1fef-4f8c-a8ab-edac7629df89"
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
                                    .setPassword("xvOOtL7nCOANMbD7ed0522B5")
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
            getNotifierClient().testNotifier(notifier)
            return true
        } catch (Exception e) {
            println e.toString()
            return false
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

    static String addAzureACRRegistry() {
        String azurePassword = System.getenv("AZURE_REGISTRY_PASSWORD")

        ImageIntegration integration =  ImageIntegration.newBuilder()
                .setName("azure")
                .setType("azure")
                .addCategories(ImageIntegrationOuterClass.ImageIntegrationCategory.REGISTRY)
                .setDocker(
                ImageIntegrationOuterClass.DockerConfig.newBuilder()
                        .setEndpoint("stackroxacr.azurecr.io")
                        .setUsername("3e30919c-a552-4b1f-a67a-c68f8b32dad8")
                        .setPassword(azurePassword)
                        .build()
        ).build()
        return createImageIntegration(integration)
    }

    static String addGcrRegistryAndScanner() {
        String serviceAccount = System.getenv("GOOGLE_CREDENTIALS_GCR_SCANNER")

        ImageIntegration integration = ImageIntegration.newBuilder()
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

        return createImageIntegration(integration)
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

    static boolean roxDetectedDeployment(String deploymentID, String name) {
        try {
            def deployment = getDeploymentClient().
                    getDeployment(ResourceByID.newBuilder().setId(deploymentID).build())
            if (deployment.getContainersList().size() == 0) {
                println("Deployment ${name} found but it had no containers...")
                return false
            }
            if (deployment.getContainers(0).getImage() == null) {
                println("Deployment ${name} found by SR, but images not correlated yet... ")
                return false
            }
            return true
        } catch (Exception e) {
            println "SR does not detect the deployment ${name} yet: ${e.toString()}"
            return false
        }
    }

    static boolean waitForSRDeletion(objects.Deployment deployment) {
        // Wait until the deployment disappears from StackRox.
        long sleepTime = 0
        long sleepInterval = 1000
        boolean disappearedFromStackRox = false
        while (sleepTime < 60000) {
            if (!roxDetectedDeployment(deployment.deploymentUid, deployment.name)) {
                disappearedFromStackRox = true
                break
            }
            sleep(sleepInterval)
            sleepTime += sleepInterval
        }
        return disappearedFromStackRox
    }

    static waitForDeployment(objects.Deployment deployment, int iterations = 30, int interval = 2) {
        if (deployment.deploymentUid == null) {
            println "deploymentID for [${deployment.name}] is null, checking orchestrator directly for deployment ID"
            deployment.deploymentUid = OrchestratorType.orchestrator.getDeploymentId(deployment)
            if (deployment.deploymentUid == null) {
                println "deployment does not exist in orchestrator"
                return false
            }
        }

        Timer t = new Timer(iterations, interval)
        while (t.IsValid()) {
            if (roxDetectedDeployment(deployment.deploymentUid, deployment.getName())) {
                println "SR found deployment ${deployment.name} within ${t.SecondsSince()}s"
                return true
            }
            println "SR has not found deployment ${deployment.name} yet"
        }
        println "SR did not detect the deployment ${deployment.name} in ${iterations * interval} seconds"
        return false
    }

    static waitForImage(objects.Deployment deployment, int iterations = 30, int interval = 2) {
        def imageName = deployment.image.contains(":") ? deployment.image : deployment.image + ":latest"
        Timer t = new Timer(iterations, interval)
        while (t.IsValid()) {
            if (ImageService.getImages().find { it.name.endsWith(imageName) }) {
                println "SR found image ${imageName} within ${t.SecondsSince()}s"
                return true
            }
            println "SR has not found image ${imageName} yet"
        }
        println "SR did not detect the image ${imageName} in ${iterations * interval} seconds"
        return false
    }

    static Notifier getWebhookIntegrationConfiguration(Boolean enableTLS, String caCert,
                                                       Boolean skipTLSVerification, Boolean auditLoggingEnabled)  {
        NotifierOuterClass.GenericOrBuilder genericBuilder =  NotifierOuterClass.Generic.newBuilder()
                .setEndpoint("http://webhookserver.stackrox:8080")
                .setCaCert(caCert)
                .setSkipTLSVerify(skipTLSVerification)
                .setAuditLoggingEnabled(auditLoggingEnabled)
                .setUsername("admin")
                .setPassword("admin")
                .addHeaders(
                    Common.KeyValuePair.newBuilder().setKey("headerkey").setValue("headervalue").build()
                )
                .addExtraFields(Common.KeyValuePair.newBuilder().setKey("fieldkey").setValue("fieldvalue").build())
        if (enableTLS) {
            genericBuilder.setEndpoint("https://webhookserver.stackrox:8443")
        }

        return Notifier.newBuilder()
            .setName("generic")
            .setType("generic")
            .setGeneric(genericBuilder.build())
            .setUiEndpoint("localhost:8000")
        .build()
    }

    static addNotifier(Notifier notifier) {
        try {
            return getNotifierClient().postNotifier(notifier).getId()
        } catch (Exception e) {
            println e.toString()
            return ""
        }
    }

    static cleanupNetworkPolicies(List<NetworkPolicy> policies) {
        policies.each {
            if (it.uid) {
                OrchestratorType.orchestrator.deleteNetworkPolicy(it)
            }
        }
        policies.each {
            if (it.uid) {
                NetworkPolicyService.waitForNetworkPolicyRemoval(it.uid)
            }
        }
    }

}
