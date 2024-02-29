import static util.Helpers.withRetry

import io.stackrox.proto.api.v1.DetectionServiceOuterClass
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

import spock.lang.Shared
import spock.lang.Tag

class DeploymentCheck extends BaseSpecification {
    private final static String DEPLOYMENT_CHECK = "check-deployments"

    @Shared
    private String clusterId

    def setupSpec(){
        // Ensure a secured cluster exists
        assert ClusterService.getClusters().size() > 0, "There must be at least one secured cluster"
        clusterId = ClusterService.getClusters().get(0).getId()
        assert clusterId

        // Create required resources
        orchestrator.createNamespace(DEPLOYMENT_CHECK)

    }

    def cleanupSpec(){
        orchestrator.deleteNamespace(DEPLOYMENT_CHECK)
        orchestrator.waitForNamespaceDeletion(DEPLOYMENT_CHECK)
    }

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
                clusterAdmin = r
                break
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
        assert res.getRemarks(0).getPermissionLevel() == Rbac.PermissionLevel.CLUSTER_ADMIN.toString()
        assert res.getRemarks(0).getAppliedNetworkPoliciesList().size() == 1
        assert res.getRemarks(0).getAppliedNetworkPolicies(0) == DEPLOYMENT_CHECK
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
      serviceAccountName: check-deployment-sa
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
        """.formatted(deploymentName,deploymentName,deploymentName,deploymentName,deploymentName)
    }
}




