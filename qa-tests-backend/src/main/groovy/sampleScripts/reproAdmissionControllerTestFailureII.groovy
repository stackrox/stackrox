package sampleScripts

import org.slf4j.Logger
import org.slf4j.LoggerFactory

import common.Constants
import io.stackrox.proto.storage.ClusterOuterClass
import io.stackrox.proto.storage.ClusterOuterClass.AdmissionControllerConfig
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import io.stackrox.proto.storage.ScopeOuterClass

import objects.Deployment
import orchestratormanager.OrchestratorMain
import orchestratormanager.OrchestratorType
import services.BaseService
import services.ClusterService
import services.ImageService
import services.PolicyService
import util.Env

// This script attempts to reproduce a failure with
// AdmissionControllerTest.Verify Admission Controller Config: nginx w/ inline scan

Logger LOG = LoggerFactory.getLogger("test")
LOG.debug("test")

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

String TEST_NAMESPACE = "qa-admission-controller-test"

String NGINX = "qanginx"
String NGINX_IMAGE = "quay.io/rhacs-eng/qa-multi-arch:nginx-1.21.1"

String CLONED_POLICY_SUFFIX = "(${TEST_NAMESPACE})"
String LATEST_TAG = "Latest tag"
String SEVERITY = "Fixable Severity at least Important"

Deployment NGINX_DEPLOYMENT = new Deployment()
            .setName(NGINX)
            .setNamespace(TEST_NAMESPACE)
            .setImage(NGINX_IMAGE)
            .addLabel("app", "test")

if (false) {
        // Create namespace scoped policies for test based on "Latest Tag" and
        // "Fixable Severity at least Important"
        for (policyName : [LATEST_TAG, SEVERITY]) {
                Policy policy = PolicyService.getPolicy(policyName)
                def scopedPolicyForTest = policy.toBuilder()
                        .clearId()
                        .setName(policy.getName() + " ${CLONED_POLICY_SUFFIX}")
                        .clearScope()
                        .addScope(ScopeOuterClass.Scope.newBuilder().setNamespace(TEST_NAMESPACE))
                        .clearEnforcementActions()
                        .addEnforcementActions(PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT)
                        .build()
                String policyID = PolicyService.createNewPolicy(scopedPolicyForTest)
                assert policyID
        }

        // Wait for propagation to sensor
        sleep(10000)

        // Pre run scan to avoid timeouts with inline scans in the tests below
        ImageService.scanImage(NGINX_IMAGE)

        client.ensureNamespaceExists(TEST_NAMESPACE)

        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(true)
                .setDisableBypass(true)
                .setScanInline(true)
                .setTimeoutSeconds(30)
                .build()

        assert ClusterService.updateAdmissionController(ac)
        // Wait for propagation to sensor
        sleep(5000)
}

def i=0
while (i<1000) {
        def created = client.createDeploymentNoWait(NGINX_DEPLOYMENT)
        assert !created
        println "iteration ${i++}"
}
