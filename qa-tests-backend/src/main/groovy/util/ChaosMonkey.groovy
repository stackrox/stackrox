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
    static final private int ADMISSION_CONTROLLER_EXPECTED_PODS = 3

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
                admCtrlPods.removeIf { Pod p -> !orchestrator.podReady(p) }

                if (admCtrlPods.size() < minReadyReplicas) {
                    log.warn "Fewer than ${minReadyReplicas} ready ${ADMISSION_CONTROLLER_APP_NAME} pods encountered!" +
                             " This should not happen!"
                }
                if (admCtrlPods.size() <= minReadyReplicas) {
                    lock.lock()
                    effectCond.signalAll()
                    lock.unlock()
                }

                // If there are more than the minimum number of ready replicas, randomly pick some to delete
                if (admCtrlPods.size() > minReadyReplicas) {
                    Collections.shuffle(admCtrlPods)
                    def podsToDelete = admCtrlPods.drop(minReadyReplicas)
                    podsToDelete.forEach {
                        log.info "Deleting pod ${it.metadata.name}."
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
        while (true) {
            def admCtrlPods = orchestrator.getPods(Constants.STACKROX_NAMESPACE, ADMISSION_CONTROLLER_APP_NAME)
            if (admCtrlPods.size() < ADMISSION_CONTROLLER_EXPECTED_PODS) {
                continue
            }
            def readyPods = admCtrlPods.findAll { Pod p -> orchestrator.podReady(p) }
            if (readyPods.size() == admCtrlPods.size()) {
                def readyPodNames = readyPods.collect { Pod p -> p.metadata.name }
                log.info "ChaosMonkey: All admission control pod replicas ready: ${readyPodNames}"
                break
            }
            Helpers.sleepWithRetryBackoff(1000)
        }
    }
}

