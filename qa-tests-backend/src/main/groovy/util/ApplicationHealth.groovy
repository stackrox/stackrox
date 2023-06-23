package util

import common.Constants
import groovy.util.logging.Slf4j
import io.fabric8.kubernetes.client.KubernetesClientException
import objects.DaemonSet
import objects.Deployment
import orchestratormanager.OrchestratorMain

@Slf4j
class ApplicationHealth {
    OrchestratorMain client
    Integer waitTimeForHealthiness
    final Integer delayBetweenChecks = 5
    final Map<String, String> readyLogMessages = [
            "admission-control": "Applied new admission control settings",
            "collector": "Sensor connectivity is successful",
            "sensor": "TLS-enabled multiplexed HTTP/gRPC server listening on",
    ]

    ApplicationHealth(OrchestratorMain client, Integer waitTimeForHealthiness) {
        this.client = client
        this.waitTimeForHealthiness = waitTimeForHealthiness
    }

    void waitForSensorHealthiness() {
        Deployment sensor = new Deployment().setNamespace(Constants.STACKROX_NAMESPACE).setName("sensor")
        waitForHealthiness(sensor)
    }

    void waitForCollectorHealthiness() {
        Deployment collector = new DaemonSet().setNamespace(Constants.STACKROX_NAMESPACE).setName("collector")
        waitForHealthiness(collector)
    }

    void waitForAdmissionControllerHealthiness() {
        Deployment admissionController = new Deployment().setNamespace(Constants.STACKROX_NAMESPACE)
            .setName("admission-control")
        waitForHealthiness(admissionController)
    }

    void waitForHealthiness(Deployment deployment) {
        Long endAt = System.currentTimeSeconds() + this.waitTimeForHealthiness

        Integer replicaCount = this.getReplicaCount(deployment, endAt)
        this.waitForHealthyPods(deployment, replicaCount, endAt)
    }

    private Integer getReplicaCount(Deployment deployment, Long endAt) {
        Integer replicaCount

        while (endAt > System.currentTimeSeconds()) {
            try {
                if (deployment instanceof DaemonSet) {
                    replicaCount = client.getDaemonSetReplicaCount(deployment) as Integer
                } else if (deployment instanceof Deployment) {
                    replicaCount = client.getDeploymentReplicaCount(deployment) as Integer
                }
                else {
                    throw new RuntimeException("Expect DaemonSet or Deployment")
                }
                if (replicaCount == null) {
                    throw new RuntimeException("Expected to get replica count")
                }
                log.debug "${replicaCount} ${deployment.name} pods expected"
                return replicaCount
            }
            catch (Exception e) {
                Long timeLeft = endAt - System.currentTimeSeconds()
                Long thisWait = timeLeft < delayBetweenChecks ? timeLeft : delayBetweenChecks
                log.debug("Cannot get ${deployment.name} replica count: ${e}, will retry in ${thisWait} seconds", e)
                sleep(thisWait * 1000)
            }
        }

        throw new RuntimeException("Gave up trying to get replica count")
    }

    private void waitForHealthyPods(Deployment deployment, Integer replicaCount, Long endAt) {
        String readyLogMessage = this.readyLogMessages[deployment.name]
        if (!readyLogMessage) {
            throw new RuntimeException("There is no ready log message for ${deployment.name}")
        }

        Integer healthyPods = 0
        while (endAt > System.currentTimeSeconds()) {
            healthyPods = 0
            client.getPods(Constants.STACKROX_NAMESPACE, deployment.name).each {
                String logs
                try {
                    logs = client.getContainerlogs(Constants.STACKROX_NAMESPACE, it.metadata.name, deployment.name)
                }
                catch (KubernetesClientException e) {
                    log.error("Cannot get container logs", e)
                    return
                }
                if (logs.contains(readyLogMessage)) {
                    log.debug "${deployment.name} ${it.metadata.name} is in the desired state"
                    healthyPods++
                }
                else {
                    log.debug "${deployment.name} ${it.metadata.name} is not in the desired state"
                }
            }

            if (healthyPods == replicaCount) {
                log.debug "${deployment.name} is healthy"
                return
            }

            Long timeLeft = endAt - System.currentTimeSeconds()
            Long thisWait = timeLeft < delayBetweenChecks ? timeLeft : delayBetweenChecks
            log.debug "${deployment.name} has yet to reach an operable state, "+
                    "will try again in ${thisWait} seconds, ${timeLeft} seconds remain"
            sleep(thisWait * 1000)
        }

        throw new RuntimeException("Gave up waiting for pods to reach an operable state")
    }
}
