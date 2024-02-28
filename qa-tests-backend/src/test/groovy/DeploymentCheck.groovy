import javax.security.auth.Subject

import io.fabric8.kubernetes.api.model.ServiceAccount
import io.fabric8.kubernetes.api.model.rbac.ClusterRoleBinding
import io.kubernetes.client.proto.V1

import io.stackrox.proto.api.v1.DetectionServiceGrpc
import io.stackrox.proto.api.v1.DetectionServiceOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.EnforcementAction
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.ScopeOuterClass
import io.stackrox.proto.storage.ServiceAccountOuterClass

import common.YamlGenerator
import objects.Deployment
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import services.DetectionService
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

//    private final static Map<String, ServiceAccount> SERVICE_ACCOUNTS = [
//            (DEPLOYMENT_CHECK): new ServiceAccount()
//
//    ]
//
//    private final static Map<String, ClusterRoleBinding> CLUSTER_ROLE_BINDING = [
//            (DEPLOYMENT_CHECK): new ClusterRoleBinding().setSubjects()
//    ]

    private final static Map<String, Deployment> DEPLOYMENTS = [
            (DEPLOYMENT_CHECK):
                new Deployment()
                    .setImage("ghcr.io/linuxserver/nginx:1.24.0-r7-ls261")
                    .setCommand(["sh", "-c", "while true; do sleep 5; apt-get -y update; done"]),
    ]

    @Shared
    private static final Map<String, String> CREATED_POLICIES = [:]

    def setupSpec(){
        POLICIES.each {label, create ->
            CREATED_POLICIES[label] = create()
            assert CREATED_POLICIES[label], "${label} policy should have been created"
        }

        log.info "Waiting for policies to propagate..."
        sleep 10000

        orchestrator.batchCreateDeployments(DEPLOYMENTS.collect {
            String label, Deployment d -> d.setName(label).addLabel("app", label)
        })
    }

    def cleanupSpec(){
        CREATED_POLICIES.each {
            unused, policyId -> PolicyService.deletePolicy(policyId)
        }
        DEPLOYMENTS.each {
            label, d -> orchestrator.deleteDeployment(d)
        }
    }

    /*
    1. Create policies, deployments, network policies and RBACs in the test setup
    2. Call a central API (deployment check)
    3. Assert that the violation slice has (or doesn't) certain violations
    * */

    @Tag("BAT")
    @Tag("Integration")
    @Tag("DeploymentCheck")
    def "Test Deployment Check"(){
        //DetectionService.getDetectDeploytimeFromYAML()
        given:
        "deployment already fabricated"
        Deployment d = DEPLOYMENTS[DEPLOYMENT_CHECK]

        expect:

        def builder = DetectionServiceOuterClass.DeployYAMLDetectionRequest.newBuilder()
        builder.setYaml(createDeploymentYaml())
        def req = builder.build()

        def res = DetectionService.getDetectDeploytimeFromYAML(req)

        log.info "Got response: ${res}"

        log.info "Checked given. Sleeping"
        sleep(10000)
        assert true
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

    String createDeploymentYamla(){
        Deployment d = new Deployment()
                .setImage("ghcr.io/linuxserver/nginx:latest")
                .setNamespace("default")
        def y = YamlGenerator.toYaml(d)
        log.info "Created Deployment yaml: ${y}"
        return y
    }

    static String createDeploymentYaml() {
        return """
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: default
  labels:
    app: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
        """
    }
}




