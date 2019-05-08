package orchestratormanager

import common.YamlGenerator
import io.fabric8.kubernetes.api.model.Capabilities
import io.fabric8.kubernetes.api.model.Container
import io.fabric8.kubernetes.api.model.ContainerPort
import io.fabric8.kubernetes.api.model.ContainerStatus
import io.fabric8.kubernetes.api.model.EnvVar
import io.fabric8.kubernetes.api.model.HostPathVolumeSource
import io.fabric8.kubernetes.api.model.IntOrString
import io.fabric8.kubernetes.api.model.LabelSelector
import io.fabric8.kubernetes.api.model.LocalObjectReference
import io.fabric8.kubernetes.api.model.Namespace
import io.fabric8.kubernetes.api.model.ObjectMeta
import io.fabric8.kubernetes.api.model.Pod
import io.fabric8.kubernetes.api.model.PodList
import io.fabric8.kubernetes.api.model.PodSpec
import io.fabric8.kubernetes.api.model.PodTemplateSpec
import io.fabric8.kubernetes.api.model.Quantity
import io.fabric8.kubernetes.api.model.ResourceRequirements
import io.fabric8.kubernetes.api.model.Secret
import io.fabric8.kubernetes.api.model.SecretVolumeSource
import io.fabric8.kubernetes.api.model.SecurityContext
import io.fabric8.kubernetes.api.model.Service
import io.fabric8.kubernetes.api.model.ServiceAccount
import io.fabric8.kubernetes.api.model.ServiceList
import io.fabric8.kubernetes.api.model.ServicePort
import io.fabric8.kubernetes.api.model.ServiceSpec
import io.fabric8.kubernetes.api.model.Volume
import io.fabric8.kubernetes.api.model.VolumeMount
import io.fabric8.kubernetes.api.model.apps.Deployment as K8sDeployment
import io.fabric8.kubernetes.api.model.apps.DaemonSet as K8sDaemonSet
import io.fabric8.kubernetes.api.model.apps.DaemonSetList
import io.fabric8.kubernetes.api.model.apps.DaemonSetSpec
import io.fabric8.kubernetes.api.model.apps.DeploymentList
import io.fabric8.kubernetes.api.model.apps.DeploymentSpec
import io.fabric8.kubernetes.api.model.apps.DoneableDaemonSet
import io.fabric8.kubernetes.api.model.apps.DoneableDeployment
import io.fabric8.kubernetes.api.model.networking.NetworkPolicyBuilder
import io.fabric8.kubernetes.api.model.networking.NetworkPolicyEgressRuleBuilder
import io.fabric8.kubernetes.api.model.networking.NetworkPolicyIngressRuleBuilder
import io.fabric8.kubernetes.api.model.networking.NetworkPolicyPeerBuilder
import io.fabric8.kubernetes.api.model.rbac.ClusterRole
import io.fabric8.kubernetes.api.model.rbac.ClusterRoleBinding
import io.fabric8.kubernetes.api.model.rbac.PolicyRule
import io.fabric8.kubernetes.api.model.rbac.Role
import io.fabric8.kubernetes.api.model.rbac.RoleBinding
import io.fabric8.kubernetes.api.model.rbac.RoleRef
import io.fabric8.kubernetes.api.model.rbac.Subject
import io.fabric8.kubernetes.client.Callback
import io.fabric8.kubernetes.client.DefaultKubernetesClient
import io.fabric8.kubernetes.client.KubernetesClient
import io.fabric8.kubernetes.client.KubernetesClientException
import io.fabric8.kubernetes.client.dsl.ExecListener
import io.fabric8.kubernetes.client.dsl.ExecWatch
import io.fabric8.kubernetes.client.dsl.MixedOperation
import io.fabric8.kubernetes.client.dsl.Resource
import io.fabric8.kubernetes.client.dsl.ScalableResource
import io.fabric8.kubernetes.client.utils.BlockingInputStreamPumper
import io.kubernetes.client.models.V1beta1ValidatingWebhookConfiguration
import objects.DaemonSet
import objects.Deployment
import objects.K8sPolicyRule
import objects.K8sRole
import objects.K8sRoleBinding
import objects.K8sServiceAccount
import objects.K8sSubject
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import objects.Node
import okhttp3.Response
import util.Timer

import java.util.concurrent.CountDownLatch
import java.util.concurrent.Executors
import java.util.concurrent.Future
import java.util.concurrent.ScheduledExecutorService
import java.util.concurrent.TimeUnit
import java.util.stream.Collectors

class Kubernetes implements OrchestratorMain {
    final int sleepDuration = 5000
    final int maxWaitTime = 90000

    String namespace
    KubernetesClient client

    MixedOperation<K8sDaemonSet, DaemonSetList, DoneableDaemonSet, Resource<K8sDaemonSet, DoneableDaemonSet>> daemonsets

    MixedOperation<K8sDeployment, DeploymentList, DoneableDeployment,
            ScalableResource<K8sDeployment, DoneableDeployment>> deployments

    Kubernetes(String ns) {
        this.namespace = ns
        this.client = new DefaultKubernetesClient()
        this.client.configuration.setRollingTimeout(60 * 60 * 1000)
        this.deployments = this.client.apps().deployments()
        this.daemonsets = this.client.apps().daemonSets()
    }

    Kubernetes() {
        Kubernetes("default")
    }

    def ensureNamespaceExists(String ns) {
        Namespace namespace = new Namespace("v1", null, new ObjectMeta(name: ns), null, null)
        try {
            client.namespaces().create(namespace)
            println "Created namespace ${ns}"
        } catch (KubernetesClientException kce) {
            // 409 is already exists
            if (kce.code != 409) {
                throw kce
            }
        }
    }

    def setup() {
        ensureNamespaceExists(this.namespace)
    }

    def cleanup() {
    }

    /*
        Deployment Methods
    */

    def createDeployment(Deployment deployment) {
        ensureNamespaceExists(deployment.namespace)
        createDeploymentNoWait(deployment)
        waitForDeploymentAndPopulateInfo(deployment)
    }

    def batchCreateDeployments(List<Deployment> deployments) {
        for (Deployment deployment : deployments) {
            ensureNamespaceExists(deployment.namespace)
            createDeploymentNoWait(deployment)
        }
        for (Deployment deployment : deployments) {
            waitForDeploymentAndPopulateInfo(deployment)
        }
    }

    def getAndPrintPods(String ns, String name) {
        LabelSelector selector = new LabelSelector()
        selector.matchLabels = new HashMap<String, String>()
        selector.matchLabels.put("app", name)
        PodList list = client.pods().inNamespace(ns).withLabelSelector(selector).list()

        println "Status of ${name}'s pods:"
        for (Pod pod : list.getItems()) {
            println "\t- ${pod.metadata.name}"
            for (ContainerStatus status : pod.status.containerStatuses) {
                println "\t  Container status: ${status.state}"
            }
        }
    }

    def waitForDeploymentDeletion(Deployment deploy, int iterations = 30, int intervalSeconds = 5) {
        Timer t = new Timer(iterations, intervalSeconds)

        K8sDeployment d
        while (t.IsValid()) {
            d = this.deployments.inNamespace(deploy.namespace).withName(deploy.name).get()
            if (d == null) {
                println "${deploy.name}: deployment removed."
                return
            }
            getAndPrintPods(deploy.namespace, deploy.name)
        }
        println "Timed out waiting for deployment ${deploy.name} to be deleted"
    }

    def deleteAndWaitForDeploymentDeletion(Deployment... deployments) {
        for (Deployment deployment : deployments) {
            this.deleteDeployment(deployment)
        }
        for (Deployment deployment : deployments) {
            this.waitForDeploymentDeletion(deployment)
        }
    }

    def deleteDeployment(Deployment deployment) {
        if (deployment.exposeAsService) {
            this.deleteService(deployment.name, deployment.namespace)
        }
        // Retry deletion due to race condition in sdk and controller
        // See https://github.com/fabric8io/kubernetes-client/issues/1477
        Timer t = new Timer(10, 1)
        while (t.IsValid()) {
            try {
                this.deployments.inNamespace(deployment.namespace).withName(deployment.name).delete()
                break
            } catch (KubernetesClientException ex) {
                println "Failed to delete deployment: ${ex.status.message}"
            }
        }
        println "removing the deployment: ${deployment.name}"
    }

    def createOrchestratorDeployment(K8sDeployment dep) {
        dep.setApiVersion("")
        dep.metadata.setResourceVersion("")
        return this.deployments.create(dep)
    }

    K8sDeployment getOrchestratorDeployment(String ns, String name) {
        return this.deployments.inNamespace(ns).withName(name).get()
    }

    String getDeploymentId(Deployment deployment) {
        return this.deployments.inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get()?.metadata?.uid
    }

    def getDeploymentReplicaCount(Deployment deployment) {
        K8sDeployment d = this.deployments.inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get()
        if (d != null) {
            println "${deployment.name}: Replicas=${d.getSpec().getReplicas()}"
            return d.getSpec().getReplicas()
        }
    }

    def getDeploymentUnavailableReplicaCount(Deployment deployment) {
        K8sDeployment d = this.deployments
                .inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get()
        if (d != null) {
            println "${deployment.name}: Unavailable Replicas=${d.getStatus().getUnavailableReplicas()}"
            return d.getStatus().getUnavailableReplicas()
        }
    }

    def getDeploymentNodeSelectors(Deployment deployment) {
        K8sDeployment d = this.deployments
                .inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get()
        if (d != null) {
            println "${deployment.name}: Host=${d.getSpec().getTemplate().getSpec().getNodeSelector()}"
            return d.getSpec().getTemplate().getSpec().getNodeSelector()
        }
    }

    Set<String> getDeploymentSecrets(Deployment deployment) {
        K8sDeployment d = this.deployments
                .inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get()
        Set<String> secretSet = [] as Set
        if (d != null) {
            d.getSpec()?.getTemplate()?.getSpec()?.getVolumes()?.each { secretSet.add(it.secret.secretName) }
        }
        return secretSet
    }

    def getDeploymentCount(String ns = null) {
        return this.deployments.inNamespace(ns).list().getItems().collect { it.metadata.name }
    }

    /*
        DaemonSet Methods
    */

    def createDaemonSet(DaemonSet daemonSet) {
        ensureNamespaceExists(daemonSet.namespace)
        createDaemonSetNoWait(daemonSet)
        waitForDaemonSetAndPopulateInfo(daemonSet)
    }

    def deleteDaemonSet(DaemonSet daemonSet) {
        this.daemonsets.inNamespace(daemonSet.namespace).withName(daemonSet.name).delete()
        println "${daemonSet.name}: daemonset removed."
    }

    def waitForDaemonSetDeletion(String name, String ns = namespace) {
        Timer t = new Timer(30, 5)

        while (t.IsValid()) {
            if (this.daemonsets.inNamespace(ns).withName(name).get() == null) {
                println "Daemonset ${name} has been deleted"
                return
            }
        }
        println "Timed out waiting for daemonset ${name} to stop"
    }

    def getDaemonSetReplicaCount(DaemonSet daemonSet) {
        K8sDaemonSet d = this.daemonsets
                .inNamespace(daemonSet.namespace)
                .withName(daemonSet.name)
                .get()
        if (d != null) {
            println "${daemonSet.name}: Replicas=${d.getStatus().getDesiredNumberScheduled()}"
            return d.getStatus().getDesiredNumberScheduled()
        }
        return null
    }

    def getDaemonSetUnavailableReplicaCount(DaemonSet daemonSet) {
        K8sDaemonSet d = this.daemonsets
                .inNamespace(daemonSet.namespace)
                .withName(daemonSet.name)
                .get()
        if (d != null) {
            println "${daemonSet.name}: Unavailable Replicas=${d.getStatus().getNumberUnavailable()}"
            return d.getStatus().getNumberUnavailable() == null ? 0 : d.getStatus().getNumberUnavailable()
        }
        return null
    }

    def getDaemonSetNodeSelectors(DaemonSet daemonSet) {
        K8sDaemonSet d = this.daemonsets
                .inNamespace(daemonSet.namespace)
                .withName(daemonSet.name)
                .get()
        if (d != null) {
            println "${daemonSet.name}: Host=${d.getSpec().getTemplate().getSpec().getNodeSelector()}"
            return d.getSpec().getTemplate().getSpec().getNodeSelector()
        }
        return null
    }

    def getDaemonSetCount(String ns = null) {
        return this.daemonsets.inNamespace(ns).list().getItems().collect { it.metadata.name }
    }

    /*
        Container Methods
    */

    def deleteContainer(String containerName, String namespace = this.namespace) {
        client.pods().inNamespace(namespace).withName(containerName).delete()
    }

    def wasContainerKilled(String containerName, String namespace = this.namespace) {
        Timer t = new Timer(20, 3)

        Pod pod
        while (t.IsValid()) {
            try {
                pod = client.pods().inNamespace(namespace).withName(containerName).get()
                if (pod == null) {
                    println "Could not query K8S for pod details, assuming pod was killed"
                    return true
                }
                println "Pod Deletion Timestamp: ${pod.metadata.deletionTimestamp}"
                if (pod.metadata.deletionTimestamp != null ) {
                    return true
                }
            } catch (Exception e) {
                println "wasContainerKilled: error fetching pod details - retrying"
            }
        }
        println "wasContainerKilled: did not determine container was killed before 60s timeout"
        println "container details were found:\n${containerName}: ${pod.toString()}"
        return false
    }

    def isKubeProxyPresent() {
        PodList pods = client.pods().inAnyNamespace().list()
        return pods.getItems().findAll {
            it.getSpec().getContainers().find {
                it.getImage().contains("kube-proxy")
            }
        }
    }

    def isKubeDashboardRunning() {
        PodList pods = client.pods().inAnyNamespace().list()
        List<Pod> kubeDashboards = pods.getItems().findAll {
            it.getSpec().getContainers().find {
                it.getImage().contains("kubernetes-dashboard")
            }
        }
        return kubeDashboards.size() > 0
    }

    def getContainerlogs(Deployment deployment) {
        PodList pods = client.pods().inNamespace(deployment.namespace).list()
        Pod pod = pods.getItems().find { it.getMetadata().getName().startsWith(deployment.name) }

        try {
            println client.pods()
                    .inNamespace(pod.metadata.namespace)
                    .withName(pod.metadata.name)
                    .tailingLines(5000)
                    .watchLog(System.out)
        } catch (Exception e) {
            println "Error getting container logs: ${e.toString()}"
        }
    }

    def getStaticPodCount(String ns = null) {
        // This method assumes that a static pod name will contain the node name that the pod is running on
        def nodeNames = client.nodes().list().items.collect { it.metadata.name }
        Set<String> staticPods = [] as Set
        client.pods().inNamespace(ns).list().items.each {
            for (String node : nodeNames) {
                if (it.metadata.name.contains(node)) {
                    staticPods.add(it.metadata.name[0..it.metadata.name.indexOf(node) - 2])
                }
            }
        }
        return staticPods
    }

    /*
        Service Methods
    */

    def createService(Deployment deployment) {
        Service service = new Service(
                metadata: new ObjectMeta(
                        name: deployment.serviceName ? deployment.serviceName : deployment.name,
                        namespace: deployment.namespace,
                        labels: deployment.labels
                ),
                spec: new ServiceSpec(
                        ports: deployment.getPorts().collect {
                            k, v ->
                                new ServicePort(
                                        name: k as String,
                                        port: k as Integer,
                                        protocol: v,
                                        targetPort:
                                                new IntOrString(deployment.targetport) ?: new IntOrString(k as Integer)
                                )
                        },
                        selector: deployment.labels,
                        type: deployment.createLoadBalancer ? "LoadBalancer" : "ClusterIP"
                )
        )
        client.services().inNamespace(deployment.namespace).createOrReplace(service)
        println "${deployment.name}: Service created"
    }

    def createService(objects.Service s) {
        Service service = new Service(
                metadata: new ObjectMeta(
                        name: s.name,
                        namespace: s.namespace,
                        labels: s.labels
                ),
                spec: new ServiceSpec(
                        ports: s.getPorts().collect {
                            k, v ->
                                new ServicePort(
                                        name: k as String,
                                        port: k as Integer,
                                        protocol: v,
                                        targetPort:
                                                new IntOrString(s.targetport) ?: new IntOrString(k as Integer)
                                )
                        },
                        selector: s.labels,
                        type: s.type.toString()
                )
        )
        client.services().inNamespace(s.namespace).createOrReplace(service)
        println "${s.name}: Service created"
    }

    def deleteService(String name, String namespace = this.namespace) {
        client.services().inNamespace(namespace).withName(name).delete()
    }

    def waitForServiceDeletion(objects.Service service) {
        int waitTime = 0
        boolean beenDeleted = false

        while (waitTime < maxWaitTime && !beenDeleted) {
            Service s =
                    client.services().inNamespace(service.namespace).withName(service.name).get()
            beenDeleted = true

            println "Waiting for service ${service.name} to be deleted"
            if (s != null) {
                sleep(sleepDuration)
                waitTime += sleepDuration
                beenDeleted = false
            }
        }

        if (beenDeleted) {
            println service.name + ": service removed."
        } else {
            println "Timed out waiting for service ${service.name} to be removed"
        }
    }

    def createLoadBalancer(Deployment deployment) {
        if (deployment.createLoadBalancer) {
            int waitTime = 0
            println "Waiting for LB external IP for " + deployment.name
            while (waitTime < maxWaitTime) {
                ServiceList sList = client.services().inNamespace(deployment.namespace).list()

                for (Service service : sList.getItems()) {
                    if (service.getMetadata().getName() == deployment.name) {
                        if (service.getStatus().getLoadBalancer().getIngress() != null &&
                                service.getStatus().getLoadBalancer().getIngress().size() > 0) {
                            deployment.loadBalancerIP =
                                    service.getStatus().getLoadBalancer().getIngress().get(0).getIp() != null ?
                                            service.getStatus().getLoadBalancer().getIngress().get(0).getIp() :
                                            service.getStatus().getLoadBalancer().getIngress().get(0).getHostname()
                            println "LB IP: " + deployment.loadBalancerIP
                            waitTime += maxWaitTime
                        }
                    }
                }
                sleep(sleepDuration)
                waitTime += sleepDuration
            }
        }
    }

    /*
        Secrets Methods
    */
    def waitForSecretCreation(String secretName, String namespace = this.namespace) {
        int waitTime = 0

        while (waitTime < maxWaitTime) {
            Secret secret = client.secrets().inNamespace(namespace).withName(secretName).get()
            if (secret != null) {
                println secretName + ": secret created."
                return
            }
            sleep(sleepDuration)
            waitTime += sleepDuration
        }
        println "Timed out waiting for secret ${secretName} to be created"
    }

    String createSecret(String name, String namespace = this.namespace) {
        Map<String, String> data = new HashMap<String, String>()
        data.put("username", "YWRtaW4=")
        data.put("password", "MWYyZDFlMmU2N2Rm")

        Secret secret = new Secret(
                apiVersion: "v1",
                kind: "Secret",
                type: "Opaque",
                data: data,
                metadata: new ObjectMeta(
                        name: name
                )
        )

        try {
            Secret createdSecret = client.secrets().inNamespace(namespace).createOrReplace(secret)
            if (createdSecret != null) {
                waitForSecretCreation(name, namespace)
                return createdSecret.metadata.uid
            }
        } catch (Exception e) {
            println("Error creating secret" + e.toString())
        }
        return null
    }

    def deleteSecret(String name, String namespace = this.namespace) {
        client.secrets().inNamespace(namespace).withName(name).delete()
        sleep(sleepDuration)
        println name + ": Secret removed."
    }

    def getSecretCount(String ns = null) {
        return client.secrets().inNamespace(ns).list().getItems().findAll {
            !it.type.startsWith("kubernetes.io/service-account-token")
        }.size()
    }

    /*
        Network Policy Methods
    */

    String applyNetworkPolicy(NetworkPolicy policy) {
        io.fabric8.kubernetes.api.model.networking.NetworkPolicy networkPolicy =
                createNetworkPolicyObject(policy)

        println "${networkPolicy.metadata.name}: NetworkPolicy created:"
        println YamlGenerator.toYaml(networkPolicy)
        io.fabric8.kubernetes.api.model.networking.NetworkPolicy createdPolicy =
                client.network().networkPolicies()
                        .inNamespace(networkPolicy.metadata.namespace ?
                        networkPolicy.metadata.namespace :
                        this.namespace).createOrReplace(networkPolicy)
        policy.uid = createdPolicy.metadata.uid
        return createdPolicy.metadata.uid
    }

    boolean deleteNetworkPolicy(NetworkPolicy policy) {
        Boolean status = client.network().networkPolicies()
                .inNamespace(policy.namespace ? policy.namespace : this.namespace)
                .withName(policy.name)
                .delete()
        if (status) {
            println "${policy.name}: NetworkPolicy removed."
            return true
        }
        println "${policy.name}: Failed to remove NetworkPolicy."
        return false
    }

    def getNetworkPolicyCount(String ns) {
        return client.network().networkPolicies().inNamespace(ns).list().items.size()
    }

    def getAllNetworkPoliciesNamesByNamespace(Boolean ignoreUndoneStackroxGenerated = false) {
        Map<String, List<String>> networkPolicies = [:]
        client.network().networkPolicies().inAnyNamespace().list().items.each {
            boolean skip = false
            if (ignoreUndoneStackroxGenerated) {
                if (it.spec.podSelector.matchLabels.get("network-policies.stackrox.io/disable") == "nomatch") {
                    skip = true
                }
            }
            skip ?: networkPolicies.containsKey(it.metadata.namespace) ?
                        networkPolicies.get(it.metadata.namespace).add(it.metadata.name) :
                        networkPolicies.put(it.metadata.namespace, [it.metadata.name])
        }
        return networkPolicies
    }

    /*
        Node Methods
     */

    def getNodeCount() {
        return client.nodes().list().getItems().size()
    }

    List<Node> getNodeDetails() {
        return client.nodes().list().items.collect {
            new Node(
                    uid: it.metadata.uid,
                    name: it.metadata.name,
                    labels: it.metadata.labels,
                    annotations: it.metadata.annotations,
                    internalIps: it.status.addresses.findAll { it.type == "InternalIP" }*.address,
                    externalIps: it.status.addresses.findAll { it.type == "ExternalIP" }*.address,
                    containerRuntimeVersion: it.status.nodeInfo.containerRuntimeVersion,
                    kernelVersion: it.status.nodeInfo.kernelVersion,
                    osImage: it.status.nodeInfo.osImage
            )
        }
    }

    def supportsNetworkPolicies() {
        List<Node> gkeNodes = client.nodes().list().getItems().findAll {
            it.getStatus().getNodeInfo().getKubeletVersion().contains("gke")
        }
        return gkeNodes.size() > 0
    }

    /*
        Namespace Methods
     */

    List<objects.Namespace> getNamespaceDetails() {
        return client.namespaces().list().items.collect {
            new objects.Namespace(
                    uid: it.metadata.uid,
                    name: it.metadata.name,
                    labels: it.metadata.labels,
                    deploymentCount: getDeploymentCount(it.metadata.name) +
                            getDaemonSetCount(it.metadata.name) +
                            getStaticPodCount(it.metadata.name),
                    secretsCount: getSecretCount(it.metadata.name),
                    networkPolicyCount: getNetworkPolicyCount(it.metadata.name)
            )
        }
    }

    /*
        Service Accounts
     */

    List<ServiceAccount> getServiceAccounts() {
        def serviceAccounts = []
        client.serviceAccounts().inAnyNamespace().list().items.each {
            serviceAccounts.add(new K8sServiceAccount(
                    name: it.metadata.name,
                    namespace: it.metadata.namespace,
                    labels: it.metadata.labels ? it.metadata.labels : [:],
                    annotations: it.metadata.annotations ? it.metadata.annotations : [:],
                    secrets: it.secrets*.name,
                    imagePullSecrets: it.imagePullSecrets*.name
            ))
        }
        return serviceAccounts
    }

    def createServiceAccount(K8sServiceAccount serviceAccount) {
        ServiceAccount sa  = new ServiceAccount(
                metadata: new ObjectMeta(
                        name: serviceAccount.name,
                        namespace: serviceAccount.namespace,
                        labels: serviceAccount.labels,
                        annotations: serviceAccount.annotations
                ),
                secrets: serviceAccount.secrets,
                imagePullSecrets: serviceAccount.imagePullSecrets
        )
        client.serviceAccounts().inNamespace(sa.metadata.namespace).createOrReplace(sa)
    }

    def deleteServiceAccount(K8sServiceAccount serviceAccount) {
        client.serviceAccounts().inNamespace(serviceAccount.namespace).withName(serviceAccount.name).delete()
    }

    /*
        Roles
     */

    List<K8sRole> getRoles() {
        def roles = []
        client.rbac().roles().inAnyNamespace().list().items.each {
            roles.add(new K8sRole(
                    name: it.metadata.name,
                    namespace: it.metadata.namespace,
                    clusterRole: false,
                    labels: it.metadata.labels ? it.metadata.labels : [:],
                    annotations: it.metadata.annotations ? it.metadata.annotations : [:],
                    rules: it.rules.collect {
                        new K8sPolicyRule(
                                verbs: it.verbs,
                                apiGroups: it.apiGroups,
                                resources: it.resources,
                                nonResourceUrls: it.nonResourceURLs,
                                resourceNames: it.resourceNames
                        )
                    }
            ))
        }
        return roles
    }

    def createRole(K8sRole role) {
        Role r = new Role(
                metadata: new ObjectMeta(
                        name: role.name,
                        namespace: role.namespace,
                        labels: role.labels,
                        annotations: role.annotations
                ),
                rules: role.rules.collect {
                    new PolicyRule(
                            verbs: it.verbs,
                            apiGroups: it.apiGroups,
                            resources: it.resoures,
                            nonResourceURLs: it.nonResourceUrls,
                            resourceNames: it.resourceNames
                    )
                }
        )
        role.uid = client.rbac().roles().inNamespace(role.namespace).createOrReplace(r).metadata.uid
    }

    def deleteRole(K8sRole role) {
        client.rbac().roles().inNamespace(role.namespace).withName(role.name).delete()
    }

    /*
        RoleBindings
     */

    List<K8sRoleBinding> getRoleBindings() {
        def bindings = []
        client.rbac().roleBindings().inAnyNamespace().list().items.each {
            def b = new K8sRoleBinding(
                    new K8sRole(
                            name: it.metadata.name,
                            namespace: it.metadata.namespace,
                            clusterRole: false,
                            labels: it.metadata.labels ? it.metadata.labels : [:],
                            annotations: it.metadata.annotations ? it.metadata.annotations : [:]
                    ),
                    it.subjects.collect { new K8sSubject(kind: it.kind, name: it.name, namespace: it.namespace) }
            )
            def uid = client.rbac().clusterRoles().withName(it.roleRef.name).get()?.metadata?.uid ?:
                    client.rbac().roles()
                            .inNamespace(it.metadata.namespace)
                            .withName(it.roleRef.name).get()?.metadata?.uid
            b.roleRef.uid = uid ?: ""
            bindings.add(b)
        }
        return bindings
    }

    def createRoleBinding(K8sRoleBinding roleBinding) {
        RoleBinding r = new RoleBinding(
                metadata: new ObjectMeta(
                        name: roleBinding.name,
                        namespace: roleBinding.namespace,
                        labels: roleBinding.labels,
                        annotations: roleBinding.annotations
                ),
                subjects: roleBinding.subjects.collect {
                    new Subject(kind: it.kind, name: it.name, namespace: it.namespace)
                },
                roleRef: new RoleRef(
                        name: roleBinding.roleRef.name,
                        kind: roleBinding.roleRef.clusterRole ? "ClusterRole" : "Role"
                )
        )
        client.rbac().roleBindings().inNamespace(roleBinding.namespace).createOrReplace(r)
    }

    def deleteRoleBinding(K8sRoleBinding roleBinding) {
        client.rbac().roleBindings()
                .inNamespace(roleBinding.namespace)
                .withName(roleBinding.name)
                .delete()
    }

    /*
        ClusterRoles
     */

    List<K8sRole> getClusterRoles() {
        def clusterRoles = []
        client.rbac().clusterRoles().inAnyNamespace().list().items.each {
            clusterRoles.add(new K8sRole(
                    name: it.metadata.name,
                    namespace: "",
                    clusterRole: true,
                    labels: it.metadata.labels ? it.metadata.labels : [:],
                    annotations: it.metadata.annotations ? it.metadata.annotations : [:],
                    rules: it.rules.collect {
                        new K8sPolicyRule(
                                verbs: it.verbs,
                                apiGroups: it.apiGroups,
                                resources: it.resources,
                                nonResourceUrls: it.nonResourceURLs,
                                resourceNames: it.resourceNames
                        )
                    }
            ))
        }
        return clusterRoles
    }

    def createClusterRole(K8sRole role) {
        ClusterRole r = new ClusterRole(
                metadata: new ObjectMeta(
                        name: role.name,
                        labels: role.labels,
                        annotations: role.annotations
                ),
                rules: role.rules.collect {
                    new PolicyRule(
                            verbs: it.verbs,
                            apiGroups: it.apiGroups,
                            resources: it.resoures,
                            nonResourceURLs: it.nonResourceUrls,
                            resourceNames: it.resourceNames
                    )
                }
        )
        role.uid = client.rbac().clusterRoles().createOrReplace(r).metadata.uid
    }

    def deleteClusterRole(K8sRole role) {
        client.rbac().clusterRoles().withName(role.name).delete()
    }

    /*
        ClusterRoleBindings
     */

    List<K8sRoleBinding> getClusterRoleBindings() {
        def clusterBindings = []
        client.rbac().clusterRoleBindings().inAnyNamespace().list().items.each {
            def b = new K8sRoleBinding(
                    new K8sRole(
                            name: it.metadata.name,
                            namespace: "",
                            clusterRole: true,
                            labels: it.metadata.labels ? it.metadata.labels : [:],
                            annotations: it.metadata.annotations ? it.metadata.annotations : [:]
                    ),
                    it.subjects.collect { new K8sSubject(kind: it.kind, name: it.name, namespace: it.namespace) }
            )
            def uid = client.rbac().clusterRoles().withName(it.roleRef.name).get()?.metadata?.uid ?:
                    client.rbac().roles()
                            .inNamespace(it.metadata.namespace)
                            .withName(it.roleRef.name).get()?.metadata?.uid
            b.roleRef.uid = uid ?: ""
            clusterBindings.add(b)
        }
        return clusterBindings
    }

    def createClusterRoleBinding(K8sRoleBinding roleBinding) {
        ClusterRoleBinding r = new ClusterRoleBinding(
                metadata: new ObjectMeta(
                        name: roleBinding.name,
                        labels: roleBinding.labels,
                        annotations: roleBinding.annotations
                ),
                subjects: roleBinding.subjects.collect {
                    new Subject(kind: it.kind, name: it.name, namespace: it.namespace)
                },
                roleRef: new RoleRef(
                        name: roleBinding.roleRef.name,
                        kind: roleBinding.roleRef.clusterRole ? "ClusterRole" : "Role"
                )
        )
        client.rbac().clusterRoleBindings().createOrReplace(r)
    }

    def deleteClusterRoleBinding(K8sRoleBinding roleBinding) {
        client.rbac().clusterRoleBindings().withName(roleBinding.name).delete()
    }

    /*
        Misc/Helper Methods
    */

    def execInContainer(Deployment deployment, String cmd) {
        ScheduledExecutorService executorService = Executors.newScheduledThreadPool(20)
        try {
            CountDownLatch latch = new CountDownLatch(1)
            ExecWatch watch =
                    client.pods().inNamespace(deployment.namespace).withName(deployment.pods.get(0).name)
                            .redirectingOutput().usingListener(new ExecListener() {
            @Override
            void onOpen(Response response) {
            }

            @Override
            void onFailure(Throwable t, Response response) {
                latch.countDown()
            }

            @Override
            void onClose(int code, String reason) {
                latch.countDown()
            }
            }).exec(cmd.split(" "))
            BlockingInputStreamPumper pump = new BlockingInputStreamPumper(watch.getOutput(), new SystemOutCallback())
            Future<String> outPumpFuture = executorService.submit(pump, "Done")
            executorService.scheduleAtFixedRate(new FutureChecker("Exec", cmd, outPumpFuture), 0, 2, TimeUnit.SECONDS)

            latch.await(30, TimeUnit.SECONDS)
            watch.close()
            pump.close()
        } catch (Exception e) {
            println "Error exec'ing in pod: ${e.toString()}"
            return false
        }
        executorService.shutdown()
        return true
    }

    def createClairifyDeployment() {
        //create clairify service
        Service clairifyService = new Service(
                apiVersion: "v1",
                metadata: new ObjectMeta(
                        name: "clairify",
                        namespace: "stackrox"
                ),
                spec: new ServiceSpec(
                        ports: [
                                new ServicePort(
                                        name: "http-clair",
                                        port: 6060,
                                        targetPort: new IntOrString(6060)
                                ),
                                new ServicePort(
                                        name: "http-clairify",
                                        port: 8080,
                                        targetPort: new IntOrString(8080)
                                )
                        ],
                        type: "ClusterIP",
                        selector: ["app":"clairify"]
                )
        )
        client.services().inNamespace("stackrox").createOrReplace(clairifyService)

        //create clairify deployment
        Container clairifyContainer = new Container(
                name: "clairify",
                image: "stackrox/clairify:0.5.3",
                env: [new EnvVar(
                        name: "CLAIR_ARGS",
                        value: "-insecure-tls")
                ],
                command: ["/init", "/clairify"],
                imagePullPolicy: "Always",
                ports: [new ContainerPort(containerPort: 6060, name: "clair-http"),
                        new ContainerPort(containerPort: 8080, name: "clairify-http")
                ],
                securityContext: new SecurityContext(
                        capabilities: new Capabilities(
                                drop: ["NET_RAW"]
                        )
                )
        )

        K8sDeployment clairifyDeployment =
                new K8sDeployment(
                        metadata: new ObjectMeta(
                                name: "clairify",
                                namespace: "stackrox",
                                labels: ["app":"clairify"],
                                annotations: ["owner":"stackrox", "email":"support@stackrox.com"]
                        ),
                        spec: new DeploymentSpec(
                                replicas: 1,
                                minReadySeconds: 15,
                                selector: new LabelSelector(
                                        matchLabels: ["app":"clairify"]
                                ),
                                template: new PodTemplateSpec(
                                        metadata: new ObjectMeta(
                                                namespace: "stackrox",
                                                labels: ["app":"clairify"]
                                        ),
                                        spec: new PodSpec(
                                                containers: [clairifyContainer],
                                                imagePullSecrets: [new LocalObjectReference(
                                                        name: "stackrox"
                                                )]
                                        )
                                )
                        )
                )
        this.deployments.inNamespace("stackrox").createOrReplace(clairifyDeployment)
        waitForDeploymentCreation("clairify", "stackrox")
    }

    String getClairifyEndpoint() {
        return "clairify.stackrox:8080"
    }

    String generateYaml(Object orchestratorObject) {
        if (orchestratorObject instanceof NetworkPolicy) {
            return YamlGenerator.toYaml(createNetworkPolicyObject(orchestratorObject))
        }

        return ""
    }

    String getNameSpace() {
        return this.namespace
    }

    String getSensorContainerName() {
        return client.pods().inNamespace("stackrox").list().items.find {
            it.metadata.name.startsWith("sensor")
        }.metadata.name
    }

    def waitForSensor() {
        def start = System.currentTimeMillis()
        def running = client.apps().deployments()
                .inNamespace("stackrox")
                .withName("sensor")
                .get().status.readyReplicas < 1
        while (!running && (System.currentTimeMillis() - start) < 30000) {
            println "waiting for sensor to come back online. Trying again in 1s..."
            sleep 1000
            running = client.apps().deployments()
                    .inNamespace("stackrox")
                    .withName("sensor")
                    .get().status.readyReplicas < 1
        }
        if (!running) {
            println "Failed to detect sensor came back up within 30s... Future tests may be impacted."
        }
    }

    /*
        Private K8S Support functions
    */

    def createDeploymentNoWait(Deployment deployment) {
        deployment.getNamespace() != null ?: deployment.setNamespace(this.namespace)

        // Create service if needed
        if (deployment.exposeAsService) {
            createService(deployment)
            createLoadBalancer(deployment)
        }

        K8sDeployment d = new K8sDeployment(
                metadata: new ObjectMeta(
                        name: deployment.name,
                        namespace: deployment.namespace,
                        labels: deployment.labels
                ),
                spec: new DeploymentSpec(
                        selector: new LabelSelector(null, deployment.labels),
                        replicas: deployment.replicas,
                        minReadySeconds: 15,
                        template: new PodTemplateSpec(
                                metadata: new ObjectMeta(
                                        name: deployment.name,
                                        namespace: deployment.namespace,
                                        labels: deployment.labels,
                                        annotations: deployment.annotation
                                ),
                                spec: generatePodSpec(deployment)
                        )

                )
        )

        try {
            client.apps().deployments().inNamespace(deployment.namespace).createOrReplace(d)
            println("Told the orchestrator to create " + deployment.getName())
        } catch (Exception e) {
            println("Error creating k8s deployment: " + e.toString())
        }
    }

    def waitForDeploymentAndPopulateInfo(Deployment deployment) {
        try {
            deployment.deploymentUid = waitForDeploymentCreation(
                    deployment.getName(),
                    deployment.getNamespace(),
                    deployment.skipReplicaWait
            )
            updateDeploymentDetails(deployment)
        } catch (Exception e) {
            println("Error while waiting for deployment/populating deployment info: " + e.toString())
        }
    }

    def waitForDeploymentCreation(String deploymentName, String namespace, Boolean skipReplicaWait = false) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            println "Waiting for ${deploymentName} to start"
            K8sDeployment d = this.deployments.inNamespace(namespace).withName(deploymentName).get()
            getAndPrintPods(namespace, deploymentName)
            if (d == null) {
                println "${deploymentName} not found yet"
                continue
            } else if (skipReplicaWait) {
                // If skipReplicaWait is set, we still want to sleep for a few seconds to allow the deployment
                // to work its way through the system.
                sleep(sleepDuration)
                println "${deploymentName}: deployment created (skipped replica wait)."
                return
            }
            if (d.getStatus().getReadyReplicas() == d.getSpec().getReplicas()) {
                println "All ${d.getSpec().getReplicas()} replicas found " +
                        "in ready state for ${deploymentName}"
                println "Took ${t.SecondsSince()} seconds for k8s deployment ${deploymentName}"
                return d.getMetadata().getUid()
            }
            println "${d.getStatus().getReadyReplicas()}/" +
                    "${d.getSpec().getReplicas()} are in the ready state for ${deploymentName}"
        }
    }

    def createDaemonSetNoWait(DaemonSet daemonSet) {
        daemonSet.getNamespace() != null ?: daemonSet.setNamespace(this.namespace)

        K8sDaemonSet ds = new K8sDaemonSet(
                metadata: new ObjectMeta(
                        name: daemonSet.name,
                        namespace: daemonSet.namespace,
                        labels: daemonSet.labels
                ),
                spec: new DaemonSetSpec(
                        minReadySeconds: 15,
                        selector: new LabelSelector(null, daemonSet.labels),
                        template: new PodTemplateSpec(
                                metadata: new ObjectMeta(
                                        name: daemonSet.name,
                                        namespace: daemonSet.namespace,
                                        labels: daemonSet.labels
                                ),
                                spec: generatePodSpec(daemonSet)
                        )
                )
        )

        try {
            this.daemonsets.inNamespace(daemonSet.namespace).createOrReplace(ds)
            println("Told the orchestrator to create " + daemonSet.getName())
        } catch (Exception e) {
            println("Error creating k8s deployment" + e.toString())
        }
    }

    def waitForDaemonSetAndPopulateInfo(DaemonSet daemonSet) {
        try {
            daemonSet.deploymentUid = waitForDaemonSetCreation(
                    daemonSet.getName(),
                    daemonSet.getNamespace(),
                    daemonSet.skipReplicaWait
            )
            updateDeploymentDetails(daemonSet)
        } catch (Exception e) {
            println("Error while waiting for daemonset/populating daemonset info: " + e.toString())
        }
    }

    def waitForDaemonSetCreation(String name, String namespace, Boolean skipReplicaWait = false) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            println "Waiting for ${name} to start"
            K8sDaemonSet d = this.daemonsets.inNamespace(namespace).withName(name).get()
            getAndPrintPods(namespace, name)
            if (d == null) {
                println "${name} not found yet"
                continue
            } else if (skipReplicaWait) {
                // If skipReplicaWait is set, we still want to sleep for a few seconds to allow the deployment
                // to work its way through the system.
                sleep(sleepDuration)
                println "${name}: daemonset created (skipped replica wait)."
                return
            }
            if (d.getStatus().getCurrentNumberScheduled() == d.getStatus().getDesiredNumberScheduled()) {
                println "All ${d.getStatus().getDesiredNumberScheduled()} replicas found in ready state for ${name}"
                return d.getMetadata().getUid()
            }
            println "${d.getStatus().getCurrentNumberScheduled()}/" +
                    "${d.getStatus().getDesiredNumberScheduled()} are in the ready state for ${name}"
        }
    }

    def generatePodSpec(Deployment deployment) {
        List<ContainerPort> depPorts = deployment.ports.collect {
            k, v -> new ContainerPort(
                    k as Integer,
                    null,
                    null,
                    "port" + (k as String),
                    v as String
            )
        }

        List<LocalObjectReference> imagePullSecrets = new LinkedList<>()
        for (String str : deployment.getImagePullSecret()) {
            LocalObjectReference obj = new LocalObjectReference(name: str)
            imagePullSecrets.add(obj)
        }

        List<EnvVar> envVars = deployment.env.collect {
            k, v -> new EnvVar(k, v, null)
        }

        List<Volume> volumes = []
        deployment.volumes.each {
            k, v -> Volume vol = new Volume(
                    name: k,
                    hostPath: v ? new HostPathVolumeSource(
                            path: v,
                            type: "Directory") :
                            null,
                    secret: deployment.secretNames.get(k) ?
                            new SecretVolumeSource(secretName: deployment.secretNames.get(k)) :
                            null
            )
            volumes.add(vol)
        }

        List<VolumeMount> volMounts = []
        deployment.volumeMounts.each {
            k, v -> VolumeMount volMount = new VolumeMount(
                    mountPath: v,
                    name: k
            )
            volMounts.add(volMount)
        }

        Map<String , Quantity> limits = new HashMap<>()
        for (String key:deployment.limits.keySet()) {
            Quantity quantity = new Quantity(deployment.limits.get(key))
            limits.put(key, quantity)
        }

        Map<String , Quantity> requests = new HashMap<>()
        for (String key:deployment.request.keySet()) {
            Quantity quantity = new Quantity(deployment.request.get(key))
            requests.put(key, quantity)
        }

        Container container = new Container(
                name: deployment.name,
                image: deployment.image,
                command: deployment.command,
                args: deployment.args,
                ports: depPorts,
                volumeMounts: volMounts,
                env: envVars,
                resources: new ResourceRequirements(limits, requests),
                securityContext: new SecurityContext(privileged: deployment.isPrivileged)
        )

        PodSpec podSpec = new PodSpec(
                containers: [container],
                volumes: volumes,
                imagePullSecrets: imagePullSecrets,
                hostNetwork: deployment.hostNetwork,
                serviceAccountName: deployment.serviceAccountName
        )
        return podSpec
    }

    def updateDeploymentDetails(Deployment deployment) {
        // Filtering pod query by using the "name=<name>" because it should always be present in the deployment
        // object - IF this is ever missing, it may cause problems fetching pod details
        def deployedPods = client.pods().inNamespace(deployment.namespace).withLabel("name", deployment.name).list()
        for (Pod pod : deployedPods.getItems()) {
            deployment.addPod(
                    pod.getMetadata().getName(),
                    pod.getMetadata().getUid(),
                    pod.getStatus().getContainerStatuses() != null ?
                            pod.getStatus().getContainerStatuses().stream().map {
                                container -> container.getContainerID()
                            }.collect(Collectors.toList()) :
                            [],
                    pod.getStatus().getPodIP()
            )
        }
    }

    private io.fabric8.kubernetes.api.model.networking.NetworkPolicy createNetworkPolicyObject(NetworkPolicy policy) {
        def networkPolicy = new NetworkPolicyBuilder()
                .withApiVersion("networking.k8s.io/v1")
                .withKind("NetworkPolicy")
                .withNewMetadata()
                .withName(policy.name)

        if (policy.namespace) {
            networkPolicy.withNamespace(policy.namespace)
        }

        if (policy.labels != null) {
            networkPolicy.withLabels(policy.labels)
        }

        networkPolicy = networkPolicy.endMetadata().withNewSpec()

        if (policy.metadataPodSelector != null) {
            networkPolicy.withNewPodSelector().withMatchLabels(policy.metadataPodSelector).endPodSelector()
        }

        if (policy.types != null) {
            def polTypes = []
            for (NetworkPolicyTypes type : policy.types) {
                polTypes.add(type.toString())
            }
            networkPolicy.withPolicyTypes(polTypes)
        }

        if (policy.ingressPodSelector != null) {
            networkPolicy.withIngress(
                    new NetworkPolicyIngressRuleBuilder().withFrom(
                            new NetworkPolicyPeerBuilder()
                                    .withNewPodSelector()
                                    .withMatchLabels(policy.ingressPodSelector)
                                    .endPodSelector().build()
                    ).build()
            )
        }

        if (policy.egressPodSelector != null) {
            networkPolicy.withEgress(
                    new NetworkPolicyEgressRuleBuilder().withTo(
                            new NetworkPolicyPeerBuilder()
                                    .withNewPodSelector()
                                    .withMatchLabels(policy.egressPodSelector)
                                    .endPodSelector().build()
                    ).build()
            )
        }

        if (policy.ingressNamespaceSelector != null) {
            networkPolicy.withIngress(
                    new NetworkPolicyIngressRuleBuilder().withFrom(
                            new NetworkPolicyPeerBuilder()
                                    .withNewNamespaceSelector()
                                    .withMatchLabels(policy.ingressNamespaceSelector)
                                    .endNamespaceSelector().build()
                    ).build()
            )
        }

        if (policy.egressNamespaceSelector != null) {
            networkPolicy.withEgress(
                    new NetworkPolicyEgressRuleBuilder().withTo(
                            new NetworkPolicyPeerBuilder()
                                    .withNewNamespaceSelector()
                                    .withMatchLabels(policy.egressNamespaceSelector)
                                    .endNamespaceSelector().build()
                    ).build()
            )
        }

        return networkPolicy.endSpec().build()
    }

    V1beta1ValidatingWebhookConfiguration getAdmissionController() {
        println "get admission controllers stub"
    }

    def deleteAdmissionController(String name) {
        println "delete admission controllers stub: ${name}"
    }

    def createAdmissionController(V1beta1ValidatingWebhookConfiguration config) {
        println "create admission controllers stub: ${config}"
    }

    String createNamespace(String ns) {
        Namespace namespace = new Namespace("v1", null, new ObjectMeta(name: ns), null, null)
        return client.namespaces().createOrReplace(namespace).metadata.getUid()
    }

    def deleteNamespace(String ns) {
        client.namespaces().withName(ns).delete()
    }

    def waitForNamespaceDeletion(String ns, int iterations = 20, int intervalSeconds = 3) {
        println "Waiting for namespace ${ns} to be deleted"
        Timer t = new Timer(iterations, intervalSeconds)
        while (t.IsValid()) {
            if (client.namespaces().withName(ns).get() == null ) {
                println "K8s found that namespace ${ns} was deleted"
                return true
            }
            println "Retrying in ${intervalSeconds}..."
        }
        println "K8s did not detect that namespace ${ns} was deleted"
        return false
    }

    private static class SystemOutCallback implements Callback<byte[]> {
        @Override
        void call(byte[] data) {
            System.out.print(new String(data))
        }
    }

    private static class FutureChecker implements Runnable {
        private final String name
        private final String cmd
        private final Future<String> future

        private FutureChecker(String name, String cmd, Future<String> future) {
            this.name = name
            this.cmd = cmd
            this.future = future
        }

        @Override
        void run() {
            if (!future.isDone()) {
                System.out.println(name + ":[" + cmd + "] is not done yet")
            }
        }
    }
}
