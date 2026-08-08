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

    private static final Set<String> ANYUID_SCCS = ["anyuid", "qatest-anyuid"] as Set
    private static final Set<String> RESTRICTED_SCCS = ["restricted", "restricted-v2", "restricted-v3"] as Set

    private void verifySccApplied(String ns, String sccName) {
        String probePodName = "scc-probe-" + System.nanoTime()
        int maxAttempts = 10
        for (int i = 0; i < maxAttempts; i++) {
            try {
                def pod = client.pods().inNamespace(ns).createOrReplace(
                    new io.fabric8.kubernetes.api.model.PodBuilder()
                        .withNewMetadata().withName(probePodName).endMetadata()
                        .withNewSpec()
                            .addNewContainer()
                                .withName("probe")
                                .withImage("registry.access.redhat.com/ubi9/ubi-minimal:latest")
                                .withCommand("true")
                            .endContainer()
                            .withRestartPolicy("Never")
                        .endSpec()
                        .build()
                )
                String assignedScc = pod?.metadata?.annotations?.get("openshift.io/scc") ?: ""
                client.pods().inNamespace(ns).withName(probePodName).delete()
                if (assignedScc == sccName || ANYUID_SCCS.contains(assignedScc)) {
                    log.info "Verified SCC '${assignedScc}' is active for namespace ${ns} (attempt ${i + 1})"
                    return
                }
                if (RESTRICTED_SCCS.contains(assignedScc)) {
                    log.warn "Probe pod got restrictive SCC '${assignedScc}' instead of '${sccName}', " +
                        "retrying (${i + 1}/${maxAttempts})"
                } else {
                    log.info "Probe pod got SCC '${assignedScc}' (not '${sccName}' but not restricted), " +
                        "accepting (attempt ${i + 1})"
                    return
                }
            } catch (Exception e) {
                log.warn "SCC probe pod failed: ${e.message}, retrying (${i + 1}/${maxAttempts})"
                try { client.pods().inNamespace(ns).withName(probePodName).delete() } catch (Exception ignored) {}
            }
            sleep(2000)
        }
        log.error "SCC verification failed: namespace ${ns} still gets a restricted SCC after ${maxAttempts} attempts"
        throw new RuntimeException("SCC verification failed for namespace ${ns} - " +
            "admission controller assigned a restricted SCC instead of '${sccName}' or equivalent")
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
