package orchestratormanager

import io.fabric8.kubernetes.api.model.EnvVar
import io.fabric8.kubernetes.api.model.Pod
import io.fabric8.kubernetes.api.model.admissionregistration.v1.ValidatingWebhookConfiguration
import objects.ConfigMap
import objects.DaemonSet
import objects.Deployment
import objects.Job
import objects.K8sRole
import objects.K8sRoleBinding
import objects.K8sServiceAccount
import objects.Namespace
import objects.NetworkPolicy
import objects.Node
import objects.Secret
import objects.Service

interface OrchestratorMain {
    def setup()
    def cleanup()

    // Pods
    List<Pod> getPods(String ns, String appName)
    List<Pod> getPodsByLabel(String ns, Map<String, String> label)
    Boolean deletePod(String ns, String podName, Long gracePeriodSecs)
    Boolean deletePodAndWait(String ns, String podName, int retries, int intervalSeconds)
    def deleteAllPods(String ns, Map<String, String> labels)
    void deleteAllPodsAndWait(String ns, Map<String, String> labels)
    Boolean restartPodByLabelWithExecKill(String ns, Map<String, String> labels)
    def restartPodByLabels(String ns, Map<String, String> labels, int retries, int intervalSecond)
    def waitForAllPodsToBeRemoved(String ns, Map<String, String>labels, int iterations, int intervalSeconds)
    def waitForPodsReady(String ns, Map<String, String> labels, int minReady, int iterations, int intervalSeconds)
    def waitForPodRestart(String ns, String name, int prevRestartCount, int retries, int intervalSeconds)
    String getPodLog(String ns, String name)
    def copyFileToPod(String fromPath, String ns, String podName, String toPath)
    boolean podReady(Pod pod)
    def addPodAnnotationByApp(String ns, String appName, String key, String value)

    //Deployments
    io.fabric8.kubernetes.api.model.apps.Deployment getOrchestratorDeployment(String ns, String name)
    def createOrchestratorDeployment(io.fabric8.kubernetes.api.model.apps.Deployment dep)
    def createDeploymentNoWait(Deployment deployment)
    def createDeployment(Deployment deployment)
    def updateDeployment(Deployment deployment)
    boolean updateDeploymentNoWait(Deployment deployment)
    def batchCreateDeployments(List<Deployment> deployments)
    def deleteAndWaitForDeploymentDeletion(Deployment... deployments)
    def deleteDeployment(Deployment deployment)
    def waitForDeploymentDeletion(Deployment deploy)
    String getDeploymentId(Deployment deployment)
    def getDeploymentReplicaCount(Deployment deployment)
    def getDeploymentUnavailableReplicaCount(Deployment deployment)
    def getDeploymentNodeSelectors(Deployment deployment)
    def getDeploymentCount()
    def getDeploymentCount(String ns)
    Set<String> getDeploymentSecrets(Deployment deployment)
    def createPortForward(int port, Deployment deployment)
    def updateDeploymentEnv(String ns, String name, String key, String value)
    EnvVar getDeploymentEnv(String ns, String name, String key)
    def scaleDeployment(String ns, String name, Integer replicas)
    List<String> getDeployments(String ns)
    boolean deploymentReady(String ns, String name)

    //DaemonSets
    def createDaemonSet(DaemonSet daemonSet)
    def deleteDaemonSet(DaemonSet daemonSet)
    boolean containsDaemonSetContainer(String ns, String name, String containerName)
    def updateDaemonSetEnv(String ns, String name, String containerName, String key, String value)
    def getDaemonSetReplicaCount(DaemonSet daemonSet)
    def getDaemonSetNodeSelectors(DaemonSet daemonSet)
    def getDaemonSetUnavailableReplicaCount(DaemonSet daemonSet)
    def getDaemonSetCount()
    def getDaemonSetCount(String ns)
    boolean daemonSetReady(String ns, String name)
    boolean daemonSetEnvVarUpdated(String ns, String name, String containerName, String envVarName, String envVarValue)
    def waitForDaemonSetDeletion(String name)
    String getDaemonSetId(DaemonSet daemonSet)

    // StatefulSets
    def getStatefulSetCount()

    //Containers
    def deleteContainer(String containerName, String namespace)
    def wasContainerKilled(String containerName, String namespace)
    def isKubeDashboardRunning()
    String getContainerlogs(String ns, String podName, String containerName)
    def getStaticPodCount()
    def getStaticPodCount(String ns)

    //Services
    def createService(Deployment deployment)
    def createService(Service service)
    def deleteService(String serviceName, String namespace)
    def waitForServiceDeletion(Service service)
    def getServiceIP(String serviceName, String ns)
    def addOrUpdateServiceLabel(String serviceName, String ns, String name, String value)

    //Routes
    def createRoute(String routeName, String namespace)
    def deleteRoute(String routeName, String namespace)

    //Secrets
    def createSecret(Secret secret)
    def createSecret(String name, String namespace)
    def createImagePullSecret(String name, String username, String password, String namespace, String server)
    def createImagePullSecret(Secret secret)
    def deleteSecret(String name, String namespace)
    int getSecretCount(String ns)
    int getSecretCount()
    io.fabric8.kubernetes.api.model.Secret getSecret(String name, String namespace)
    def updateSecret(io.fabric8.kubernetes.api.model.Secret secret)

    //Namespaces
    def ensureNamespaceExists(String ns)
    String createNamespace(String ns)
    def deleteNamespace(String ns)
    def waitForNamespaceDeletion(String ns)
    def addNamespaceAnnotation(String ns, String key, String value)
    def removeNamespaceAnnotation(String ns, String key)
    def getAllNetworkPoliciesNamesByNamespace(Boolean ignoreUndoneStackroxGenerated)
    List<Namespace> getNamespaceDetails()
    List<String> getNamespaces()

    //NetworkPolicies
    String applyNetworkPolicy(NetworkPolicy policy)
    boolean deleteNetworkPolicy(NetworkPolicy policy)
    def getNetworkPolicyCount(String ns)

    //Nodes
    def getNodeCount()
    List<Node> getNodeDetails()
    def isGKE()

    //Service Accounts
    List<K8sServiceAccount> getServiceAccounts()
    def createServiceAccount(K8sServiceAccount serviceAccount)
    def deleteServiceAccount(K8sServiceAccount serviceAccount)
    def addServiceAccountImagePullSecret(String accountName, String secretName, String namespace)
    def removeServiceAccountImagePullSecret(String accountName, String secretName, String namespace)

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

    //Jobs
    def createJob(Job job)
    def deleteJob(Job job)
    def getJobCount()

    //ConfigMaps
    def createConfigMap(ConfigMap configMap)
    def createConfigMap(String name, Map<String,String> data, String namespace)
    ConfigMap getConfigMap(String name, String namespace)
    def deleteConfigMap(String name, String namespace)

    //Misc
    def execInContainer(Deployment deployment, String cmd)
    boolean execInContainerByPodName(String name, String namespace, String cmd, int retries)
    String generateYaml(Object orchestratorObject)
    String getNameSpace()
    String getSensorContainerName()
    def waitForSensor()
    int getAllDeploymentTypesCount(String ns)

    ValidatingWebhookConfiguration getAdmissionController()
    def deleteAdmissionController(String name)
    def createAdmissionController(ValidatingWebhookConfiguration config)

    /*TODO:
        def getDeploymenton(String deploymentName)
        def updateDeploymenton()
    */
}
