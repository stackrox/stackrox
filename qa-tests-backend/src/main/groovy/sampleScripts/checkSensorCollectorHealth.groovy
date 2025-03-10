package sampleScripts

import common.Constants
import orchestratormanager.Kubernetes
import orchestratormanager.OrchestratorType
import util.ApplicationHealth
import util.Env

// Get a cluster client. Assumes you have a working kube configuration. Relies on:
// CLUSTER: Either `OPENSHIFT` or `K8S`. This is inferred from the most recent
//   `deploy/{k8s,openshift}/central-deploy` dir
Kubernetes client = OrchestratorType.create(
        Env.mustGetOrchestratorType(),
        Constants.ORCHESTRATOR_NAMESPACE
)

ApplicationHealth ah = new ApplicationHealth(client, 600)

ah.waitForCollectorHealthiness()
