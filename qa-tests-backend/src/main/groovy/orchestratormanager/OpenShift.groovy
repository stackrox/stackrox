package orchestratormanager

import static util.Helpers.withRetry

import groovy.transform.CompileStatic
import io.stackrox.annotations.Retry
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
@CompileStatic
class OpenShift extends Kubernetes {
    OpenShiftClient oClient

    OpenShift(String ns) {
        super(ns)
        oClient = client.adapt(OpenShiftClient)
    }

    OpenShift() {
        this('default')
    }

    @Override
    @Retry()
    void ensureNamespaceExists(String ns) {
        ProjectRequest projectRequest = new ProjectRequestBuilder()
                .withNewMetadata()
                .withName(ns)
                .addToLabels("project", ns)
                .endMetadata()
                .build()

        try {
            oClient.projectrequests().create(projectRequest)
            log.info "Created namespace ${ns}"
        } catch (KubernetesClientException kce) {
            if (kce.code != 409) {
                throw kce
            }
            log.debug("Namespace ${ns} already exists")
        }

        provisionDefaultServiceAccount(ns)

        String sccName = "anyuid"
        if (Env.CI_JOB_NAME =~ /^(rosa|aro)-/ || Env.CI_JOB_NAME =~ /^osd-/) {
            log.debug "Using a non default SCC"
            sccName = "qatest-anyuid"
        }
        try {
            SecurityContextConstraints scc = oClient.securityContextConstraints().withName(sccName).get()
            if (scc == null) {
                log.error "SCC '${sccName}' not found on this cluster"
                return
            }
            String saUser = "system:serviceaccount:" + ns + ":default"
            if (!scc.users.contains(saUser) ||
                    !scc.allowHostNetwork ||
                    !scc.allowHostDirVolumePlugin ||
                    !scc.allowHostPorts) {
                log.info "Adding ${saUser} to ${sccName} SCC"
                scc.with {
                    users.addAll([saUser])
                    setAllowHostNetwork(true)
                    setAllowHostDirVolumePlugin(true)
                    setAllowHostPorts(true)
                    setAllowPrivilegedContainer(true)
                    setRequiredDropCapabilities([])
                    setAllowedCapabilities(["*"])
                    setAllowedUnsafeSysctls(["*"])
                }
                oClient.securityContextConstraints().createOrReplace(scc)
            }
        } catch (Exception e) {
            log.error "Failed to configure SCC '${sccName}' for namespace ${ns}", e
            throw e
        }

        verifySccApplied(ns, sccName)
    }

    private void verifySccApplied(String ns, String sccName) {
        int maxAttempts = 10
        for (int i = 0; i < maxAttempts; i++) {
            SecurityContextConstraints scc = oClient.securityContextConstraints().withName(sccName).get()
            String saUser = "system:serviceaccount:" + ns + ":default"
            if (scc != null && scc.users.contains(saUser)) {
                log.info "Verified SCC '${sccName}' includes ${saUser} (attempt ${i + 1})"
                return
            }
            log.warn "SCC '${sccName}' does not yet include ${saUser}, retrying (${i + 1}/${maxAttempts})"
            sleep(1000)
        }
        log.error "SCC '${sccName}' was not applied to namespace ${ns} after ${maxAttempts} attempts"
        throw new RuntimeException("SCC '${sccName}' verification failed for namespace ${ns}")
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
    @SuppressWarnings('BuilderMethodWithSideEffects')
    void createRoute(String routeName, String namespace) {
        log.debug "Creating a route: " + routeName
        withRetry(2, 3) {
            Route route = new RouteBuilder().withNewMetadata().withName(routeName).endMetadata()
                    .withNewSpec().withNewTo().withName(routeName).endTo().endSpec().build()
            oClient.routes().inNamespace(namespace).createOrReplace(route)
        }
    }

    @Override
    void deleteRoute(String routeName, String namespace) {
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
