import io.stackrox.proto.api.v1.DeploymentServiceOuterClass
import io.stackrox.proto.api.v1.ImageServiceGrpc
import io.stackrox.proto.api.v1.ImageServiceOuterClass
import io.stackrox.proto.api.v1.DetectionServiceOuterClass.BuildDetectionRequest
import io.stackrox.proto.storage.DeploymentOuterClass.Pod
import io.stackrox.proto.storage.DeploymentOuterClass.ContainerImage
import io.stackrox.proto.storage.RiskOuterClass
import objects.NetworkPolicy
import orchestratormanager.OrchestratorType
import services.AlertService
import services.BaseService
import io.stackrox.proto.api.v1.ImageServiceOuterClass.ListImagesResponse
import io.stackrox.proto.api.v1.Common.ResourceByID
import io.stackrox.proto.api.v1.DeploymentServiceGrpc
import io.stackrox.proto.api.v1.PodServiceGrpc
import io.stackrox.proto.api.v1.DetectionServiceGrpc
import io.stackrox.proto.api.v1.PolicyServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.api.v1.AlertServiceOuterClass.ListAlertsRequest
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.DeploymentOuterClass.ListDeployment
import io.stackrox.proto.storage.DeploymentOuterClass.Deployment
import io.stackrox.proto.storage.ImageOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.EnforcementAction
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.PolicyOuterClass.ListPolicy
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import io.stackrox.proto.storage.PolicyOuterClass.Whitelist
import io.stackrox.proto.storage.ScopeOuterClass
import services.ImageService
import services.NetworkPolicyService
import util.Timer

class Services extends BaseService {

    static ResourceByID getResourceByID(String id) {
        return ResourceByID.newBuilder().setId(id).build()
    }

    static getImageClient() {
        return ImageServiceGrpc.newBlockingStub(getChannel())
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

    static getPodClient() {
        return PodServiceGrpc.newBlockingStub(getChannel())
    }

    static getSearchServiceClient() {
        return SearchServiceGrpc.newBlockingStub(getChannel())
    }

    static List<ListPolicy> getPolicies(RawQuery query = RawQuery.newBuilder().build()) {
        return getPolicyClient().listPolicies(query).policiesList
    }

    static Policy getPolicyByName(String policyName) {
        return getPolicy(
                getPolicies().find { it.name == policyName }.id
        )
    }

    static ImageOuterClass.Image getImageById(String id) {
        return getImageClient().getImage(getResourceByID(id))
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

    static DeploymentServiceOuterClass.GetDeploymentWithRiskResponse getDeploymentWithRisk(String id) {
        return getDeploymentClient().getDeploymentWithRisk(getResourceByID(id))
    }

    static List<Pod> getPods(RawQuery query = RawQuery.newBuilder().build()) {
        return getPodClient().getPods(query).podsList
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
        int retries = timeoutSeconds / intervalSeconds
        Timer t = new Timer(retries, intervalSeconds)
        while (t.IsValid()) {
            RiskOuterClass.Risk risk = Services.getDeploymentWithRisk(deploymentId).risk
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
        int retries = timeoutSeconds / intervalSeconds

        Timer t = new Timer(retries, intervalSeconds)
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

    static scanImage(String image) {
        return getImageClient().scanImage(
                ImageServiceOuterClass.ScanImageRequest.newBuilder()
                         .setImageName(image).build()
        )
    }

    static String getImageIdByName(String imageName) {
        String id = null
        Timer t = new Timer(10, 1)
        while (id == null &&  t.IsValid()) {
            id = ImageService.getImages().find { it?.name == imageName }?.id
        }
        return id
    }

    static List<ListImagesResponse> getImages(RawQuery query = RawQuery.newBuilder().build()) {
        return getImageClient().listImages(query)
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
        sleep 10000 // Sleep for a little bit to make sure the update propagates in Central and then to Sensor.

        if (enforcementActions != null && !enforcementActions.isEmpty()) {
            println "Updated enforcement of '${policyName}' to ${enforcementActions}"
        } else {
            println "Updated enforcement of '${policyName}' to have no enforcement actions"
        }
        return policyMeta.getEnforcementActionsList()
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
        Timer t = new Timer(60, 1)
        boolean disappearedFromStackRox = false
        while (t.IsValid()) {
            if (!roxDetectedDeployment(deployment.deploymentUid, deployment.name)) {
                disappearedFromStackRox = true
                break
            }
        }
        return disappearedFromStackRox
    }

    static waitForDeployment(objects.Deployment deployment, int retries = 30, int interval = 2) {
        if (deployment.deploymentUid == null) {
            println "deploymentID for [${deployment.name}] is null, checking orchestrator directly for deployment ID"
            deployment.deploymentUid = OrchestratorType.orchestrator.getDeploymentId(deployment)
            if (deployment.deploymentUid == null) {
                println "deployment does not exist in orchestrator"
                return false
            }
        }

        Timer t = new Timer(retries, interval)
        while (t.IsValid()) {
            if (roxDetectedDeployment(deployment.deploymentUid, deployment.getName())) {
                println "SR found deployment ${deployment.name} within ${t.SecondsSince()}s"
                return true
            }
            println "SR has not found deployment ${deployment.name} yet"
        }
        println "SR did not detect the deployment ${deployment.name} in ${t.SecondsSince()} seconds"
        return false
    }

    static waitForImage(objects.Deployment deployment, int retries = 30, int interval = 2) {
        def imageName = deployment.image.contains(":") ? deployment.image : deployment.image + ":latest"
        Timer t = new Timer(retries, interval)
        while (t.IsValid()) {
            if (ImageService.getImages().find { it.name.endsWith(imageName) }) {
                println "SR found image ${imageName} within ${t.SecondsSince()}s"
                return true
            }
            println "SR has not found image ${imageName} yet"
        }
        println "SR did not detect the image ${imageName} in ${t.SecondsSince()} seconds"
        return false
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
