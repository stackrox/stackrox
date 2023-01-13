package sampleScripts

import orchestratormanager.OrchestratorMain
import orchestratormanager.OrchestratorType
import common.Constants
import util.Env
import org.slf4j.Logger
import org.slf4j.LoggerFactory

OrchestratorMain orchestrator = OrchestratorType.create(
           Env.mustGetOrchestratorType(),
           Constants.ORCHESTRATOR_NAMESPACE
)
Logger log = LoggerFactory.getLogger("repro")

log.info "scaling down..."
orchestrator.scaleDeployment("stackrox-operator", "admission-control", 0)
log.info "waiting for pods to be removed"
res = orchestrator.waitForAllPodsToBeRemoved("stackrox-operator", [app: "admission-control"], 30, 1)
log.info("Admission controller scaled to 0, was 3")
orchestrator.scaleDeployment("stackrox-operator", "admission-control", 3)
orchestrator.waitForPodsReady("stackrox-operator", [app: "admission-control"], 3, 30, 1)
log.info("Admission controller scaled back to 3")
