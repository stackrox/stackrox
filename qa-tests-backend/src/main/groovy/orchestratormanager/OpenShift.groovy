package orchestratormanager

import io.fabric8.kubernetes.api.model.Capabilities
import io.fabric8.kubernetes.api.model.Container
import io.fabric8.kubernetes.api.model.ContainerPort
import io.fabric8.kubernetes.api.model.EnvVar
import io.fabric8.kubernetes.api.model.IntOrString
import io.fabric8.kubernetes.api.model.LabelSelector
import io.fabric8.kubernetes.api.model.LocalObjectReference
import io.fabric8.kubernetes.api.model.ObjectMeta
import io.fabric8.kubernetes.api.model.PodSpec
import io.fabric8.kubernetes.api.model.PodTemplateSpec
import io.fabric8.kubernetes.api.model.SecurityContext
import io.fabric8.kubernetes.api.model.Service
import io.fabric8.kubernetes.api.model.ServiceAccount
import io.fabric8.kubernetes.api.model.ServicePort
import io.fabric8.kubernetes.api.model.ServiceSpec
import io.fabric8.kubernetes.api.model.apps.DeploymentSpec
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
import io.fabric8.openshift.client.OpenShiftClient
import objects.Deployment

class OpenShift extends Kubernetes {
    OpenShiftClient oClient

    OpenShift(String ns) {
        super(ns)
        oClient = client.adapt(OpenShiftClient)
    }

    OpenShift() {
        OpenShift("default")
    }

    @Override
    def ensureNamespaceExists(String ns) {
        ProjectRequest projectRequest = new ProjectRequestBuilder()
                .withNewMetadata()
                .withName(ns)
                .addToLabels("project", ns)
                .endMetadata()
                .build()

        try {
            oClient.projectrequests().create(projectRequest)
            println "Created namespace ${ns}"
        } catch (KubernetesClientException kce) {
            // 409 is already exists
            if (kce.code != 409) {
                throw kce
            }
        }

        try {
            SecurityContextConstraints anyuid = oClient.securityContextConstraints().withName("anyuid").get()
            if (anyuid != null && !anyuid.users.contains("system:serviceaccount:" + ns + ":default")) {
                println "Adding system:serviceaccount:" + ns + ":default to anyuid user list"
                anyuid.users.addAll(["system:serviceaccount:" + ns + ":default"])
                oClient.securityContextConstraints().createOrReplace(anyuid)
            }
        } catch (Exception e) {
            println e.toString()
        }
    }

    /*
        Deployment Methods
    */

    @Override
    def getDeploymentCount() {
        return oClient.deploymentConfigs().inAnyNamespace().list().getItems().size() +
                oClient.apps().deployments().inAnyNamespace().list().getItems().size()
    }

    /*
        Service Methods
    */

    @Override
    def createLoadBalancer(Deployment deployment) {
        if (deployment.createLoadBalancer) {
            Route route = new Route(
                    "v1",
                    "Route",
                    new ObjectMeta(name: deployment.name),
                    new RouteSpec(to: new RouteTargetReference("Service", deployment.name, null)),
                    null
            )
            oClient.routes().inNamespace(deployment.namespace).createOrReplace(route)
            int waitTime = 0
            println "Waiting for Route for " + deployment.name
            while (waitTime < maxWaitTime) {
                RouteList rList
                rList = oClient.routes().inNamespace(deployment.namespace).list()

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
        Misc/Helper Methods
    */

    @Override
    def createClairifyDeployment() {
        //create clairify service account
        ServiceAccount clairifyServiceAccount = new ServiceAccount(
                metadata: new ObjectMeta(
                        name: "clairify",
                        namespace: "stackrox"
                ),
                imagePullSecrets: [new LocalObjectReference(name: "stackrox")]
        )
        oClient.serviceAccounts().inNamespace("stackrox").createOrReplace(clairifyServiceAccount)

        //create clairify securitycontext
        SecurityContextConstraints clairifySCC = new SecurityContextConstraints(
                metadata: new ObjectMeta(
                        name: "clairify",
                        annotations: [
                                "kubernetes.io/description":
                                        "clairify is the security constraint for the Clairify container",
                        ]
                ),
                priority: 100,
                runAsUser: new RunAsUserStrategyOptions(
                        type: "RunAsAny"
                ),
                seLinuxContext: new SELinuxContextStrategyOptions(
                        type: "RunAsAny"
                ),
                seccompProfiles: ["*"],
                users: ["system:serviceaccount:stackrox:clairify"],
                volumes: ["*"]
        )
        oClient.securityContextConstraints().createOrReplace(clairifySCC)

        //create clairify service
        Service clairifyService = new Service(
                metadata: new ObjectMeta(
                        name: "clairify",
                        namespace: "stackrox"
                ),
                spec: new ServiceSpec(
                        ports: [
                                new ServicePort(
                                        name: "clair-http",
                                        port: 6060,
                                        targetPort: new IntOrString(6060)
                                ),
                                new ServicePort(
                                        name: "clairify-http",
                                        port: 8080,
                                        targetPort: new IntOrString(8080)
                                )
                        ],
                        type: "ClusterIP",
                        selector: ["app":"clairify"]
                )
        )
        oClient.services().inNamespace("stackrox").createOrReplace(clairifyService)

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
                ports: [new ContainerPort(containerPort: 6060, name: "clair"),
                        new ContainerPort(containerPort: 8080, name: "clairify")
                ],
                securityContext: new SecurityContext(
                        capabilities: new Capabilities(
                                drop: ["NET_RAW"]
                        )
                )
        )

        io.fabric8.kubernetes.api.model.apps.Deployment clairifyDeployment =
                new io.fabric8.kubernetes.api.model.apps.Deployment(
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
                                                serviceAccount: "clairify"
                                        )
                                )
                        )
                )
        oClient.apps().deployments().inNamespace("stackrox").createOrReplace(clairifyDeployment)
        waitForDeploymentCreation("clairify", "stackrox")
    }

}
