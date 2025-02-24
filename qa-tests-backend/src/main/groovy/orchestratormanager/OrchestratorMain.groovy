package orchestratormanager

import groovy.transform.CompileStatic
import io.fabric8.kubernetes.api.model.apps.Deployment as K8sDeployment
import io.fabric8.kubernetes.api.model.batch.v1.Job as K8sJob
import io.fabric8.kubernetes.api.model.ObjectMeta
import io.fabric8.kubernetes.api.model.Pod
import io.fabric8.kubernetes.api.model.StatusDetails
import io.fabric8.kubernetes.api.model.admissionregistration.v1.ValidatingWebhookConfiguration
import io.fabric8.kubernetes.client.LocalPortForward

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

@CompileStatic
interface OrchestratorMain {
    void setup()

    // Pods
    List<Pod> getPods(String ns, String appName)
    List<Pod> getPodsByLabel(String ns, Map<String, String> label)
    Boolean deletePod(String ns, String podName, Long gracePeriodSecs)
    Boolean deletePodAndWait(String ns, String podName, int retries, int intervalSeconds)
    List<StatusDetails> deleteAllPods(String ns, Map<String, String> labels)
    boolean restartPodByLabels(String ns, Map<String, String> labels, int retries, int intervalSecond)
    boolean waitForAllPodsToBeRemoved(String ns, Map<String, String>labels, int iterations, int intervalSeconds)
    boolean waitForPodsReady(String ns, Map<String, String> labels, int minReady, int iterations, int intervalSeconds)
    boolean waitForPodRestart(String ns, String name, int prevRestartCount, int retries, int intervalSeconds)
    String getPodLog(String ns, String name)
    boolean copyFileToPod(String fromPath, String ns, String podName, String toPath)
    boolean podReady(Pod pod)
    Pod addPodAnnotationByApp(String ns, String appName, String key, String value)

    //Deployments
    K8sDeployment getOrchestratorDeployment(String ns, String name)
    K8sDeployment createOrchestratorDeployment(K8sDeployment dep)
    boolean createDeploymentNoWait(Deployment deployment)
    @SuppressWarnings('BuilderMethodWithSideEffects')
    void createDeployment(Deployment deployment)
    void updateDeployment(Deployment deployment)
    boolean updateDeploymentNoWait(Deployment deployment)
    void batchCreateDeployments(List<Deployment> deployments)
    void deleteAndWaitForDeploymentDeletion(Deployment... deployments)
    void deleteDeployment(Deployment deployment)
    void waitForDeploymentDeletion(Deployment deploy)
    String getDeploymentId(Deployment deployment)
    Integer getDeploymentReplicaCount(Deployment deployment)
    Integer getDeploymentUnavailableReplicaCount(Deployment deployment)
    Map<String, String> getDeploymentNodeSelectors(Deployment deployment)
    List<String> getDeploymentCount()
    List<String> getDeploymentCount(String ns)
    Set<String> getDeploymentSecrets(Deployment deployment)
    LocalPortForward createPortForward(int port, Deployment deployment)
    void scaleDeployment(String ns, String name, Integer replicas)
    List<String> getDeployments(String ns)
    boolean deploymentReady(String ns, String name)

    //DaemonSets
    @SuppressWarnings('BuilderMethodWithSideEffects')
    void createDaemonSet(DaemonSet daemonSet)
    void deleteDaemonSet(DaemonSet daemonSet)
    boolean containsDaemonSetContainer(String ns, String name, String containerName)
    void updateDaemonSetEnv(String ns, String name, String containerName, String key, String value)
    Integer getDaemonSetReplicaCount(DaemonSet daemonSet)
    Map<String, String> getDaemonSetNodeSelectors(DaemonSet daemonSet)
    Integer getDaemonSetUnavailableReplicaCount(DaemonSet daemonSet)
    List<String> getDaemonSetCount()
    List<String> getDaemonSetCount(String ns)
    boolean daemonSetReady(String ns, String name)
    boolean daemonSetEnvVarUpdated(String ns, String name, String containerName, String envVarName, String envVarValue)
    void waitForDaemonSetDeletion(String name)
    String getDaemonSetId(DaemonSet daemonSet)

    // StatefulSets
    List<String> getStatefulSetCount()

    //Containers
    void deleteContainer(String containerName, String namespace)
    boolean wasContainerKilled(String containerName, String namespace)
    boolean isKubeDashboardRunning()
    String getContainerlogs(String ns, String podName, String containerName)
    Set<String> getStaticPodCount()
    Set<String> getStaticPodCount(String ns)

    //Services
    @SuppressWarnings('BuilderMethodWithSideEffects')
    void createService(Deployment deployment)
    @SuppressWarnings('BuilderMethodWithSideEffects')
    void createService(Service service)
    void deleteService(String serviceName, String namespace)
    void waitForServiceDeletion(Service service)
    String getServiceIP(String serviceName, String ns)
    void addOrUpdateServiceLabel(String serviceName, String ns, String name, String value)

    //Routes
    @SuppressWarnings('BuilderMethodWithSideEffects')
    void createRoute(String routeName, String namespace)
    void deleteRoute(String routeName, String namespace)

    //Secrets
    String createSecret(Secret secret)
    String createSecret(String name, String namespace)
    String createImagePullSecret(String name, String username, String password, String namespace, String server)
    String createImagePullSecret(Secret secret)
    void deleteSecret(String name, String namespace)
    int getSecretCount(String ns)
    int getSecretCount()
    io.fabric8.kubernetes.api.model.Secret getSecret(String name, String namespace)
    void updateSecret(io.fabric8.kubernetes.api.model.Secret secret)

    //Namespaces
    void ensureNamespaceExists(String ns)
    String createNamespace(String ns)
    void deleteNamespace(String ns)
    boolean waitForNamespaceDeletion(String ns)
    void addNamespaceAnnotation(String ns, String key, String value)
    void removeNamespaceAnnotation(String ns, String key)
    Map<String, List<String>> getAllNetworkPoliciesNamesByNamespace(Boolean ignoreUndoneStackroxGenerated)
    Namespace getNamespaceDetailsByName(String name)
    boolean ownerIsTracked(ObjectMeta obj)
    List<String> getNamespaces()

    //NetworkPolicies
    String applyNetworkPolicy(NetworkPolicy policy)
    boolean deleteNetworkPolicy(NetworkPolicy policy)
    int getNetworkPolicyCount(String ns)

    //Nodes
    int getNodeCount()
    List<Node> getNodeDetails()
    boolean isGKE()

    //Service Accounts
    List<K8sServiceAccount> getServiceAccounts()
    @SuppressWarnings('BuilderMethodWithSideEffects')
    void createServiceAccount(K8sServiceAccount serviceAccount)
    void deleteServiceAccount(K8sServiceAccount serviceAccount)
    void addServiceAccountImagePullSecret(String accountName, String secretName, String namespace)
    void removeServiceAccountImagePullSecret(String accountName, String secretName, String namespace)

    //Roles
    List<K8sRole> getRoles()
    @SuppressWarnings('BuilderMethodWithSideEffects')
    void createRole(K8sRole role)
    void deleteRole(K8sRole role)

    //RoleBindings
    List<K8sRoleBinding> getRoleBindings()
    @SuppressWarnings('BuilderMethodWithSideEffects')
    void createRoleBinding(K8sRoleBinding roleBinding)
    void deleteRoleBinding(K8sRoleBinding roleBinding)

    //ClusterRoles
    List<K8sRole> getClusterRoles()
    @SuppressWarnings('BuilderMethodWithSideEffects')
    void createClusterRole(K8sRole role)
    void deleteClusterRole(K8sRole role)

    //ClusterRoleBindings
    List<K8sRoleBinding> getClusterRoleBindings()
    @SuppressWarnings('BuilderMethodWithSideEffects')
    void createClusterRoleBinding(K8sRoleBinding roleBinding)
    void deleteClusterRoleBinding(K8sRoleBinding roleBinding)

    //Jobs
    K8sJob createJob(Job job)
    void deleteJob(Job job)
    List<String> getJobCount()

    //ConfigMaps
    String createConfigMap(ConfigMap configMap)
    String createConfigMap(String name, Map<String,String> data, String namespace)
    ConfigMap getConfigMap(String name, String namespace)
    void deleteConfigMap(String name, String namespace)

    //Misc
    boolean execInContainer(Deployment deployment, String cmd)
    boolean execInContainerByPodName(String name, String namespace, String cmd, int retries)
    String generateYaml(Object orchestratorObject)
    String getNameSpace()
    String getSensorContainerName()
    void waitForSensor()
    int getAllDeploymentTypesCount(String ns)

    ValidatingWebhookConfiguration getAdmissionController()
    void deleteAdmissionController(String name)
    @SuppressWarnings('BuilderMethodWithSideEffects')
    void createAdmissionController(ValidatingWebhookConfiguration config)
}
