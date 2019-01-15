package orchestratormanager

import common.YamlGenerator
import io.fabric8.kubernetes.api.model.Capabilities
import io.fabric8.kubernetes.api.model.Container
import io.fabric8.kubernetes.api.model.ContainerPort
import io.fabric8.kubernetes.api.model.EnvVar
import io.fabric8.kubernetes.api.model.IntOrString
import io.fabric8.kubernetes.api.model.LabelSelector
import io.fabric8.kubernetes.api.model.LocalObjectReference
import io.fabric8.kubernetes.api.model.ObjectMeta
import io.fabric8.kubernetes.api.model.Pod
import io.fabric8.kubernetes.api.model.PodList
import io.fabric8.kubernetes.api.model.PodSpec
import io.fabric8.kubernetes.api.model.ResourceRequirements
import io.fabric8.kubernetes.api.model.Secret
import io.fabric8.kubernetes.api.model.SecretList
import io.fabric8.kubernetes.api.model.SecretVolumeSource
import io.fabric8.kubernetes.api.model.SecurityContext
import io.fabric8.kubernetes.api.model.ServiceAccountBuilder
import io.fabric8.kubernetes.api.model.ServiceBuilder
import io.fabric8.kubernetes.api.model.ServicePort
import io.fabric8.kubernetes.api.model.Volume
import io.fabric8.kubernetes.api.model.VolumeMount
import io.fabric8.kubernetes.api.model.apps.DaemonSetBuilder
import io.fabric8.kubernetes.api.model.apps.DaemonSetList
import io.fabric8.kubernetes.api.model.apps.DeploymentBuilder
import io.fabric8.kubernetes.api.model.networking.NetworkPolicyBuilder
import io.fabric8.kubernetes.api.model.networking.NetworkPolicyEgressRuleBuilder
import io.fabric8.kubernetes.api.model.networking.NetworkPolicyIngressRuleBuilder
import io.fabric8.kubernetes.api.model.networking.NetworkPolicyPeerBuilder
import io.fabric8.kubernetes.client.KubernetesClientException
import io.fabric8.openshift.api.model.ProjectRequest
import io.fabric8.openshift.api.model.ProjectRequestBuilder
import io.fabric8.openshift.api.model.Route
import io.fabric8.openshift.api.model.RouteList
import io.fabric8.openshift.api.model.RouteSpec
import io.fabric8.openshift.api.model.RouteTargetReference
import io.fabric8.openshift.api.model.RunAsUserStrategyOptions
import io.fabric8.openshift.api.model.SELinuxContextStrategyOptions
import io.fabric8.openshift.api.model.SecurityContextConstraints
import io.fabric8.openshift.api.model.SecurityContextConstraintsBuilder
import io.fabric8.openshift.client.DefaultOpenShiftClient
import io.fabric8.openshift.client.OpenShiftClient
import io.fabric8.kubernetes.api.model.Quantity
import io.kubernetes.client.models.V1beta1ValidatingWebhookConfiguration
import objects.DaemonSet
import objects.Deployment
import objects.NetworkPolicy
import objects.NetworkPolicyTypes

import java.util.stream.Collectors

class OpenShift extends OrchestratorCommon implements OrchestratorMain {
    private final String namespace
    private final int sleepDuration = 5000
    private final int maxWaitTime = 90000

    final private OpenShiftClient osClient

    OpenShift(String ns) {
        this.namespace = ns

        osClient = new DefaultOpenShiftClient()
        ensureNamespaceExists(this.namespace)
    }

    def ensureNamespaceExists(String ns) {
        ProjectRequest projectRequest = new ProjectRequestBuilder()
                .withNewMetadata()
                .withName(ns)
                .addToLabels("project", ns)
                .endMetadata()
                .build()

        try {
            osClient.projectrequests().create(projectRequest)
            println "Created namespace ${ns}"
        } catch (KubernetesClientException kce) {
            // 409 is already exists
            if (kce.code != 409) {
                throw kce
            }
        }

        try {
            SecurityContextConstraints anyuid = osClient.securityContextConstraints().withName("anyuid").get()
            if (anyuid != null && !anyuid.users.contains("system:serviceaccount:" + ns + ":default")) {
                println "Adding system:serviceaccount:" + ns + ":default to anyuid user list"
                anyuid.users.addAll(["system:serviceaccount:" + ns + ":default"])
                osClient.securityContextConstraints().createOrReplace(anyuid)
            }
        } catch (Exception e) {
            println e.toString()
        }
    }

    @Override
    def setup() {
    }

    @Override
    def cleanup() {
    }

    /*
        Deployment Methods
    */

    def createDeployment(Deployment deployment) {
        createDeploymentNoWait(deployment)
        waitForDeploymentAndPopulateInfo(deployment)
    }

    def batchCreateDeployments(List<Deployment> deployments) {
        for (Deployment deployment: deployments) {
            createDeploymentNoWait(deployment)
        }
        for (Deployment deployment: deployments) {
            waitForDeploymentAndPopulateInfo(deployment)
        }
    }

    def waitForDeploymentDeletion(Deployment deploy) {
        int waitTime = 0
        boolean beenDeleted = false
        String deploymentName = deploy.getName()
        String namespace = this.getNameSpace()

        while (waitTime < maxWaitTime && !beenDeleted) {
            def dList = osClient.apps().deployments().inNamespace(namespace).list()
            beenDeleted = true

            println "Waiting for " + deploymentName + " to be deleted"
            for (io.fabric8.kubernetes.api.model.apps.Deployment deployment : dList.getItems()) {
                if (deployment.getMetadata().getName() == deploymentName) {
                    sleep(sleepDuration)
                    waitTime += sleepDuration
                    beenDeleted = false
                    break
                }
            }
        }
        if (beenDeleted) {
            println deploymentName + ": deployment removed."
        } else {
            println "Timed out waiting for " + deploymentName
        }
    }

    def deleteDeployment(Deployment deployment) {
        if (deployment.exposeAsService) {
            this.deleteService(deployment.name, deployment.namespace)
        }
        osClient.apps().deployments().inNamespace(deployment.namespace).withName(deployment.name).delete()

        println "removing the deployment:" + deployment.name
    }

    String getDeploymentId(Deployment deployment) {
        return osClient.apps().deployments()
                .inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get().metadata.uid
    }

    def getDeploymentReplicaCount(Deployment deployment) {
        io.fabric8.kubernetes.api.model.apps.Deployment d = osClient.apps().deployments()
                .inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get()
        if (d != null) {
            println "${deployment.name}: Replicas=${d.getSpec().getReplicas()}"
            return d.getSpec().getReplicas()
        }
    }

    def getDeploymentUnavailableReplicaCount(Deployment deployment) {
        io.fabric8.kubernetes.api.model.apps.Deployment d = osClient.apps().deployments()
                .inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get()
        if (d != null) {
            println "${deployment.name}: Unavailable Replicas=${d.getStatus().getUnavailableReplicas()}"
            return d.getStatus().getUnavailableReplicas()
        }
    }

    def getDeploymentNodeSelectors(Deployment deployment) {
        io.fabric8.kubernetes.api.model.apps.Deployment d = osClient.apps().deployments()
                .inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get()
        if (d != null) {
            println "${deployment.name}: Host=${d.getSpec().getTemplate().getSpec().getNodeSelector()}"
            return d.getSpec().getTemplate().getSpec().getNodeSelector()
        }
    }

    def getDeploymentCount() {
        return osClient.deploymentConfigs().inAnyNamespace().list().getItems().size() +
                osClient.apps().deployments().inAnyNamespace().list().getItems().size()
    }
    Set<String> getDeploymentSecrets(Deployment deployment) {
        Set<String> secretSet = [] as Set
        io.fabric8.kubernetes.api.model.apps.Deployment d = osClient.apps().deployments()
                .inNamespace(deployment.namespace)
                .withName(deployment.name)
                .get()
        if (d != null) {
            List<Volume> volumeList = d.getSpec().getTemplate().getSpec().getVolumes()
            for (Volume volume : volumeList) {
                secretSet.add(volume.getSecret().getSecretName())
            }
        }
        return secretSet
    }

    /*
        DaemonSet Methods
    */

    def createDaemonSet(DaemonSet daemonSet) {
        createDaemonSetNoWait(daemonSet)
        waitForDaemonSetAndPopulateInfo(daemonSet)
    }

    def deleteDaemonSet(DaemonSet daemonSet) {
        osClient.apps().daemonSets().inNamespace(daemonSet.namespace).withName(daemonSet.name).delete()
        println daemonSet.name + ": daemonset removed."
    }

    def waitForDaemonSetDeletion(String name, String ns = namespace) {
        int waitTime = 0

        while (waitTime < maxWaitTime) {
            io.fabric8.kubernetes.api.model.apps.DaemonSet ds =
                    osClient.apps().daemonSets().inNamespace(ns).withName(name).get()
            if (ds == null) {
                return
            }

            sleep(sleepDuration)
            waitTime += sleepDuration
        }

        println "Timed out waiting for daemonset ${name} to stop"
    }

    def getDaemonSetReplicaCount(DaemonSet daemonSet) {
        io.fabric8.kubernetes.api.model.apps.DaemonSet d = osClient.apps().daemonSets()
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
        io.fabric8.kubernetes.api.model.apps.DaemonSet d = osClient.apps().daemonSets()
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
        io.fabric8.kubernetes.api.model.apps.DaemonSet d = osClient.apps().daemonSets()
                .inNamespace(daemonSet.namespace)
                .withName(daemonSet.name)
                .get()
        if (d != null) {
            println "${daemonSet.name}: Host=${d.getSpec().getTemplate().getSpec().getNodeSelector()}"
            return d.getSpec().getTemplate().getSpec().getNodeSelector()
        }
        return null
    }

    def getDaemonSetCount() {
        return osClient.apps().daemonSets().inAnyNamespace().list().getItems().size()
    }

    /*
        Container Methods
    */

    def wasContainerKilled(String containerName, String namespace = this.namespace) {
        Long startTime = System.currentTimeMillis()
        PodList pods = new PodList()

        while (System.currentTimeMillis() - startTime < 60000) {
            try {
                pods = osClient.pods().inNamespace(namespace).withField("metadata.name", containerName).list()
                if (pods.items.size() == 0) {
                    println "Could not query K8S for pod details, assuming pod was killed"
                    return true
                }
            } catch (Exception e) {
                println "wasContainerKilled: error fetching pod details - retrying"
            }
        }

        println "wasContainerKilled: did not determine container was killed before 60s timeout"
        println "container details were found:\n${containerName}: ${pods.getItems().get(0).toString()}"
        return false
    }

    def isKubeProxyPresent() {
        PodList pods = osClient.pods().inAnyNamespace().list()
        return pods.getItems().findAll {
            it.getSpec().getContainers().find {
                it.getImage().contains("kube-proxy")
            }
        }
    }

    def isKubeDashboardRunning() {
        PodList pods = osClient.pods().inAnyNamespace().list()
        List<Pod> kubeDashboards = pods.getItems().findAll {
            it.getSpec().getContainers().find {
                it.getImage().contains("kubernetes-dashboard")
            }
        }
        return kubeDashboards.size() > 0
    }

    def getContainerlogs(Deployment deployment) {
        println "get logs not supported on openshift: ${deployment.name}"
    }

    /*
        Service Methods
    */

    def createService(Deployment deployment) {
        def service = osClient.services().inNamespace(deployment.namespace).createOrReplaceWithNew()
                .withNewMetadata()
                        .withName(deployment.name)
                        .withName(deployment.serviceName ? deployment.serviceName : deployment.name)
                        .withNamespace(deployment.namespace)
                        .withLabels(deployment.labels)
                .endMetadata()
                .withNewSpec()
                        .withSelector(deployment.labels)
                        .withType(deployment.createLoadBalancer ? "LoadBalancer" : "ClusterIP")

        deployment.ports.each {
            k, v -> service.addNewPort()
                    .withName(k as String)
                    .withPort(k as Integer)
                    .withProtocol(v)
                    .withTargetPort(new IntOrString(deployment.targetport))
                    .endPort()
        }

        service.endSpec().done()
        println "${deployment.name}: Service created"
    }

    def deleteService(String name, String namespace = this.namespace) {
        osClient.services().inNamespace(namespace).withName(name).delete()
    }

    def createLoadBalancer(Deployment deployment) {
        if (deployment.createLoadBalancer) {
            Route route = new Route(
                    "v1",
                    "Route",
                    new ObjectMeta(name: deployment.name),
                    new RouteSpec(to: new RouteTargetReference("Service", deployment.name, null)),
                    null
            )
            osClient.routes().inNamespace(deployment.namespace).createOrReplace(route)
            int waitTime = 0
            println "Waiting for Route for " + deployment.name
            while (waitTime < maxWaitTime) {
                RouteList rList
                rList = osClient.routes().inNamespace(deployment.namespace).list()

                for (Route r : rList.getItems()) {
                    if (r.getMetadata().getName() == deployment.name) {
                        if (r.getStatus().getIngress() != null) {
                            println "Route Host: " +
                                    r.getStatus().getIngress().get(0).getHost()
                            deployment.loadBalancerIP =
                                    r.getStatus().getIngress().get(0).getHost()
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
            Secret secret = osClient.secrets().inNamespace(namespace).withName(secretName).get()
            if (secret != null) {
                println secretName + ": secret created."
                return
            }
            sleep(sleepDuration)
            waitTime += sleepDuration
        }
        println "Timed out waiting for " + secretName
    }
    String createSecret(String name) {
        Map<String, String> data = new HashMap<String, String>()
        data.put("username", "YWRtaW4=")
        data.put("password", "MWYyZDFlMmU2N2Rm")

        try {
            Secret createdSecret = osClient.secrets().inNamespace(this.namespace).createOrReplaceWithNew()
                    .withApiVersion("v1")
                    .withKind("Secret")
                    .withNewMetadata()
                        .withName(name)
                        .withNamespace(this.namespace)
                    .endMetadata()
                    .withType("Opaque")
                    .withData(data)
                    .done()
            waitForSecretCreation(name, this.namespace)
            return createdSecret.metadata.uid
        } catch (Exception e) {
            println("Error creating openshift secret: " + e.toString())
        }

        return null
    }

    def deleteSecret(String name, String namespace = this.namespace) {
        osClient.secrets().inNamespace(namespace).withName(name).delete()
        sleep(sleepDuration)
        println name + ": Secret removed."
    }

    def getSecretCount() {
        SecretList secrets =  osClient.secrets().inAnyNamespace().list()
        return secrets.getItems().findAll {
            !it.type.startsWith("kubernetes.io/docker") &&
                    !it.type.startsWith("kubernetes.io/service-account-token")
        }.size()
    }

    /*
        Network Policy Methods
    */

    String applyNetworkPolicy(NetworkPolicy policy) {
        io.fabric8.kubernetes.api.model.networking.NetworkPolicy networkPolicy = createNetworkPolicyObject(policy)

        println "${networkPolicy.metadata.name}: NetworkPolicy created:"
        println YamlGenerator.toYaml(networkPolicy)
        io.fabric8.kubernetes.api.model.networking.NetworkPolicy createdPolicy =
                osClient.extensions().networkPolicies().inNamespace(policy.namespace).createOrReplace(networkPolicy)
        policy.uid = createdPolicy.metadata.uid
        return createdPolicy.metadata.uid
    }

    boolean deleteNetworkPolicy(NetworkPolicy policy) {
        Boolean success = osClient.extensions()
                .networkPolicies().
                inNamespace(policy.namespace)
                .withName(policy.name)
                .delete()
        if (success) {
            println "${policy.name}: NetworkPolicy removed."
            return true
        }

        println "${policy.name}: Failed to remove NetworkPolicy."
        return false
    }

    /*
        Node Methods
     */

    def getNodeCount() {
        return osClient.nodes().list().getItems().size()
    }

    def supportsNetworkPolicies() {
        List<Node> gkeNodes = osClient.nodes().list().getItems().findAll {
            it.getStatus().getNodeInfo().getKubeletVersion().contains("gke")
        }
        return gkeNodes.size() > 0
    }

    /*
        Misc/Helper Methods
    */

    String getNameSpace() {
        return this.namespace
    }

    String generateYaml(Object orchestratorObject) {
        if (orchestratorObject instanceof NetworkPolicy) {
            return YamlGenerator.toYaml(createNetworkPolicyObject(orchestratorObject))
        }

        return ""
    }

    String getClairifyEndpoint() {
        return "clairify.stackrox:8080"
    }

    def createClairifyDeployment() {
        //create clairify service account
        ServiceAccountBuilder clairifyServiceAccount = new ServiceAccountBuilder()
                .withApiVersion("v1")
                .withKind("ServiceAccount")
                .withNewMetadata()
                        .withName("clairify")
                        .withNamespace("stackrox")
                .endMetadata()
                .withImagePullSecrets(new LocalObjectReference("stackrox"))
        osClient.serviceAccounts().inNamespace("stackrox").createOrReplace(clairifyServiceAccount.build())

        //create clairify securitycontext
        SecurityContextConstraintsBuilder clairifySCC = new SecurityContextConstraintsBuilder()
                .withApiVersion("security.openshift.io/v1")
                .withNewMetadata()
                        .withAnnotations([
                                "kubernetes.io/description":
                                        "clairify is the security constraint for the Clairify container",
                        ])
                        .withName("clairify")
                .endMetadata()
                .withPriority(100)
                .withRunAsUser(new RunAsUserStrategyOptions("RunAsAny", null, null, null))
                .withSeLinuxContext(new SELinuxContextStrategyOptions(null, "RunAsAny"))
                .withSeccompProfiles("*")
                .withUsers("system:serviceaccount:stackrox:clairify")
                .withVolumes("*")
        osClient.securityContextConstraints().createOrReplace(clairifySCC.build())

        //create clairify service
        ServiceBuilder clairifyService = new ServiceBuilder()
                .withApiVersion("v1")
                .withNewMetadata()
                        .withName("clairify")
                        .withNamespace("stackrox")
                .endMetadata()
                .withNewSpec()
                        .withPorts(new ServicePort(
                                "clair-http",
                                null,
                                6060,
                                null,
                                new IntOrString(6060)
                        ))
                        .withPorts(new ServicePort(
                                "clairify-http",
                                null,
                                8080,
                                null,
                                new IntOrString(8080)
                        ))
                        .withType("ClusterIP")
                        .withSelector(["app":"clairify"])
                .endSpec()
        osClient.services().inNamespace("stackrox").createOrReplace(clairifyService.build())

        //create clairify deployment
        Container clairifyContainer = new Container(
            name: "clairify",
            image: "stackrox/clairify:0.3.1",
            env: [new EnvVar("CLAIR_ARGS", "-insecure-tls", null)],
            command: ["/init", "/clairify"],
            imagePullPolicy: "Always",
            ports: [new ContainerPort(6060, null, null, "clair", null),
                     new ContainerPort(8080, null, null, "clairify", null)],
            securityContext: new SecurityContext(
                            null,
                            new Capabilities(null, ["NET_RAW"]),
                            null,
                            null,
                            null,
                            null,
                            null
            )
        )

        DeploymentBuilder clairifyDeployment = new DeploymentBuilder()
                .withNewMetadata()
                        .withName("clairify")
                        .withNamespace("stackrox")
                        .withLabels(["app":"clairify"])
                        .withAnnotations(["owner":"stackrox", "email":"support@stackrox.com"])
                .endMetadata()
                .withNewSpec()
                        .withReplicas(1)
                        .withMinReadySeconds(15)
                        .withSelector(new LabelSelector(null, ["app":"clairify"]))
                        .withNewTemplate()
                                .withNewMetadata()
                                        .withNamespace("stackrox")
                                        .withLabels(["app":"clairify"])
                                .endMetadata()
                                .withNewSpec()
                                        .withContainers(clairifyContainer)
                                        .withServiceAccount("clairify")
                                .endSpec()
                        .endTemplate()
                .endSpec()
        osClient.apps().deployments().inNamespace("stackrox").createOrReplace(clairifyDeployment.build())
        waitForDeploymentCreation("clairify", "stackrox")
    }

    /*
        Private K8S Support functions
    */

    def createDeploymentNoWait(Deployment deployment) {
        deployment.getNamespace() != null ?: deployment.setNamespace(this.namespace)

        DeploymentBuilder dep = new DeploymentBuilder()
                .withNewMetadata()
                        .withName(deployment.name)
                        .withNamespace(deployment.namespace)
                        .withLabels(deployment.labels)
                .endMetadata()
                .withNewSpec()
                        .withSelector(new LabelSelector(null, deployment.labels))
                        .withReplicas(deployment.getReplicas())
                        .withMinReadySeconds(15)
                        .withNewTemplate()
                                .withNewMetadata()
                                        .withName(deployment.name)
                                        .withNamespace(deployment.namespace)
                                        .withLabels(deployment.labels)
                                .endMetadata()
                                .withSpec(generatePodSpec(deployment))
                        .endTemplate()
                .endSpec()

        try {
            println("Told the orchestrator to create " + deployment.getName())
            osClient.apps().deployments().inNamespace(deployment.namespace).createOrReplace(dep.build())
        } catch (Exception e) {
            println("Error creating os deployment: " + e.toString())
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

            // Create service if needed
            if (deployment.exposeAsService) {
                createService(deployment)
                createLoadBalancer(deployment)
            }
        } catch (Exception e) {
            println("Error while waiting for deployment/populating deployment info: " + e.toString())
        }
    }

    def waitForDeploymentCreation(String deploymentName, String namespace, Boolean skipReplicaWait = false) {
        int waitTime = 0

        while (waitTime < maxWaitTime) {
            def dList = osClient.apps().deployments().inNamespace(namespace).list()

            println "Waiting for " + deploymentName
            for (io.fabric8.kubernetes.api.model.apps.Deployment deployment : dList.getItems()) {
                if (deployment.getMetadata().getName() == deploymentName) {
                    // Using the 'skipReplicaWait' bool to avoid timeout waiting for ready replicas if we know
                    // the deployment will not have replicas available
                    if (deployment.getStatus().getReadyReplicas() ==
                            deployment.getSpec().getReplicas() ||
                            skipReplicaWait) {
                        // If skipReplicaWait is set, we still want to sleep for a few seconds to allow the deployment
                        // to work its way through the system.
                        if (skipReplicaWait) {
                            sleep(sleepDuration)
                        }
                        println deploymentName + ": deployment created."
                        return deployment.getMetadata().getUid()
                    }
                }
            }
            sleep(sleepDuration)
            waitTime += sleepDuration
        }
        println "Timed out waiting for " + deploymentName
    }

    def createDaemonSetNoWait(DaemonSet daemonSet) {
        daemonSet.getNamespace() != null ?: daemonSet.setNamespace(this.namespace)

        DaemonSetBuilder dep = new DaemonSetBuilder()
                .withNewMetadata()
                        .withName(daemonSet.name)
                        .withNamespace(daemonSet.namespace)
                        .withLabels(daemonSet.labels)
                .endMetadata()
                .withNewSpec()
                        .withMinReadySeconds(15)
                        .withNewSelector()
                                .withMatchLabels(daemonSet.labels)
                        .endSelector()
                        .withNewTemplate()
                                .withNewMetadata()
                                        .withName(daemonSet.name)
                                        .withNamespace(daemonSet.namespace)
                                        .withLabels(daemonSet.labels)
                                .endMetadata()
                                .withSpec(generatePodSpec(daemonSet))
                        .endTemplate()
                .endSpec()

        try {
            osClient.apps().daemonSets().inNamespace(daemonSet.namespace).createOrReplace(dep.build())
            println("Told the orchestrator to create " + daemonSet.getName())
        } catch (Exception e) {
            println("Error creating os deployment" + e.toString())
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

    def waitForDaemonSetCreation(String deploymentName, String namespace, Boolean skipReplicaWait = false) {
        int waitTime = 0

        while (waitTime < maxWaitTime) {
            DaemonSetList dList = osClient.apps().daemonSets().inNamespace(namespace).list()

            println "Waiting for " + deploymentName
            for (io.fabric8.kubernetes.api.model.apps.DaemonSet daemonSet : dList.getItems()) {
                if (daemonSet.getMetadata().getName() == deploymentName) {
                    // Using the 'skipReplicaWait' bool to avoid timeout waiting for ready replicas if we know
                    // the deployment will not have replicas available
                    if (daemonSet.getStatus().getNumberReady() ==
                            daemonSet.getStatus().getDesiredNumberScheduled() ||
                            skipReplicaWait) {
                        // If skipReplicaWait is set, we still want to sleep for a few seconds to allow the deployment
                        // to work its way through the system.
                        if (skipReplicaWait) {
                            sleep(sleepDuration)
                        }
                        println deploymentName + ": deployment created."
                        return daemonSet.getMetadata().getUid()
                    }
                }
            }
            sleep(sleepDuration)
            waitTime += sleepDuration
        }
        println "Timed out waiting for " + deploymentName
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

        List<VolumeMount> volMounts = deployment.volMounts.collect {
            new VolumeMount(deployment.mountpath, null, it, true, null)
        }

        List<EnvVar> envVars = deployment.env.collect {
            k, v -> new EnvVar(k, v, null)
        }

        List<Volume> volumes = []
        deployment.volNames.each {
            Volume v = new Volume(
                name: it
            )
            if (deployment.secretNames != null || deployment.secretNames.size() != 0) {
                for (String secret : deployment.secretNames) {
                    v.setSecret(new SecretVolumeSource(null, null, null, secret))
                }
            }
            volumes.add(v)
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
                securityContext: new SecurityContext(null, null, deployment.isPrivileged, null, null, null, null)
        )
        PodSpec podSpec = new PodSpec(
                containers: [container],
                volumes: volumes,
                imagePullSecrets: imagePullSecrets
        )
        return podSpec
    }

    def updateDeploymentDetails(Deployment deployment) {
        // Filtering pod query by using the "name=<name>" because it should always be present in the deployment
        // object - IF this is ever missing, it may cause problems fetching pod details
        def deployedPods = osClient.pods().inNamespace(deployment.namespace).withLabel("name", deployment.name).list()
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
    }

    def deleteAdmissionController(String name) {
        println name
    }

    def createAdmissionController(V1beta1ValidatingWebhookConfiguration config) {
        println config
    }
}
