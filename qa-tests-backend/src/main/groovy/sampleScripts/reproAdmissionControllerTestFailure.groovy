package sampleScripts

import common.Constants
import io.stackrox.proto.storage.ClusterOuterClass
import objects.Deployment
import orchestratormanager.OrchestratorMain
import orchestratormanager.OrchestratorType
import services.BaseService
import services.ClusterService
import util.ChaosMonkey
import util.Env

// This script attempts to reproduce a failure with
// AdmissionControllerTest.Verify admission controller does not impair cluster operations when unstable
// i.e. https://issues.redhat.com/browse/ROX-7875

// Use basic authentication for API calls to central. Relies on:
// ROX_USERNAME (defaults to admin)
// ROX_PASSWORD (inferred from the most recent deploy/{k8s,openshift}/central-deploy/password)
// API_HOSTNAME & API_PORT
BaseService.useBasicAuth()
BaseService.setUseClientCert(false)

// Get a cluster client. Assumes you have a working kube configuration. Relies on:
// CLUSTER: Either `OPENSHIFT` or `K8S`. This is inferred from the most recent
//   `deploy/{k8s,openshift}/central-deploy` dir
OrchestratorMain client = OrchestratorType.create(
        Env.mustGetOrchestratorType(),
        Constants.ORCHESTRATOR_NAMESPACE
)

// Tests rely on a standard namespace (qa)
client.ensureNamespaceExists(Constants.ORCHESTRATOR_NAMESPACE)

ClusterOuterClass.AdmissionControllerConfig ac = ClusterOuterClass.AdmissionControllerConfig.newBuilder()
        .setEnabled(false)
        .setScanInline(false)
        .setTimeoutSeconds(10)
        .build()

assert ClusterService.updateAdmissionController(ac)

// Give sensor a chance to catch up to changed configuration
sleep(5000)

// Start a chaos monkey thread that kills _all_ ready admission control replicas with a short grace period
def killAllChaosMonkey = new ChaosMonkey(client, 0, 1L)

def deployment = new Deployment()
        .setName("random-busybox")
        .setImage("quay.io/rhacs-eng/qa:busybox-1-30")
        .addLabel("app", "random-busybox")
assert client.createDeploymentNoWait(deployment)

for (int i = 0; i < 450; i++) {
    sleep(1000)
    deployment.addAnnotation("qa.stackrox.io/iteration", "${i}")
    assert client.updateDeploymentNoWait(deployment)
}

killAllChaosMonkey.stop()
client.deleteDeployment(deployment)
