import javax.security.auth.Subject

import io.fabric8.kubernetes.api.model.ServiceAccount
import io.fabric8.kubernetes.api.model.rbac.ClusterRoleBinding
import io.kubernetes.client.proto.V1

import io.stackrox.proto.api.v1.DetectionServiceGrpc
import io.stackrox.proto.storage.ServiceAccountOuterClass

import objects.Deployment
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import services.DetectionService

import spock.lang.Tag

class DeploymentCheck extends BaseSpecification {
    // Test labels - each test has its own unique label space. This is also used to name
    // each tests policy and deployment.
    private final static String DEPLOYMENT_CHECK = "check-deployments"

//    private final static Map<String, NetworkPolicy> POLICIES = [
//            (DEPLOYMENT_CHECK):
//            new NetworkPolicy("multi-port-egress")
//                    .setNamespace("qa")
//                    .addPodSelector("app":DEPLOYMENT_CHECK)
//                    .addPolicyType(NetworkPolicyTypes.INGRESS)
//
//    ]
//
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
                        .setName(DEPLOYMENT_CHECK)
                        .setNamespace("qa")
                        .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-14-alpine")
                        .addPort(80)
                        .setSkipReplicaWait(true)
                        .addLabel("app", DEPLOYMENT_CHECK)
                        .setServiceAccountName(""),
    ]



    def setupSpec(){}

    def cleanupSpec(){}

    /*
    1. Create policies, deployments, network policies and RBACs in the test setup
    2. Call a central API (deployment check)
    3. Assert that the violation slice has (or doesn't) certain violations
    * */

    @Tag("BAT")
    @Tag("Integration")
    @Tag("DeploymentCheck")
    def "Test Deployment Check"(){
        DetectionService.getDetectDeploytimeFromYAML()

    }
}
