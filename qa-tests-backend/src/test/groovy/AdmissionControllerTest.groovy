import common.Constants
import groups.BAT
import groups.SensorBounceNext
import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.PolicyServiceOuterClass
import io.stackrox.proto.storage.ClusterOuterClass.AdmissionControllerConfig
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.PolicyGroup
import io.stackrox.proto.storage.PolicyOuterClass.PolicySection
import io.stackrox.proto.storage.PolicyOuterClass.PolicyValue
import io.stackrox.proto.storage.ScopeOuterClass
import objects.Deployment
import objects.GCRImageIntegration
import org.junit.experimental.categories.Category
import services.CVEService
import services.ClusterService
import services.ImageIntegrationService
import services.PolicyService
import spock.lang.Retry
import spock.lang.Shared
import spock.lang.Timeout
import spock.lang.Unroll
import util.ChaosMonkey
import util.Env
import util.Helpers
import util.Timer

class AdmissionControllerTest extends BaseSpecification {
    @Shared
    private List<PolicyOuterClass.EnforcementAction> latestTagEnforcements
    @Shared
    private List<PolicyOuterClass.EnforcementAction> severityEnforcements
    @Shared
    private String gcrId
    @Shared
    private String clusterId

    private ChaosMonkey chaosMonkey

    static final private String GCR_NGINX         = "qagcrnginx"
    static final private String BUSYBOX_NO_BYPASS = "busybox-no-bypass"
    static final private String BUSYBOX_BYPASS    = "busybox-bypass"

    private final static String LATEST_TAG = "Latest tag"
    private final static String SEVERITY = "Fixable Severity at least Important"

    static final private Deployment GCR_NGINX_DEPLOYMENT = new Deployment()
            .setName(GCR_NGINX)
            .setImage("us.gcr.io/stackrox-ci/nginx:1.10.1")
            .addLabel("app", "test")

    static final private Deployment BUSYBOX_NO_BYPASS_DEPLOYMENT = new Deployment()
            .setName(BUSYBOX_NO_BYPASS)
            .setImage("busybox:latest")
            .addLabel("app", "test")

    static final private Deployment BUSYBOX_BYPASS_DEPLOYMENT = new Deployment()
            .setName(BUSYBOX_BYPASS)
            .setImage("busybox:latest")
            .addLabel("app", "test")
            .addAnnotation("admission.stackrox.io/break-glass", "yay")

    static final private Deployment MISC_DEPLOYMENT = new Deployment()
        .setName("random-busybox")
        .setImage("quay.io/rhacs-eng/qa:busybox-1-30")
        .addLabel("app", "random-busybox")

    def setupSpec() {
        clusterId = ClusterService.getClusterId()
        assert clusterId

        latestTagEnforcements = Services.updatePolicyEnforcement(
                LATEST_TAG,
                [PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,]
        )

        severityEnforcements = Services.updatePolicyEnforcement(
                SEVERITY,
                [PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,]
        )

        ImageIntegrationService.deleteStackRoxScannerIntegrationIfExists()
        gcrId = GCRImageIntegration.createDefaultIntegration()
        assert gcrId != ""
    }

    def setup() {
        // https://stack-rox.atlassian.net/browse/ROX-7026 - Disable ChaosMonkey
        // // By default, operate with a chaos monkey that keeps one ready replica alive and deletes with a 10s grace
        // // period, which should be sufficient for K8s to pick up readiness changes and update endpoints.
        // chaosMonkey = new ChaosMonkey(orchestrator, 1, 10L)
        // chaosMonkey.waitForEffect()
    }

    def cleanup() {
        if (chaosMonkey) {
            chaosMonkey.stop()
            chaosMonkey.waitForReady()
        }
    }

    def cleanupSpec() {
        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(false)
                .build()

        assert ClusterService.updateAdmissionController(ac)

        Services.updatePolicyEnforcement(
                LATEST_TAG,
                latestTagEnforcements
        )

        Services.updatePolicyEnforcement(
                SEVERITY,
                severityEnforcements
        )
        assert ImageIntegrationService.deleteImageIntegration(gcrId)
        ImageIntegrationService.addStackroxScannerIntegration()
    }

    @Unroll
    @Category([BAT])
    def "Verify Admission Controller Config (#desc)"() {
        when:

        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                                .setEnabled(true)
                                .setDisableBypass(!bypassable)
                                .setScanInline(scan)
                                .setTimeoutSeconds(timeout)
                            .build()

        assert ClusterService.updateAdmissionController(ac)
        // Maximum time to wait for propagation to sensor
        Helpers.sleepWithRetryBackoff(5000)

        then:
        "Run deployment request"
        def created = orchestrator.createDeploymentNoWait(deployment)
        assert created == launched

        cleanup:
        "Revert Cluster"
        if (created) {
            deleteDeploymentWithCaution(deployment)
        }

        where:
        "Data inputs are: "

        timeout | scan  | bypassable | deployment                   | launched | desc
        3       | false | false      | BUSYBOX_NO_BYPASS_DEPLOYMENT | false    | "no bypass annotation, non-bypassable"
        3       | false | false      | BUSYBOX_BYPASS_DEPLOYMENT    | false    | "bypass annotation, non-bypassable"
        3       | false | true       | BUSYBOX_BYPASS_DEPLOYMENT    | true     | "bypass annotation, bypassable"
        30      | true  | false      | GCR_NGINX_DEPLOYMENT         | false    | "nginx w/ inline scan"
    }

    @Unroll
    @Category([BAT])
    def "Verify CVE snoozing applies to images scanned by admission controller #image"() {
        given:
         "Create policy looking for a specific CVE"
        // We don't want to block on SEVERITY
        Services.updatePolicyEnforcement(
                SEVERITY,
                severityEnforcements
        )

        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(true)
                .setEnforceOnUpdates(false)
                .setDisableBypass(false)
                .setScanInline(true)
                .setTimeoutSeconds(5)
                .build()
        assert ClusterService.updateAdmissionController(ac)

        log.info("Admission control configuration updated")

        def policyGroup = PolicyGroup.newBuilder()
                .setFieldName("CVE")
                .setBooleanOperator(PolicyOuterClass.BooleanOperator.AND)
        policyGroup.addAllValues([PolicyValue.newBuilder().setValue("CVE-2019-3462").build(),])

        PolicyOuterClass.Policy policy = PolicyOuterClass.Policy.newBuilder()
                .setName("Matching CVE (CVE-2019-3462)")
                .addLifecycleStages(PolicyOuterClass.LifecycleStage.DEPLOY)
                .addCategories("Testing")
                .setSeverity(PolicyOuterClass.Severity.HIGH_SEVERITY)
                .addEnforcementActions(PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT)
                .addPolicySections(
                        PolicySection.newBuilder().addPolicyGroups(policyGroup.build()).build())
                .build()
        policy = PolicyService.policyClient.postPolicy(
                PolicyServiceOuterClass.PostPolicyRequest.newBuilder()
                        .setPolicy(policy)
                        .build()
        )

        log.info("Policy created to scale-to-zero deployments with CVE-2019-3462")
        // Maximum time to wait for propagation to sensor
        Helpers.sleepWithRetryBackoff(5000 * (ClusterService.isOpenShift4() ? 4 : 1))
        log.info("Sensor and admission-controller _should_ have the policy update")

        def deployment = new Deployment()
                .setName("admission-suppress-cve")
                .setImage(image)

        def created = orchestrator.createDeploymentNoWait(deployment)
        assert !created

        // CVE needs to be saved into the DB
        Helpers.sleepWithRetryBackoff(1000)

        when:
        "Suppress CVE and check that the deployment can now launch"

        def cve = "CVE-2019-3462"
        if (Env.CI_JOBNAME.contains("postgres")) {
            CVEService.suppressImageCVE(cve)
        } else {
            CVEService.suppressCVE(cve)
        }

        log.info("Suppressed "+cve)
        // Allow propagation of CVE suppression and invalidation of cache
        Helpers.sleepWithRetryBackoff(5000 * (ClusterService.isOpenShift4() ? 4 : 1))
        log.info("Expect that the suppression has propagated")

        created = orchestrator.createDeploymentNoWait(deployment)
        assert created

        deleteDeploymentWithCaution(deployment)

        and:
        "Unsuppress CVE"
        if (Env.CI_JOBNAME.contains("postgres")) {
            CVEService.unsuppressImageCVE(cve)
        } else {
            CVEService.unsuppressCVE(cve)
        }
        log.info("Unsuppressed "+cve)
        // Allow propagation of CVE suppression and invalidation of cache
        Helpers.sleepWithRetryBackoff(15000 * (ClusterService.isOpenShift4() ? 4 : 1))
        log.info("Expect that the unsuppression has propagated")

        and:
        "Verify unsuppressing lets the deployment be blocked again"
        created = orchestrator.createDeploymentNoWait(deployment)

        then:
        assert !created

        cleanup:
        "Delete policy"
        PolicyService.policyClient.deletePolicy(Common.ResourceByID.newBuilder().setId(policy.id).build())

        if (created) {
            deleteDeploymentWithCaution(deployment)
        }

        // Add back enforcement
        Services.updatePolicyEnforcement(
                SEVERITY,
                [PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,]
        )

        where:
        "Data inputs are: "

        image | _
        "us.gcr.io/stackrox-ci/nginx:1.10.1@sha256:b53e7ca2f567bdb7f23dad7d183a3466532d32f7ddf82847783fad14f425e5d3" | _
        "us.gcr.io/stackrox-ci/nginx:1.10.1" | _
    }

    @Unroll
    @Category([BAT])
    def "Verify Admission Controller Enforcement on Updates (#desc)"() {
        when:
        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(true)
                .setEnforceOnUpdates(true)
                .setDisableBypass(!bypassable)
                .setScanInline(scan)
                .setTimeoutSeconds(timeout)
                .build()

        assert ClusterService.updateAdmissionController(ac)
        // Maximum time to wait for propagation to sensor
        Helpers.sleepWithRetryBackoff(5000)

        and:
        "Create the deployment with a harmless image"
        def modDeployment = deployment.clone()
        modDeployment.image = "quay.io/rhacs-eng/qa:busybox-1-28"
        def created = orchestrator.createDeploymentNoWait(modDeployment)
        assert created

        then:
        "Verify that the admission controller reacts to an update"
        def updated = orchestrator.updateDeploymentNoWait(deployment)
        assert updated == success

        cleanup:
        "Revert Cluster"
        if (created) {
            deleteDeploymentWithCaution(deployment)
        }

        where:
        "Data inputs are: "

        timeout | scan  | bypassable | deployment                   | success  | desc
        3       | false | false      | BUSYBOX_NO_BYPASS_DEPLOYMENT | false    | "no bypass annotation, non-bypassable"
        3       | false | false      | BUSYBOX_BYPASS_DEPLOYMENT    | false    | "bypass annotation, non-bypassable"
        3       | false | true       | BUSYBOX_BYPASS_DEPLOYMENT    | true     | "bypass annotation, bypassable"
        30      | true  | false      | GCR_NGINX_DEPLOYMENT         | false    | "nginx w/ inline scan"
    }

    @Unroll
    @Category([BAT])
    def "Verify Admission Controller Enforcement respects Cluster/Namespace scopes (match: #clusterMatch/#nsMatch)"() {
        when:
        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(true)
                .setScanInline(false)
                .setTimeoutSeconds(10)
                .build()

        assert ClusterService.updateAdmissionController(ac)

        and:
        "Update latest tag policy to respect scope"
        def latestTagPolicy = Services.getPolicyByName(LATEST_TAG)
        def scopedLatestTagPolicy = latestTagPolicy.toBuilder()
            .clearScope()
            .addScope(
                ScopeOuterClass.Scope.newBuilder()
                    .setCluster(clusterMatch ? clusterId : UUID.randomUUID().toString())
                    .setNamespace(nsMatch ? Constants.ORCHESTRATOR_NAMESPACE : "randomns")
            )
            .build()
        Services.updatePolicy(scopedLatestTagPolicy)

        // Maximum time to wait for propagation to sensor
        Helpers.sleepWithRetryBackoff(5000)

        then:
        "Create a deployment with a latest tag"
        def deployment = new Deployment()
                .setName("scoped-enforcement-${clusterMatch}-${nsMatch}")
                .setImage("busybox:latest")
                .addLabel("app", "test")
        def created = orchestrator.createDeploymentNoWait(deployment)

        and:
        "Verify that creation was only blocked if all scopes match"
        assert !created == (clusterMatch && nsMatch)

        cleanup:
        "Revert Cluster"
        if (created) {
            deleteDeploymentWithCaution(deployment)
        }
        Services.updatePolicy(latestTagPolicy)

        where:
        "Data inputs are: "

        clusterMatch | nsMatch
        false        | false
        false        | true
        true         | false
        true         | true
    }

    @Retry(count = 0)
    @Timeout(300)
    def "Verify admission controller does not impair cluster operations when unstable"() {
        when:
        "Check if test is applicable"
        and:
        "Stop the regular chaos monkey"
        if (chaosMonkey) {
            chaosMonkey.stop()
        }
        chaosMonkey = null

        and:
        "Configure admission controller"
        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(false)
                .setScanInline(false)
                .setTimeoutSeconds(10)
                .build()

        assert ClusterService.updateAdmissionController(ac)
        // Maximum time to wait for propagation to sensor
        Helpers.sleepWithRetryBackoff(5000)

        and:
        "Start a chaos monkey thread that kills _all_ ready admission control replicas with a short grace period"
        def killAllChaosMonkey = new ChaosMonkey(orchestrator, 0, 1L)

        then:
        "Verify deployment can be created"
        def deployment = MISC_DEPLOYMENT.clone()
        def created = orchestrator.createDeploymentNoWait(deployment, 10)
        assert created

        and:
        "Verify deployment can be modified reliably"
        for (int i = 0; i < 45; i++) {
            Helpers.sleepWithRetryBackoff(1000)
            deployment.addAnnotation("qa.stackrox.io/iteration", "${i}")
            assert orchestrator.updateDeploymentNoWait(deployment)
        }

        cleanup:
        "Stop chaos monkey"
        killAllChaosMonkey.stop()

        and:
        "Wait for all admission control replicas to become ready again"
        killAllChaosMonkey.waitForReady()

        and:
        "Delete deployment"
        if (created) {
            deleteDeploymentWithCaution(deployment)
        }
    }

    def deleteDeploymentWithCaution(Deployment deployment) {
        def timer = new Timer(30, 1)
        def deleted = false
        while (!deleted && timer.IsValid()) {
            try {
                orchestrator.deleteDeployment(deployment)
                deleted = true
            } catch (NullPointerException ignore) {
                log.info "Caught NPE while deleting deployment, retrying in 1s..."
            }
        }
        if (!deleted) {
            log.warn "Failed to delete deployment. Subsequent tests may be affected ..."
        }
    }

    @Category([SensorBounceNext])
    def "Verify admission controller performs image scans if Sensor is Unavailable"() {
        given:
        "Admission controller is enabled"
        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(true)
                .setScanInline(true)
                .setTimeoutSeconds(20)
                .build()

        assert ClusterService.updateAdmissionController(ac)
        // Maximum time to wait for propagation to sensor
        Helpers.sleepWithRetryBackoff(5000)

        and:
        "Sensor is unavailable"
        orchestrator.scaleDeployment("stackrox", "sensor", 0)
        orchestrator.waitForAllPodsToBeRemoved("stackrox", ["app": "sensor"], 30, 1)
        log.info("Sensor is now scaled to 0")

        and:
        "Admission controller is started from scratch w/o cached scans"
        def admCtrlDeploy = orchestrator.getOrchestratorDeployment("stackrox", "admission-control")
        def originalAdmCtrlReplicas = admCtrlDeploy.spec.replicas
        orchestrator.scaleDeployment("stackrox", "admission-control", 0)
        orchestrator.waitForAllPodsToBeRemoved("stackrox", admCtrlDeploy.spec.selector.matchLabels, 30, 1)
        log.info("Admission controller scaled to 0, was ${originalAdmCtrlReplicas}")
        orchestrator.scaleDeployment("stackrox", "admission-control", originalAdmCtrlReplicas)
        orchestrator.waitForPodsReady("stackrox", admCtrlDeploy.spec.selector.matchLabels,
                originalAdmCtrlReplicas, 30, 1)
        log.info("Admission controller scaled back to ${originalAdmCtrlReplicas}")

        when:
        "A deployment with an image violating a policy is created"
        def created = orchestrator.createDeploymentNoWait(GCR_NGINX_DEPLOYMENT)

        then:
        "Creation should fail"
        assert !created

        cleanup:
        orchestrator.scaleDeployment("stackrox", "sensor", 1)
        orchestrator.waitForPodsReady("stackrox", ["app": "sensor"], 1, 30, 1)
        if (created) {
            deleteDeploymentWithCaution(GCR_NGINX_DEPLOYMENT)
        }
    }
}
