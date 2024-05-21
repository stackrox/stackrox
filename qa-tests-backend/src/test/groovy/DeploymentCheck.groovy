import static util.Helpers.withRetry

import io.stackrox.proto.api.v1.DetectionServiceOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.Rbac

import objects.K8sRole
import objects.K8sRoleBinding
import objects.K8sServiceAccount
import objects.K8sSubject
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import services.ClusterService
import services.DetectionService
import services.NetworkPolicyService
import services.PolicyCategoryService
import services.PolicyService

import spock.lang.Shared
import spock.lang.Tag

class DeploymentCheck extends BaseSpecification {
    private final static String DEPLOYMENT_CHECK = "check-deployments"
    private final static String DEPLOYMENT_CHECK_POLICY_CATEGORY = "Test Category Filter"

    @Shared
    private String clusterId
    @Shared
    private NetworkPolicy netPol
    @Shared
    private K8sRole clusterAdmin
    @Shared
    private K8sServiceAccount serviceAccount
    @Shared
    private K8sRoleBinding roleBinding
    @Shared
    private String policyCategoryID
    @Shared
    private String policyID

    def setupSpec() {
        // Ensure a secured cluster exists
        assert ClusterService.getClusters().size() > 0, "There must be at least one secured cluster"
        clusterId = ClusterService.getClusters().get(0).getId()
        assert clusterId

        // Create required resources
        orchestrator.createNamespace(DEPLOYMENT_CHECK)

        netPol = new NetworkPolicy(DEPLOYMENT_CHECK)
                .setNamespace(DEPLOYMENT_CHECK)
                .addPodSelector(["app": DEPLOYMENT_CHECK])
                .addPolicyType(NetworkPolicyTypes.INGRESS)
        def netPolID = orchestrator.applyNetworkPolicy(netPol)
        assert NetworkPolicyService.waitForNetworkPolicy(netPolID)

        serviceAccount = new K8sServiceAccount(
                name: "check-deployment-sa",
                namespace: DEPLOYMENT_CHECK,
        )
        orchestrator.createServiceAccount(serviceAccount)

        def orchRoles = orchestrator.getClusterRoles()
        for (K8sRole r : orchRoles) {
            if (r.getName() == "cluster-admin") {
                clusterAdmin = r
                break
            }
        }
        assert clusterAdmin

        roleBinding = new K8sRoleBinding(clusterAdmin, [new K8sSubject(serviceAccount)])
        roleBinding.setNamespace(DEPLOYMENT_CHECK)
        roleBinding.setName(DEPLOYMENT_CHECK)
        orchestrator.createClusterRoleBinding(roleBinding)
    }

    def cleanupSpec() {
        orchestrator.deleteClusterRoleBinding(roleBinding)
        orchestrator.deleteServiceAccount(serviceAccount)
        orchestrator.deleteNetworkPolicy(netPol)
        orchestrator.deleteNamespace(DEPLOYMENT_CHECK)
        orchestrator.waitForNamespaceDeletion(DEPLOYMENT_CHECK)
        if (policyCategoryID != "") {
            PolicyCategoryService.deletePolicyCategory(policyCategoryID)
        }
        if (policyID != "") {
            PolicyService.deletePolicy(policyID)
        }
    }

    @Tag("BAT")
    @Tag("Integration")
    @Tag("DeploymentCheck")
    def "Test Deployment Check - Filtered by Category"() {
        given:
        "create the builder, policy category, and policy"
        def registry = "custom-registry.io"
        def builder = DetectionServiceOuterClass.DeployYAMLDetectionRequest.newBuilder()
        builder.setYaml(createDeploymentYaml(DEPLOYMENT_CHECK, DEPLOYMENT_CHECK, registry+"/nginx:latest"))
        builder.setNamespace(DEPLOYMENT_CHECK)
        builder.setCluster(clusterId)
        builder.addPolicyCategories(DEPLOYMENT_CHECK_POLICY_CATEGORY)
        def req = builder.build()
        DetectionServiceOuterClass.DeployDetectionResponse res
        policyCategoryID = PolicyCategoryService.createNewPolicyCategory(DEPLOYMENT_CHECK_POLICY_CATEGORY)
        def policy = PolicyOuterClass.Policy.newBuilder().
                addLifecycleStages(PolicyOuterClass.LifecycleStage.DEPLOY).
                addCategories(DEPLOYMENT_CHECK_POLICY_CATEGORY).
                setName(DEPLOYMENT_CHECK).
                setDisabled(false).
                setSeverityValue(2).
                addPolicySections(PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                        PolicyOuterClass.PolicyGroup.newBuilder().setFieldName("Image Registry").
                                setBooleanOperator(PolicyOuterClass.BooleanOperator.OR).
                                addAllValues(
                                        [registry].collect { PolicyOuterClass.
                                                PolicyValue.newBuilder().setValue(it).build() }
                                ).build()
                )).
                build()
        policyID = PolicyService.createNewPolicy(policy)

        when:
        withRetry(20, 5) {
            res = DetectionService.getDetectDeploytimeFromYAML(req)
        }

        then:
        assert res
        assert res.getRemarks(0).getName() == DEPLOYMENT_CHECK
        assert res.getRunsCount() == 1
        def run = res.getRuns(0)
        assert run
        assert run.getAlertsCount() == 1
        def alert = run.getAlerts(0)
        assert alert
        assert alert.getPolicy().getName() == DEPLOYMENT_CHECK
        assert alert.getPolicy().getCategoriesCount() == 1
        assert alert.getPolicy().getCategories(0) == DEPLOYMENT_CHECK_POLICY_CATEGORY
    }

    @Tag("BAT")
    @Tag("Integration")
    @Tag("DeploymentCheck")
    def "Test Deployment Check - Single Deployment"() {
        given:
        "builder is prepared"
        def builder = DetectionServiceOuterClass.DeployYAMLDetectionRequest.newBuilder()
        builder.setYaml(createDeploymentYaml(DEPLOYMENT_CHECK, DEPLOYMENT_CHECK))
        builder.setNamespace(DEPLOYMENT_CHECK)
        builder.setCluster(clusterId)
        def req = builder.build()
        DetectionServiceOuterClass.DeployDetectionResponse res

        when:
        withRetry(20, 5) {
            res = DetectionService.getDetectDeploytimeFromYAML(req)
        }
        log.info "Got remarks:\n ${res.remarksList}"

        then:
        assert res
        assert res.getRemarksList().size() == 1
        assert res.getRemarks(0).getName() == DEPLOYMENT_CHECK
        assert res.getRemarks(0).getPermissionLevel() == Rbac.PermissionLevel.CLUSTER_ADMIN.toString()
        assert res.getRemarks(0).getAppliedNetworkPoliciesList().size() == 1
        assert res.getRemarks(0).getAppliedNetworkPolicies(0) == DEPLOYMENT_CHECK
    }

    @Tag("BAT")
    @Tag("Integration")
    @Tag("DeploymentCheck")
    def "Test Deployment Check - Multiple Deployments"() {
        given:
        "builder is prepared"
        def secondDeployment = "de2"
        def builder = DetectionServiceOuterClass.DeployYAMLDetectionRequest.newBuilder()
        def multiDeployments = createDeploymentYaml(DEPLOYMENT_CHECK, DEPLOYMENT_CHECK) + "\n---\n" +
                createDeploymentYaml(secondDeployment, DEPLOYMENT_CHECK)
        builder.setYaml(multiDeployments)
        builder.setNamespace(DEPLOYMENT_CHECK)
        builder.setCluster(clusterId)
        def req = builder.build()
        DetectionServiceOuterClass.DeployDetectionResponse res

        when:
        withRetry(20, 5) {
            res = DetectionService.getDetectDeploytimeFromYAML(req)
        }
        log.info "Got remarks:\n ${res.remarksList}"

        then:
        assert res
        assert res.getRemarksList().size() == 2
        assert !res.getRemarksList().findAll { it.getName() == DEPLOYMENT_CHECK }.isEmpty()
        assert !res.getRemarksList().findAll { it.getName() == secondDeployment }.isEmpty()
    }

    static String createDeploymentYaml(String deploymentName, String namespace,
        String image = "quay.io/rhacs-eng/qa-multi-arch-nginx:latest") {
        """
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app: %s
spec:
  replicas: 3
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      serviceAccountName: check-deployment-sa
      containers:
      - name: nginx
        image: %s
        ports:
        - containerPort: 80
        """.formatted(deploymentName, namespace, deploymentName, deploymentName, deploymentName, image)
    }
}
