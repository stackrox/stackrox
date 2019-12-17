package orchestratormanager

import io.fabric8.kubernetes.api.model.ObjectMeta
import io.fabric8.kubernetes.client.KubernetesClientException
import io.fabric8.openshift.api.model.ProjectRequest
import io.fabric8.openshift.api.model.ProjectRequestBuilder
import io.fabric8.openshift.api.model.Route
import io.fabric8.openshift.api.model.RouteList
import io.fabric8.openshift.api.model.RouteSpec
import io.fabric8.openshift.api.model.RouteTargetReference
import io.fabric8.openshift.api.model.SecurityContextConstraints
import io.fabric8.openshift.client.OpenShiftClient
import objects.Deployment
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
            SecurityContextConstraints anyuid = oClient.securityContextConstraints().withName("anyuid").get()
            if (anyuid != null &&
                    (!anyuid.users.contains("system:serviceaccount:" + ns + ":default") ||
                            !anyuid.allowHostNetwork ||
                            !anyuid.allowHostDirVolumePlugin ||
                            !anyuid.allowHostPorts
                    )) {
                println "Adding system:serviceaccount:" + ns + ":default to anyuid user list"
                anyuid.users.addAll(["system:serviceaccount:" + ns + ":default"])
                anyuid.setAllowHostNetwork(true)
                anyuid.setAllowHostDirVolumePlugin(true)
                anyuid.setAllowHostPorts(true)
                anyuid.setAllowPrivilegedContainer(true)
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
        Service Methods
    */
    @Override
    String waitForLoadBalancer(String serviceName, String namespace) {
        def loadBalancerIP
        Route route = new Route(
                "v1",
                "Route",
                new ObjectMeta(name: serviceName),
                new RouteSpec(to: new RouteTargetReference("Service", serviceName, null)),
                null
        )
        oClient.routes().inNamespace(namespace).createOrReplace(route)
        println "Waiting for Route for " + serviceName
        int retries = maxWaitTimeSeconds / sleepDurationSeconds
        Timer t = new Timer(retries, sleepDurationSeconds)
        while (t.IsValid()) {
            RouteList rList
            rList = oClient.routes().inNamespace(namespace).list()
            for (Route r : rList.getItems()) {
                if (r.getMetadata().getName() == serviceName) {
                    if (r.getStatus().getIngress() != null) {
                        println "Route Host: " + r.getStatus().getIngress().get(0).getHost()
                        return loadBalancerIP
                    }
                }
            }
        }
        println("Could not get loadBalancer IP in ${t.SecondsSince()} seconds")
        return loadBalancerIP
    }

    /*
        Misc/Helper Methods
    */

}
