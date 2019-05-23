package orchestratormanager

import io.kubernetes.client.models.V1beta1ValidatingWebhookConfiguration
import objects.DaemonSet
import objects.Deployment
import objects.K8sRole
import objects.K8sRoleBinding
import objects.K8sServiceAccount
import objects.Namespace
import objects.NetworkPolicy
import objects.Node
import objects.Service

interface OrchestratorMain {
    def setup()
    def cleanup()

    //Deployments
    io.fabric8.kubernetes.api.model.apps.Deployment getOrchestratorDeployment(String ns, String name)
    def createOrchestratorDeployment(io.fabric8.kubernetes.api.model.apps.Deployment dep)
    def createDeploymentNoWait(Deployment deployment)
    def createDeployment(Deployment deployment)
    def batchCreateDeployments(List<Deployment> deployments)
    def deleteAndWaitForDeploymentDeletion(Deployment... deployments)
    def deleteDeployment(Deployment deployment)
    def waitForDeploymentDeletion(Deployment deploy)
    String getDeploymentId(Deployment deployment)
    def getDeploymentReplicaCount(Deployment deployment)
    def getDeploymentUnavailableReplicaCount(Deployment deployment)
    def getDeploymentNodeSelectors(Deployment deployment)
    def getDeploymentCount(String ns)
    Set<String> getDeploymentSecrets(Deployment deployment)

    //DaemonSets
    def createDaemonSet(DaemonSet daemonSet)
    def deleteDaemonSet(DaemonSet daemonSet)
    def getDaemonSetReplicaCount(DaemonSet daemonSet)
    def getDaemonSetNodeSelectors(DaemonSet daemonSet)
    def getDaemonSetUnavailableReplicaCount(DaemonSet daemonSet)
    def getDaemonSetCount(String ns)
    def waitForDaemonSetDeletion(String name)

    // StatefulSets
    def getStatefulSetCount()

    //Containers
    def deleteContainer(String containerName, String namespace)
    def wasContainerKilled(String containerName, String namespace)
    def isKubeProxyPresent()
    def isKubeDashboardRunning()
    def getContainerlogs(Deployment deployment)
    def getStaticPodCount(String ns)
    def waitForAllPodsToBeRemoved(String ns, Map<String, String>labels, int iterations, int intervalSeconds)

    //Services
    def createService(Deployment deployment)
    def createService(Service service)
    def deleteService(String serviceName, String namespace)
    def waitForServiceDeletion(Service service)

    //Secrets
    def createSecret(String name, String namespace)
    def deleteSecret(String name, String namespace)
    def getSecretCount(String ns)

    //Namespaces
    String createNamespace(String ns)
    def deleteNamespace(String ns)
    def waitForNamespaceDeletion(String ns)
    def getAllNetworkPoliciesNamesByNamespace(Boolean ignoreUndoneStackroxGenerated)

    //NetworkPolicies
    String applyNetworkPolicy(NetworkPolicy policy)
    boolean deleteNetworkPolicy(NetworkPolicy policy)
    def getNetworkPolicyCount(String ns)

    //Nodes
    def getNodeCount()
    List<Node> getNodeDetails()
    def supportsNetworkPolicies()

    //Namespaces
    List<Namespace> getNamespaceDetails()

    //Service Accounts
    List<K8sServiceAccount> getServiceAccounts()
    def createServiceAccount(K8sServiceAccount serviceAccount)
    def deleteServiceAccount(K8sServiceAccount serviceAccount)

    //Roles
    List<K8sRole> getRoles()
    def createRole(K8sRole role)
    def deleteRole(K8sRole role)

    //RoleBindings
    List<K8sRoleBinding> getRoleBindings()
    def createRoleBinding(K8sRoleBinding roleBinding)
    def deleteRoleBinding(K8sRoleBinding roleBinding)

    //ClusterRoles
    List<K8sRole> getClusterRoles()
    def createClusterRole(K8sRole role)
    def deleteClusterRole(K8sRole role)

    //ClusterRoleBindings
    List<K8sRoleBinding> getClusterRoleBindings()
    def createClusterRoleBinding(K8sRoleBinding roleBinding)
    def deleteClusterRoleBinding(K8sRoleBinding roleBinding)

    //Misc
    def execInContainer(Deployment deployment, String cmd)
    def createClairifyDeployment()
    String getClairifyEndpoint()
    String generateYaml(Object orchestratorObject)
    String getNameSpace()
    String getSensorContainerName()
    def waitForSensor()

    V1beta1ValidatingWebhookConfiguration getAdmissionController()
    def deleteAdmissionController(String name)
    def createAdmissionController(V1beta1ValidatingWebhookConfiguration config)

    /*TODO:
        def getDeploymenton(String deploymentName)
        def updateDeploymenton()
    */
}
