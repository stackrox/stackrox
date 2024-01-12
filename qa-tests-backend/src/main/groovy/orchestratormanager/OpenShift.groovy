package orchestratormanager

import static util.Helpers.withRetry

import groovy.util.logging.Slf4j
import io.fabric8.kubernetes.client.KubernetesClientException
import io.fabric8.openshift.api.model.ProjectRequest
import io.fabric8.openshift.api.model.ProjectRequestBuilder
import io.fabric8.openshift.api.model.Route
import io.fabric8.openshift.api.model.RouteBuilder
import io.fabric8.openshift.api.model.SecurityContextConstraints
import io.fabric8.openshift.client.OpenShiftClient
import util.Env
import util.Timer

@Slf4j
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
            log.info "Created namespace ${ns}"
            provisionDefaultServiceAccount(ns)
        } catch (KubernetesClientException kce) {
            if (kce.code != 409) {
                throw kce
            }
            log.debug("Namespace ${ns} already exists")
        }
    }

    /*
        Deployment Methods
    */

    @Override
    List<String> getDeploymentCount(String ns) {
        return oClient.apps().deployments().inNamespace(ns).list().getItems().collect { it.metadata.name } +
                oClient.deploymentConfigs().inNamespace(ns).list().getItems().collect { it.metadata.name }
    }

    /*
        Route Methods
    */

    @Override
    def createRoute(String routeName, String namespace) {
        log.debug "Creating a route: " + routeName
        withRetry(2, 3) {
            Route route = new RouteBuilder().withNewMetadata().withName(routeName).endMetadata()
                    .withNewSpec().withNewTo().withName(routeName).endTo().endSpec().build()
            oClient.routes().inNamespace(namespace).createOrReplace(route)
        }
    }

    @Override
    def deleteRoute(String routeName, String namespace) {
        log.debug "Deleting a route: " + routeName
        withRetry(2, 3) {
            Route route = new RouteBuilder().withNewMetadata().withName(routeName).endMetadata().build()
            oClient.routes().inNamespace(namespace).delete(route)
        }
    }

    @Override
    String waitForRouteHost(String serviceName, String namespace) {
        log.debug "Waiting for route: " + serviceName
        int retries = (int) (maxWaitTimeSeconds / sleepDurationSeconds)
        Timer t = new Timer(retries, sleepDurationSeconds)
        while (t.IsValid()) {
            Route route = oClient.routes().inNamespace(namespace).withName(serviceName).get()
            if (route?.status?.ingress?.size() > 0) {
                log.debug "Route Host: " + route.status.ingress[0].host
                return route.status.ingress[0].host
            }
        }
        log.warn("Could not get route host in ${t.SecondsSince()} seconds")
        return null
    }
}
