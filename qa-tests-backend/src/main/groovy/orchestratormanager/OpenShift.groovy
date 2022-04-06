package orchestratormanager

import io.fabric8.kubernetes.client.KubernetesClientException
import io.fabric8.openshift.api.model.ProjectRequest
import io.fabric8.openshift.api.model.ProjectRequestBuilder
import io.fabric8.openshift.api.model.Route
import io.fabric8.openshift.api.model.RouteBuilder
import io.fabric8.openshift.api.model.SecurityContextConstraints
import io.fabric8.openshift.client.OpenShiftClient
import util.Env
import util.Timer

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
            String sccName = "anyuid"
            if (Env.CI_JOBNAME =~ /-(rosa|aro)-/ || Env.CI_JOBNAME =~ /^osd-/) {
                println "Using a non default SCC"
                sccName = "qatest-anyuid"
            }
            SecurityContextConstraints anyuid = oClient.securityContextConstraints().withName(sccName).get()
            if (anyuid != null &&
                    (!anyuid.users.contains("system:serviceaccount:" + ns + ":default") ||
                            !anyuid.allowHostNetwork ||
                            !anyuid.allowHostDirVolumePlugin ||
                            !anyuid.allowHostPorts
                    )) {
                println "Adding system:serviceaccount:${ns}:default to ${sccName} user list"
                anyuid.with {
                    // (Note: + string concatenation here to avoid json unmarshal errors
                    users.addAll(["system:serviceaccount:" + ns + ":default"])
                    setAllowHostNetwork(true)
                    setAllowHostDirVolumePlugin(true)
                    setAllowHostPorts(true)
                    setAllowPrivilegedContainer(true)
                    setRequiredDropCapabilities([])
                    setAllowedCapabilities(["*"])
                    setAllowedUnsafeSysctls(["*"])
                }
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
    def getDeploymentCount(String ns = null) {
        return oClient.apps().deployments().inNamespace(ns).list().getItems().collect { it.metadata.name } +
                oClient.deploymentConfigs().inNamespace(ns).list().getItems().collect { it.metadata.name }
    }

    /*
        Route Methods
    */

    @Override
    def createRoute(String routeName, String namespace) {
        println "Creating a route: " + routeName
        withRetry(2, 3) {
            Route route = new RouteBuilder().withNewMetadata().withName(routeName).endMetadata()
                    .withNewSpec().withNewTo().withName(routeName).endTo().endSpec().build()
            oClient.routes().inNamespace(namespace).createOrReplace(route)
        }
    }

    @Override
    def deleteRoute(String routeName, String namespace) {
        println "Deleting a route: " + routeName
        withRetry(2, 3) {
            Route route = new RouteBuilder().withNewMetadata().withName(routeName).endMetadata().build()
            oClient.routes().inNamespace(namespace).delete(route)
        }
    }

    @Override
    String waitForRouteHost(String serviceName, String namespace) {
        println "Waiting for route: " + serviceName
        int retries = (int) (maxWaitTimeSeconds / sleepDurationSeconds)
        Timer t = new Timer(retries, sleepDurationSeconds)
        while (t.IsValid()) {
            Route route = oClient.routes().inNamespace(namespace).withName(serviceName).get()
            if (route?.status?.ingress?.size() > 0) {
                println "Route Host: " + route.status.ingress[0].host
                return route.status.ingress[0].host
            }
        }
        println("Could not get route host in ${t.SecondsSince()} seconds")
        return null
    }
}
