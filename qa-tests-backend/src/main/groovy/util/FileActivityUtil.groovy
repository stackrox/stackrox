package util

import static util.Helpers.waitForTrue

import io.stackrox.proto.api.v1.AlertServiceOuterClass.ListAlertsRequest
import io.stackrox.proto.storage.PolicyOuterClass

import common.Constants
import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j
import orchestratormanager.Kubernetes
import services.AlertService

@Slf4j
@CompileStatic
class FileActivityUtil {

    static boolean isFactAvailable(Kubernetes orchestrator) {
        return orchestrator.containsDaemonSetContainer(
                Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, Constants.FACT_CONTAINER)
    }

    static void setFactEnv(Kubernetes orchestrator, String paths, boolean json) {
        String jsonStr = Boolean.toString(json)
        log.info "Setting FACT env on collector DaemonSet: FACT_PATHS=${paths}, FACT_JSON=${jsonStr}"

        orchestrator.updateDaemonSetEnv(
                Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, Constants.FACT_CONTAINER,
                "FACT_PATHS", paths)
        orchestrator.updateDaemonSetEnv(
                Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, Constants.FACT_CONTAINER,
                "FACT_JSON", jsonStr)

        log.info "Waiting for collector DS to pick up FACT env vars and be ready"
        waitForTrue(20, 10) {
            orchestrator.daemonSetEnvVarUpdated(
                    Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, Constants.FACT_CONTAINER,
                    "FACT_PATHS", paths) &&
            orchestrator.daemonSetEnvVarUpdated(
                    Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, Constants.FACT_CONTAINER,
                    "FACT_JSON", jsonStr) &&
            orchestrator.daemonSetReady(Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS)
        }
    }

    static void removeFactEnv(Kubernetes orchestrator) {
        log.info "Removing FACT env vars from collector DaemonSet"

        orchestrator.removeDaemonSetEnv(
                Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, Constants.FACT_CONTAINER,
                "FACT_PATHS")
        orchestrator.removeDaemonSetEnv(
                Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS, Constants.FACT_CONTAINER,
                "FACT_JSON")

        log.info "Waiting for collector DS to be ready"
        waitForTrue(20, 10) {
            orchestrator.daemonSetReady(Constants.STACKROX_NAMESPACE, Constants.COLLECTOR_DS)
        }
    }

    static PolicyOuterClass.Policy createFileActivityPolicy(
            String name, String path, PolicyOuterClass.EventSource eventSource, String... operations) {
        def groups = [
                PolicyOuterClass.PolicyGroup.newBuilder()
                        .setFieldName("File Path")
                        .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue(path))
                        .build(),
        ]

        if (operations.length > 0) {
            def opGroup = PolicyOuterClass.PolicyGroup.newBuilder()
                    .setFieldName("File Operation")
            operations.each { String op ->
                opGroup.addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue(op))
            }
            groups << opGroup.build()
        }

        return PolicyOuterClass.Policy.newBuilder()
                .setName(name)
                .addLifecycleStages(PolicyOuterClass.LifecycleStage.RUNTIME)
                .setEventSource(eventSource)
                .setSeverityValue(2)
                .addCategories("File Activity Monitoring")
                .setDisabled(false)
                .addPolicySections(
                        PolicyOuterClass.PolicySection.newBuilder()
                                .setSectionName("file-access")
                                .addAllPolicyGroups(groups)
                                .build()
                )
                .build()
    }

    static void resolveAlertsByPolicy(String policyName) {
        def alerts = AlertService.getViolations(
                ListAlertsRequest.newBuilder()
                        .setQuery("Policy:${policyName}+Violation State:ACTIVE")
                        .build())
        for (alert in alerts) {
            AlertService.resolveAlert(alert.id)
        }
    }
}
