import static util.Helpers.withRetry

import io.fabric8.openshift.api.model.ClusterRoleBinding
import io.kubernetes.client.proto.V1Rbac

import io.stackrox.proto.api.v1.DetectionServiceOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.EnforcementAction
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.Rbac
import io.stackrox.proto.storage.ScopeOuterClass

import objects.Deployment
import objects.K8sRole
import objects.K8sRoleBinding
import objects.K8sServiceAccount
import objects.K8sSubject
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import services.ClusterService
import services.DetectionService
import services.NetworkPolicyService
import services.PolicyService

import spock.lang.Shared
import spock.lang.Tag

class DeploymentCheck extends BaseSpecification {
    // Test labels - each test has its own unique label space. This is also used to name
    // each tests policy and deployment.
    private final static String DEPLOYMENT_CHECK = "check-deployments"

    // Policies used in this test
    private final static String LATEST_TAG = "Latest tag"

    private final static Map<String, Closure> POLICIES = [
            (DEPLOYMENT_CHECK): {
                duplicatePolicyForTest(
                    LATEST_TAG,
                    DEPLOYMENT_CHECK,
                    [EnforcementAction.FAIL_BUILD_ENFORCEMENT],
                    [LifecycleStage.BUILD, LifecycleStage.DEPLOY]
            )}

    ]

    @Shared
    private String clusterId

    @Shared
    private static final Map<String, String> CREATED_POLICIES = [:]

    def setupSpec(){
        // Ensure a secured cluster exists
        assert ClusterService.getClusters().size() > 0, "There must be at least one secured cluster"
        clusterId = ClusterService.getClusters().get(0).getId()
        assert clusterId

        // Create required resources
        orchestrator.createNamespace(DEPLOYMENT_CHECK)

    }

    def cleanupSpec(){
        CREATED_POLICIES.each {
            unused, policyId -> PolicyService.deletePolicy(policyId)
        }
        orchestrator.deleteNamespace(DEPLOYMENT_CHECK)
        orchestrator.waitForNamespaceDeletion(DEPLOYMENT_CHECK)
    }

    /*
    1. Create policies, deployments, network policies and RBACs in the test setup
    2. Call a central API (deployment check)
    3. Assert that the violation slice has (or doesn't) certain violations
    * */

    @Tag("BAT")
    @Tag("Integration")
    @Tag("DeploymentCheck")
    def "Test Deployment Check - Single Deployment"(){

        given:
        "builder is prepared"
        def builder = DetectionServiceOuterClass.DeployYAMLDetectionRequest.newBuilder()
        builder.setYaml(createDeploymentYaml(DEPLOYMENT_CHECK))
        builder.setNamespace(DEPLOYMENT_CHECK)
        builder.setCluster(clusterId)
        def req = builder.build()
        DetectionServiceOuterClass.DeployDetectionResponse res

        and:
        "network policy has been created"
        NetworkPolicy pol = new NetworkPolicy(DEPLOYMENT_CHECK)
                .setNamespace(DEPLOYMENT_CHECK)
                .addPodSelector(["app":DEPLOYMENT_CHECK])
                .addPolicyType(NetworkPolicyTypes.INGRESS)
        def netPolID = orchestrator.applyNetworkPolicy(pol)
        assert NetworkPolicyService.waitForNetworkPolicy(netPolID)

        and:
        "cluster RBAC has been created"
        def sa = new K8sServiceAccount(
                name: "check-deployment-sa",
                namespace: DEPLOYMENT_CHECK,
        )
        orchestrator.createServiceAccount(sa)

        K8sRole clusterAdmin
        def orchRoles = orchestrator.getClusterRoles()
        for (K8sRole r : orchRoles) {
            if (r.getName() == "cluster-admin"){
                log.info("ClusterRole Name: ${r.getName()}")
                clusterAdmin = r
            }
        }
        assert clusterAdmin
        def crb = new K8sRoleBinding(clusterAdmin, [new K8sSubject(sa)])
        crb.setNamespace(DEPLOYMENT_CHECK)
        crb.setName(DEPLOYMENT_CHECK)
        orchestrator.createClusterRoleBinding(crb)

        when:
        withRetry(20, 5){
            res = DetectionService.getDetectDeploytimeFromYAML(req)
        }
        log.info "Got remarks:\n ${res.remarksList}"

        then:
        assert res
        assert res.getRemarksList().size() > 0
        assert res.getRemarks(0).getName() == DEPLOYMENT_CHECK
        assert res.getRemarks(0).getPermissionLevel() == Rbac.PermissionLevel.NONE.toString()
        assert res.getRemarks(0).getAppliedNetworkPoliciesList().size() == 1
        assert res.getRemarks(0).getAppliedNetworkPolicies(0) == DEPLOYMENT_CHECK

        sleep(10000)
    }

    static String duplicatePolicyForTest(
            String policyName,
            String appLabel,
            List<PolicyOuterClass.EnforcementAction> enforcementActions,
            List<PolicyOuterClass.LifecycleStage> stages = []
    ) {
        PolicyOuterClass.Policy policyMeta = Services.getPolicyByName(policyName)

        def builder = PolicyOuterClass.Policy.newBuilder(policyMeta)

        builder.setId("")
        builder.setName(appLabel)

        builder.addScope(
                ScopeOuterClass.Scope.newBuilder().
                        setLabel(ScopeOuterClass.Scope.Label.newBuilder()
                                .setKey("app").setValue(appLabel)))

        builder.clearEnforcementActions()
        if (enforcementActions != null && !enforcementActions.isEmpty()) {
            builder.addAllEnforcementActions(enforcementActions)
        } else {
            builder.addAllEnforcementActions([])
        }
        if (stages != []) {
            builder.clearLifecycleStages()
            builder.addAllLifecycleStages(stages)
        }

        def policyDef = builder.build()

        return PolicyService.createNewPolicy(policyDef)
    }


    static String createDeploymentYaml(String deploymentName) {
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
      containers:
      - name: nginx
        serviceAccountName: check-deployment-sa
        image: nginx:latest
        ports:
        - containerPort: 80
        """.formatted(deploymentName,deploymentName,deploymentName,deploymentName,deploymentName)
    }
}




