import static util.Helpers.evaluateWithRetry

import groovy.transform.CompileStatic
import orchestratormanager.OrchestratorType
import org.slf4j.Logger
import org.slf4j.LoggerFactory

import io.stackrox.proto.api.v1.AlertServiceOuterClass.ListAlertsRequest
import io.stackrox.proto.api.v1.ClustersServiceGrpc
import io.stackrox.proto.api.v1.Common.ResourceByID
import io.stackrox.proto.api.v1.DeploymentServiceGrpc
import io.stackrox.proto.api.v1.DeploymentServiceOuterClass
import io.stackrox.proto.api.v1.DetectionServiceGrpc
import io.stackrox.proto.api.v1.DetectionServiceOuterClass.BuildDetectionRequest
import io.stackrox.proto.api.v1.ImageServiceGrpc
import io.stackrox.proto.api.v1.ImageServiceOuterClass
import io.stackrox.proto.api.v1.ImageServiceOuterClass.ListImagesResponse
import io.stackrox.proto.api.v1.MetadataServiceGrpc
import io.stackrox.proto.api.v1.PodServiceGrpc
import io.stackrox.proto.api.v1.PolicyServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.storage.AlertOuterClass
import io.stackrox.proto.storage.DeploymentOuterClass.ContainerImage
import io.stackrox.proto.storage.DeploymentOuterClass.Deployment
import io.stackrox.proto.storage.DeploymentOuterClass.ListDeployment
import io.stackrox.proto.storage.DeploymentOuterClass.Pod
import io.stackrox.proto.storage.ImageOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.EnforcementAction
import io.stackrox.proto.storage.PolicyOuterClass.Exclusion
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.PolicyOuterClass.ListPolicy
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import io.stackrox.proto.storage.RiskOuterClass
import io.stackrox.proto.storage.ScopeOuterClass

import objects.NetworkPolicy
import services.AlertService
import services.BaseService
import services.ImageService
import services.NetworkPolicyService
import util.Timer

@CompileStatic
class Services extends BaseService {

    private static final Logger LOG = LoggerFactory.getLogger("test." + Services.getSimpleName())

    static ResourceByID getResourceByID(String id) {
        return ResourceByID.newBuilder().setId(id).build()
    }

    static MetadataServiceGrpc.MetadataServiceBlockingStub getMetadataClient() {
        return MetadataServiceGrpc.newBlockingStub(getChannel())
    }

    static ClustersServiceGrpc.ClustersServiceBlockingStub getClusterClient() {
        return ClustersServiceGrpc.newBlockingStub(getChannel())
    }

    static ImageServiceGrpc.ImageServiceBlockingStub getImageClient() {
        return ImageServiceGrpc.newBlockingStub(getChannel())
    }

    static DetectionServiceGrpc.DetectionServiceBlockingStub getDetectionClient() {
        return DetectionServiceGrpc.newBlockingStub(getChannel())
    }

    static PolicyServiceGrpc.PolicyServiceBlockingStub getPolicyClient() {
        return PolicyServiceGrpc.newBlockingStub(getChannel())
    }

    static DeploymentServiceGrpc.DeploymentServiceBlockingStub getDeploymentClient() {
        return DeploymentServiceGrpc.newBlockingStub(getChannel())
    }

    static PodServiceGrpc.PodServiceBlockingStub getPodClient() {
        return PodServiceGrpc.newBlockingStub(getChannel())
    }

    static SearchServiceGrpc.SearchServiceBlockingStub getSearchServiceClient() {
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
        return getImageClient().getImage(ImageServiceOuterClass.GetImageRequest.newBuilder().setId(id).build())
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
        int retries = (timeoutSeconds / intervalSeconds).intValue()
        Timer t = new Timer(retries, intervalSeconds)
        while (t.IsValid()) {
            RiskOuterClass.Risk risk = getDeploymentWithRisk(deploymentId).risk
            RiskOuterClass.Risk.Result result = risk.resultsList
                    .find { it.name == "Suspicious Process Executions" }
            if (result != null) {
                return result
            }
        }
        LOG.info "No suspicious process executions found in risk indicator after waiting ${t.SecondsSince()} seconds"
        return null
    }

    static waitForViolation(String deploymentName, String policyName, int timeoutSeconds = 30) {
        List<AlertOuterClass.ListAlert> violations = getViolationsWithTimeout(
                deploymentName, policyName, timeoutSeconds)
        if (violations == null || violations.size() == 0) {
            return false // still return false pending debate
        }
        return violations != null && violations.size() > 0
    }

    static waitForResolvedViolation(String deploymentName, String policyName, int timeoutSeconds = 30) {
        def query = "Deployment:${deploymentName}+Policy:${policyName}+Violation State:resolved"
        List<AlertOuterClass.ListAlert> violations = getViolationsHelper(query, policyName, timeoutSeconds)
        if (violations == null || violations.size() == 0) {
            return false // still return false pending debate
        }
        return violations != null && violations.size() > 0
    }

    private static List<AlertOuterClass.ListAlert> getViolationsHelper(
            String query, String policyName, int timeoutSeconds) {
        int intervalSeconds = 3
        int retries = (timeoutSeconds / intervalSeconds).intValue()

        Timer t = new Timer(retries, intervalSeconds)
        while (t.IsValid()) {
            def violations = AlertService.getViolations(ListAlertsRequest.newBuilder()
                    .setQuery(query).build())
            if (violations.size() > 0) {
                LOG.info "violation size is: ${violations.size()}"
                LOG.info "${policyName} triggered after waiting ${t.SecondsSince()} seconds"
                return violations
            }
        }
        LOG.info "Failed to trigger ${policyName} after waiting ${t.SecondsSince()} seconds"
        return []
    }

    static List<AlertOuterClass.ListAlert> getViolationsWithTimeout(
            String deploymentName, String policyName, int timeoutSeconds) {
        return getViolationsHelper("Deployment:${deploymentName}+Policy:${policyName}", policyName, timeoutSeconds)
    }

    static getAllResourceViolationsWithTimeout(String resourceType,
                                               String policyName, int timeoutSeconds) {
        return getViolationsHelper("Resource Type:${resourceType}+Policy:${policyName}",
                policyName, timeoutSeconds)
    }

    static getResourceViolationsWithTimeout(String resourceType, String resourceName,
                                            String policyName, int timeoutSeconds) {
        return getViolationsHelper("Resource Type:${resourceType}+Resource:${resourceName}+Policy:${policyName}",
                policyName, timeoutSeconds)
    }

    static List<AlertOuterClass.ListAlert> getViolationsByDeploymentID(String deploymentID, String policyName,
                                                                       boolean includeAttempted, int timeoutSeconds) {
        if (!includeAttempted) {
            return getViolationsHelper(
                    "Deployment Id:${deploymentID}+Policy:${policyName}+Violation State:Active",
                    policyName, timeoutSeconds)
        }
        // By default active and attempted violations are queried.
        return getViolationsHelper("Deployment Id:${deploymentID}+Policy:${policyName}", policyName, timeoutSeconds)
    }

    static checkForNoViolations(String deploymentName, String policyName, int checkSeconds = 5) {
        def violations = getViolationsWithTimeout(deploymentName, policyName, checkSeconds)
        return violations == null || violations.size() == 0
    }

    static boolean checkForNoViolationsByDeploymentID(String deploymentID, String policyName, int checkSeconds = 5) {
        def violations = getViolationsByDeploymentID(deploymentID, policyName, false, checkSeconds)
        return violations == null || violations.isEmpty()
    }

    static String getImageIdByName(String imageName) {
        String id = null
        Timer t = new Timer(10, 1)
        while (id == null && t.IsValid()) {
            id = ImageService.getImages().find { it?.name == imageName }?.id
        }
        return id
    }

    static ListImagesResponse getImages(RawQuery query = RawQuery.newBuilder().build()) {
        return getImageClient().listImages(query)
    }

    static requestBuildImageScan(String registry, String remote, String tag, Boolean sendNotifications = false) {
        LOG.info "Request scan of ${registry}/${remote}:${tag} with sendNotifications=${sendNotifications}"
        def request = BuildDetectionRequest.newBuilder()
            .setImage(ContainerImage.newBuilder()
                .setName(ImageOuterClass.ImageName.newBuilder()
                    .setRegistry(registry)
                    .setRemote(remote)
                    .setTag(tag)
                    .build()
                )
            )
            .setSendNotifications(sendNotifications)
            .build()
        return evaluateWithRetry(10, 15) {
            return getDetectionClient().detectBuildTime(request)
        }
    }

    static updatePolicy(Policy policyDef) {
        try {
            getPolicyClient().putPolicy(policyDef)
        } catch (Exception e) {
            LOG.warn("exception", e)
        }
    }

    static setPolicyDisabled(String policyName, boolean disabled) {
        Policy policyMeta = getPolicyByName(policyName)

        if (policyMeta.disabled == disabled) {
            return false
        }

        def policyDef = Policy.newBuilder(policyMeta).setDisabled(disabled).build()

        try {
            getPolicyClient().putPolicy(policyDef)
            return true
        } catch (Exception e) {
            LOG.warn("exception", e)
            return false
        }
    }

    static updatePolicyToExclusionDeployment(String policyName, objects.Deployment deployment) {
        Policy policyMeta = getPolicyByName(policyName)

        def policyDef = Policy.
                newBuilder(policyMeta).
                addExclusions(Exclusion.newBuilder().
                        setDeployment(Exclusion.Deployment.newBuilder().
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
            LOG.warn("exception", e)
        }
        LOG.info "Updated ${policyName} to excluded scope ${deployment.getName()}"
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
            LOG.warn("exception", e)
            return []
        }
        LOG.info "Updated lifecycleStage of '${policyName}' to ${stages}"
        return policyMeta.getLifecycleStagesList()
    }

    static updatePolicyImageExclusion(String policyName, List<String> images) {
        Policy policyMeta = getPolicyByName(policyName)

        Policy.Builder builder = Policy.newBuilder(policyMeta).clearExclusions()
        for (String image : images) {
            builder.addExclusions(
                    Exclusion.newBuilder()
                            .setImage(
                                    Exclusion.Image.newBuilder()
                                            .setName(image)
                                            .build()
                            ).build())
        }
        def policyDef = builder.build()

        try {
            getPolicyClient().putPolicy(policyDef)
        } catch (Exception e) {
            LOG.warn("exception", e)
            return []
        }
        LOG.info "Updated exclusions of '${policyName}' to ${images}"
        return images
    }

    static List<EnforcementAction> updatePolicyEnforcement(String policyName,
                                                           List<EnforcementAction> enforcementActions,
                                                           Boolean waitForPropagation = true) {
        Policy policyMeta = getPolicyByName(policyName)

        Policy.Builder builder = Policy.newBuilder(policyMeta).clearEnforcementActions()
        if (enforcementActions != null && !enforcementActions.isEmpty()) {
            builder.addAllEnforcementActions(enforcementActions)
        } else {
            builder.addAllEnforcementActions([])
        }
        def policyDef = builder.build()

        try {
            getPolicyClient().putPolicy(policyDef)
        } catch (Exception e) {
            LOG.warn("updating policy failed", e)
            throw e
        }

        if (waitForPropagation) {
            sleep 10000 // Sleep for a little bit to make sure the update propagates in Central and then to Sensor.
        }

        if (enforcementActions != null && !enforcementActions.isEmpty()) {
            LOG.info "Updated enforcement of '${policyName}' to ${enforcementActions}"
        } else {
            LOG.info "Updated enforcement of '${policyName}' to have no enforcement actions"
        }
        return policyMeta.getEnforcementActionsList()
    }

    static boolean roxDetectedDeployment(String deploymentID, String name) {
        try {
            Deployment deployment = getDeploymentClient().
                    getDeployment(ResourceByID.newBuilder().setId(deploymentID).build())
            if (deployment.getContainersList().size() == 0) {
                LOG.info("Deployment ${name} found but it had no containers...")
                return false
            }
            if (deployment.getContainers(0).getImage() == null) {
                LOG.info("Deployment ${name} found by SR, but images not correlated yet... ")
                return false
            }
            return true
        } catch (Exception e) {
            LOG.info "SR does not detect the deployment ${name} yet: ${e}"
            return false
        }
    }

    static boolean waitForSRDeletion(objects.Deployment deployment) {
        return waitForSRDeletionByID(deployment.deploymentUid, deployment.name)
    }

    static boolean waitForSRDeletionByID(String id, String name) {
        // Wait until the deployment disappears from StackRox.
        Timer t = new Timer(60, 1)
        boolean disappearedFromStackRox = false
        while (t.IsValid()) {
            if (!roxDetectedDeployment(id, name)) {
                disappearedFromStackRox = true
                break
            }
        }
        return disappearedFromStackRox
    }

    static waitForDeployment(objects.Deployment deployment, int retries = 60, int interval = 2) {
        if (deployment.deploymentUid == null) {
            LOG.info "deploymentID for [${deployment.name}] is null, checking orchestrator directly for deployment ID"
            deployment.deploymentUid = OrchestratorType.orchestrator.getDeploymentId(deployment)
            if (deployment.deploymentUid == null) {
                LOG.info "deployment does not exist in orchestrator"
                return false
            }
        }
        return waitForDeploymentByID(deployment.deploymentUid, deployment.name, retries, interval)
    }

    static waitForDeploymentByID(String id, String name, int retries = 30, int interval = 2) {
        Timer t = new Timer(retries, interval)
        while (t.IsValid()) {
            if (roxDetectedDeployment(id, name)) {
                LOG.info "SR found deployment ${name} within ${t.SecondsSince()}s"
                return true
            }
            LOG.info "SR has not found deployment ${name} yet"
        }
        LOG.info "SR did not detect the deployment ${name} in ${t.SecondsSince()} seconds"
        return false
    }

    static waitForImage(objects.Deployment deployment, int retries = 30, int interval = 2) {
        def imageName = deployment.image.contains(":") ? deployment.image : deployment.image + ":latest"
        Timer t = new Timer(retries, interval)
        while (t.IsValid()) {
            if (ImageService.getImages().find { it.name.endsWith(imageName) }) {
                LOG.info "SR found image ${imageName} within ${t.SecondsSince()}s"
                return true
            }
            LOG.info "SR has not found image ${imageName} yet"
        }
        LOG.info "SR did not detect the image ${imageName} in ${t.SecondsSince()} seconds"
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
