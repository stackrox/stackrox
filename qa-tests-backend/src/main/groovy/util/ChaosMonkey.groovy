package util

import common.Constants
import groovy.util.logging.Slf4j
import io.fabric8.kubernetes.api.model.Pod
import orchestratormanager.OrchestratorMain

import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.locks.ReentrantLock

@Slf4j
class ChaosMonkey {
    def stopFlag = new AtomicBoolean()
    def lock = new ReentrantLock()
    def effectCond = lock.newCondition()

    Thread thread
    OrchestratorMain orchestrator

    static final private String ADMISSION_CONTROLLER_APP_NAME = "admission-control"

    ChaosMonkey(OrchestratorMain client, int minReadyReplicas, Long gracePeriod) {
        orchestrator = client

        def pods = orchestrator.getPods(Constants.STACKROX_NAMESPACE, ADMISSION_CONTROLLER_APP_NAME)
        assert pods.size() > 0, "There are no ${ADMISSION_CONTROLLER_APP_NAME} pods. " +
                "Did you enable ADMISSION_CONTROLLER when deploying?"

        thread = Thread.start {
            while (!stopFlag.get()) {
                // Get the current ready, non-deleted pod replicas
                def admCtrlPods = new ArrayList<Pod>(orchestrator.getPods(
                        Constants.STACKROX_NAMESPACE, ADMISSION_CONTROLLER_APP_NAME))
                admCtrlPods.removeIf { !it?.status?.containerStatuses[0]?.ready }

                if (admCtrlPods.size() <= minReadyReplicas) {
                    lock.lock()
                    effectCond.signalAll()
                    lock.unlock()
                }

                admCtrlPods.removeIf { it?.metadata?.deletionTimestamp as boolean }

                // If there are more than the minimum number of ready replicas, randomly pick some to delete
                if (admCtrlPods.size() > minReadyReplicas) {
                    Collections.shuffle(admCtrlPods)
                    def podsToDelete = admCtrlPods.drop(minReadyReplicas)
                    podsToDelete.forEach {
                        orchestrator.deletePod(it.metadata.namespace, it.metadata.name, gracePeriod)
                    }
                }
                Helpers.sleepWithRetryBackoff(1000)
            }
        }
    }

    void stop() {
        stopFlag.set(true)
        thread.join()
    }

    def waitForEffect() {
        lock.lock()
        effectCond.await()
        lock.unlock()
    }

    void waitForReady() {
        def allReady = false
        while (!allReady) {
            Helpers.sleepWithRetryBackoff(1000)

            def admCtrlPods = orchestrator.getPods(Constants.STACKROX_NAMESPACE, ADMISSION_CONTROLLER_APP_NAME)
            if (admCtrlPods.size() < 3) {
                continue
            }
            allReady = true
            for (def pod : admCtrlPods) {
                if (!pod.status?.containerStatuses[0]?.ready) {
                    allReady = false
                    break
                }
            }
        }
        log.info "ChaosMonkey: All admission control pod replicas ready"
    }
}

