package sampleScripts

import common.Constants
import orchestratormanager.OrchestratorMain
import orchestratormanager.OrchestratorType
import org.javers.core.Javers
import org.javers.core.JaversBuilder
import org.javers.core.diff.ListCompareAlgorithm
import org.slf4j.Logger
import org.slf4j.LoggerFactory
import services.BaseService
import services.SummaryService
import services.DeploymentService
import util.Env

// Repeat the deployment counts found in SummaryTest.

Logger log = LoggerFactory.getLogger("scripts")
log.info "Hello world!"

// Use basic authentication for API calls to central. Relies on:
// ROX_USERNAME (defaults to admin)
// ROX_ADMIN_PASSWORD (inferred from the most recent deploy/{k8s,openshift}/central-deploy/password)
// API_HOSTNAME & API_PORT
BaseService.useBasicAuth()
BaseService.setUseClientCert(false)

// Get a cluster client. Assumes you have a working kube configuration. Relies on:
// CLUSTER: Either `OPENSHIFT` or `K8S`. This is inferred from the most recent
//   `deploy/{k8s,openshift}/central-deploy` dir
OrchestratorMain orchestrator = OrchestratorType.create(
        Env.mustGetOrchestratorType(),
        Constants.ORCHESTRATOR_NAMESPACE
)

//

def stackroxSummaryCounts = SummaryService.getCounts()
log.info "Stackrox deployment count: ${stackroxSummaryCounts.numDeployments}"

List<String> orchestratorDeploymentNames = orchestrator.getDeploymentCount()
List<String> orchestratorDaemonSetNames = orchestrator.getDaemonSetCount()
List<String> orchestratorStaticPodNames = orchestrator.getStaticPodCount().collect {  "static-" + it + "-pods"  }
List<String> orchestratorStatefulSetNames = orchestrator.getStatefulSetCount()
List<String> orchestratorJobNames = orchestrator.getJobCount()

log.info "orchestratorDeploymentNames: ${orchestratorDeploymentNames.size()}"
log.info "orchestratorDaemonSetNames: ${orchestratorDaemonSetNames.size()}"
log.info "orchestratorStaticPodNames: ${orchestratorStaticPodNames.size()}"
log.info "orchestratorStatefulSetNames: ${orchestratorStatefulSetNames.size()}"
log.info "orchestratorJobNames: ${orchestratorJobNames.size()}"

List<String> orchestratorResourceNames = orchestratorDeploymentNames +
    orchestratorDaemonSetNames +
    orchestratorStaticPodNames +
    orchestratorStatefulSetNames +
    orchestratorJobNames

log.info "Stackrox count: ${stackroxSummaryCounts.numDeployments}, " +
         "orchestrator count ${orchestratorResourceNames.size()}"

List<String> stackroxDeploymentNames = DeploymentService.listDeployments()*.name
Javers javers = JaversBuilder.javers()
        .withListCompareAlgorithm(ListCompareAlgorithm.AS_SET)
        .build()
log.info javers.compare(stackroxDeploymentNames, orchestratorResourceNames).prettyPrint()

log.info "Stackrox deployments: " + stackroxDeploymentNames.sort().join(",")
log.info "Orchestrator deployments: " + orchestratorResourceNames.sort().join(",")

assert Math.abs(stackroxSummaryCounts.numDeployments - orchestratorResourceNames.size()) <= 2
