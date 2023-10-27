import static util.Helpers.withRetry

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.storage.ClusterOuterClass.AdmissionControllerConfig
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.PolicyGroup
import io.stackrox.proto.storage.PolicyOuterClass.PolicySection
import io.stackrox.proto.storage.PolicyOuterClass.PolicyValue
import io.stackrox.proto.storage.ScopeOuterClass

import objects.Deployment
import services.CVEService
import services.ClusterService
import services.ImageService
import services.PolicyService
import util.ApplicationHealth
import util.ChaosMonkey
import util.Timer

import spock.lang.IgnoreIf
import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Timeout
import spock.lang.Unroll

@Tag("PZ")
class AdmissionControllerTest extends BaseSpecification {
    @Shared
    private String clusterId
    @Shared
    private List<String> createdPolicyIds

    private ChaosMonkey chaosMonkey

    static final private String TEST_NAMESPACE = "qa-admission-controller-test"

    static final private String NGINX                = "qanginx"
    static final private String NGINX_IMAGE          = "quay.io/rhacs-eng/qa-multi-arch:nginx-1.21.1"
    static final private String NGINX_IMAGE_WITH_SHA = "quay.io/rhacs-eng/qa-multi-arch:nginx-1.21.1"+
                                    "@sha256:6bf47794f923462389f5a2cda49cf5777f736db8563edc3ff78fb9d87e6e22ec"
    static final private String NGINX_CVE            = "CVE-2017-16932"

    static final private String BUSYBOX_NO_BYPASS        = "busybox-no-bypass"
    static final private String BUSYBOX_BYPASS           = "busybox-bypass"
    static final private String BUSYBOX_LATEST_TAG_IMAGE = "quay.io/rhacs-eng/qa-multi-arch-busybox:latest"

    private final static String CLONED_POLICY_SUFFIX = "(${TEST_NAMESPACE})"
    private final static String LATEST_TAG = "Latest tag"
    private final static String LATEST_TAG_FOR_TEST = "Latest tag ${CLONED_POLICY_SUFFIX}"
    private final static String SEVERITY = "Fixable Severity at least Important"
    private final static String SEVERITY_FOR_TEST = "Fixable Severity at least Important ${CLONED_POLICY_SUFFIX}"

    static final private Deployment NGINX_DEPLOYMENT = new Deployment()
            .setName(NGINX)
            .setNamespace(TEST_NAMESPACE)
            .setImage(NGINX_IMAGE)
            .addLabel("app", "test")

    static final private Deployment BUSYBOX_NO_BYPASS_DEPLOYMENT = new Deployment()
            .setName(BUSYBOX_NO_BYPASS)
            .setNamespace(TEST_NAMESPACE)
            .setImage(BUSYBOX_LATEST_TAG_IMAGE)
            .addLabel("app", "test")

    static final private Deployment BUSYBOX_BYPASS_DEPLOYMENT = new Deployment()
            .setName(BUSYBOX_BYPASS)
            .setNamespace(TEST_NAMESPACE)
            .setImage(BUSYBOX_LATEST_TAG_IMAGE)
            .addLabel("app", "test")
            .addAnnotation("admission.stackrox.io/break-glass", "yay")

    static final private Deployment MISC_DEPLOYMENT = new Deployment()
            .setName("random-busybox")
            .setNamespace(TEST_NAMESPACE)
            .setImage("quay.io/rhacs-eng/qa-multi-arch:busybox-1-30")
            .addLabel("app", "random-busybox")

    def setupSpec() {
        clusterId = ClusterService.getClusterId()
        assert clusterId

        // Create namespace scoped policies for test based on "Latest Tag" and
        // "Fixable Severity at least Important"
        createdPolicyIds = []
        for (policy : [Services.getPolicyByName(LATEST_TAG), Services.getPolicyByName(SEVERITY)]) {
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
            createdPolicyIds.add(policyID)
        }
        // Wait for propagation to sensor
        sleep(10000 * (ClusterService.isOpenShift4() ? 4 : 1))

        // Pre run scan to avoid timeouts with inline scans in the tests below
        ImageService.scanImage(NGINX_IMAGE)

        orchestrator.ensureNamespaceExists(TEST_NAMESPACE)
    }

    def setup() {
        // https://stack-rox.atlassian.net/browse/ROX-7026 - Disable ChaosMonkey
        // By default, operate with a chaos monkey that keeps one ready replica alive and deletes with a 10s grace
        // period, which should be sufficient for K8s to pick up readiness changes and update endpoints.
        // chaosMonkey = new ChaosMonkey(orchestrator, 1, 10L)
    }

    def cleanup() {
        if (chaosMonkey) {
            chaosMonkey.stop()
            chaosMonkey.waitForReady()
        }
    }

    def cleanupSpec() {
        orchestrator.deleteNamespace(TEST_NAMESPACE)

        for (policyID in createdPolicyIds) {
            PolicyService.deletePolicy(policyID)
        }

        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(false)
                .build()

        assert ClusterService.updateAdmissionController(ac)
    }

    def prepareChaosMonkey() {
        // We cannot do this in setup() because we need to make sure chaos monkey
        // is back up on retries after being stopped in "cleanup:".
        if (chaosMonkey) {
            chaosMonkey.start()
            chaosMonkey.waitForEffect()
        }
    }

    @Unroll
    @Tag("BAT")
    @Tag("Parallel")
    def "Verify Admission Controller Config: #desc"() {
        when:
        prepareChaosMonkey()

        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                                .setEnabled(true)
                                .setDisableBypass(!bypassable)
                                .setScanInline(scan)
                                .setTimeoutSeconds(timeout)
                            .build()

        assert ClusterService.updateAdmissionController(ac)
        // Maximum time to wait for propagation to sensor
        sleep(5000)

        then:
        "Run deployment request"
        def created = orchestrator.createDeploymentNoWait(deployment)
        assert created == launched

        cleanup:
        "Stop ChaosMonkey ASAP to not lose logs"
        if (chaosMonkey) {
            chaosMonkey.stop()
        }

        and:
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
        30      | true  | false      | NGINX_DEPLOYMENT             | false    | "nginx w/ inline scan"
    }

    @Unroll
    @Tag("BAT")
    @Tag("Parallel")
    @IgnoreIf({ Env.ROX_VULN_MGMT_UNIFIED_CVE_DEFERRAL == "true" })
    def "Verify CVE snoozing applies to images scanned by admission controller #image"() {
        given:
        "Chaos monkey is prepared"
        prepareChaosMonkey()

        and:
        "Scan image"
        ImageService.scanImage(image)

        "Create policy looking for a specific CVE"
        // We don't want to block on SEVERITY
        Services.updatePolicyEnforcement(
                SEVERITY_FOR_TEST,
                []
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
        policyGroup.addAllValues([PolicyValue.newBuilder().setValue(NGINX_CVE).build(),])

        String policyName = "Matching CVE (${NGINX_CVE})"
        PolicyOuterClass.Policy policy = PolicyOuterClass.Policy.newBuilder()
                .setName(policyName)
                .addLifecycleStages(PolicyOuterClass.LifecycleStage.DEPLOY)
                .addCategories("DevOps Best Practices")
                .setSeverity(PolicyOuterClass.Severity.HIGH_SEVERITY)
                .addEnforcementActions(PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT)
                .addScope(ScopeOuterClass.Scope.newBuilder().setNamespace(TEST_NAMESPACE))
                .addPolicySections(
                        PolicySection.newBuilder().addPolicyGroups(policyGroup.build()).build())
                .build()

        String policyID = PolicyService.createNewPolicy(policy)
        assert policyID

        log.info("Policy created to scale-to-zero deployments with ${NGINX_CVE}")
        // Maximum time to wait for propagation to sensor
        sleep(15000 * (ClusterService.isOpenShift4() ? 4 : 1))
        log.info("Sensor and admission-controller _should_ have the policy update")

        def deployment = new Deployment()
                .setName("admission-suppress-cve")
                .setNamespace(TEST_NAMESPACE)
                .setImage(image)

        def created = orchestrator.createDeploymentNoWait(deployment)
        assert !created

        // CVE needs to be saved into the DB
        sleep(1000)

        when:
        "Suppress CVE and check that the deployment can now launch"

        def cve = NGINX_CVE
        CVEService.suppressImageCVE(cve)

        log.info("Suppressed "+cve)
        // Allow propagation of CVE suppression and invalidation of cache
        sleep(5000 * (ClusterService.isOpenShift4() ? 4 : 1))
        log.info("Expect that the suppression has propagated")

        created = orchestrator.createDeploymentNoWait(deployment)
        assert created

        deleteDeploymentWithCaution(deployment)

        and:
        "Unsuppress CVE"
        CVEService.unsuppressImageCVE(cve)

        log.info("Unsuppressed "+cve)
        // Allow propagation of CVE suppression and invalidation of cache
        sleep(15000 * (ClusterService.isOpenShift4() ? 4 : 1))
        log.info("Expect that the unsuppression has propagated")

        and:
        "Verify unsuppressing lets the deployment be blocked again"
        created = orchestrator.createDeploymentNoWait(deployment)

        then:
        assert !created

        cleanup:
        "Stop ChaosMonkey ASAP to not lose logs"
        if (chaosMonkey) {
            chaosMonkey.stop()
        }

        and:
        "Delete policy"
        PolicyService.policyClient.deletePolicy(Common.ResourceByID.newBuilder().setId(policyID).build())

        if (created) {
            deleteDeploymentWithCaution(deployment)
        }

        // Add back enforcement
        Services.updatePolicyEnforcement(SEVERITY_FOR_TEST,
                [PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,]
        )

        where:
        "Data inputs are: "

        image | _
        NGINX_IMAGE_WITH_SHA | _
        NGINX_IMAGE | _
    }

    @Unroll
    @Tag("BAT")
    @Tag("Parallel")
    def "Verify Admission Controller Enforcement on Updates: #desc"() {
        when:
        prepareChaosMonkey()

        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(true)
                .setEnforceOnUpdates(true)
                .setDisableBypass(!bypassable)
                .setScanInline(scan)
                .setTimeoutSeconds(timeout)
                .build()

        assert ClusterService.updateAdmissionController(ac)
        // Maximum time to wait for propagation to sensor
        sleep(5000)

        and:
        "Create the deployment with a harmless image"
        def modDeployment = deployment.clone()
        modDeployment.image = "quay.io/rhacs-eng/qa-multi-arch:busybox-1-28"
        def created = orchestrator.createDeploymentNoWait(modDeployment)
        assert created

        then:
        "Verify that the admission controller reacts to an update"
        def updated = orchestrator.updateDeploymentNoWait(deployment)
        assert updated == success

        cleanup:
        "Stop ChaosMonkey ASAP to not lose logs"
        if (chaosMonkey) {
            chaosMonkey.stop()
        }

        and:
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
        30      | true  | false      | NGINX_DEPLOYMENT             | false    | "nginx w/ inline scan"
    }

    @Unroll
    @Tag("BAT")
    @Tag("Parallel")
    def "Verify Admission Controller Enforcement respects Cluster/Namespace scopes: match: #clusterMatch/#nsMatch"() {
        when:
        prepareChaosMonkey()

        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(true)
                .setScanInline(false)
                .setTimeoutSeconds(10)
                .build()

        assert ClusterService.updateAdmissionController(ac)

        and:
        "Update latest tag policy to respect scope"
        def latestTagPolicy = Services.getPolicyByName(LATEST_TAG_FOR_TEST)
        def scopedLatestTagPolicy = latestTagPolicy.toBuilder()
            .clearScope()
            .addScope(
                ScopeOuterClass.Scope.newBuilder()
                    .setCluster(clusterMatch ? clusterId : UUID.randomUUID().toString())
                    .setNamespace(nsMatch ? TEST_NAMESPACE : "randomns")
            )
            .build()
        Services.updatePolicy(scopedLatestTagPolicy)

        // Maximum time to wait for propagation to sensor
        sleep(5000)

        then:
        "Create a deployment with a latest tag"
        def deployment = new Deployment()
                .setName("scoped-enforcement-${clusterMatch}-${nsMatch}")
                .setNamespace(TEST_NAMESPACE)
                .setImage(BUSYBOX_LATEST_TAG_IMAGE)
                .addLabel("app", "test")
        def created = orchestrator.createDeploymentNoWait(deployment)

        and:
        "Verify that creation was only blocked if all scopes match"
        assert !created == (clusterMatch && nsMatch)

        cleanup:
        "Stop ChaosMonkey ASAP to not lose logs"
        if (chaosMonkey) {
            chaosMonkey.stop()
        }

        and:
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

    @Tag("Parallel")
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
        sleep(5000)

        and:
        "Start a chaos monkey thread that kills _all_ ready admission control replicas with a short grace period"
        def killAllChaosMonkey = new ChaosMonkey(orchestrator, 0, 1L)
        killAllChaosMonkey.start()
        killAllChaosMonkey.waitForEffect()

        then:
        "Verify deployment can be created"
        def deployment = MISC_DEPLOYMENT.clone()
        def created = orchestrator.createDeploymentNoWait(deployment, 10)
        assert created

        and:
        "Verify deployment can be modified reliably"
        for (int i = 0; i < 45; i++) {
            sleep(1000)
            deployment.addAnnotation("qa.stackrox.io/iteration", "${i}")
            assert orchestrator.updateDeploymentNoWait(deployment, 10)
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

    @Tag("SensorBounceNext")
    def "Verify admission controller performs image scans if Sensor is Unavailable"() {
        given:
        "Chaos monkey is prepared"
        prepareChaosMonkey()

        and:
        "Admission controller is enabled"
        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(true)
                .setScanInline(true)
                .setTimeoutSeconds(20)
                .build()

        assert ClusterService.updateAdmissionController(ac)
        // Maximum time to wait for propagation to sensor
        sleep(5000)

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

        and:
        "Admission controller is ready for work"
        ApplicationHealth ah = new ApplicationHealth(orchestrator, 60)
        ah.waitForAdmissionControllerHealthiness()

        when:
        "A deployment with an image violating a policy is created"
        def created
        def consecutiveRejectionsCount = 0
        withRetry(40, 5) {
            created = orchestrator.createDeploymentNoWait(NGINX_DEPLOYMENT)
            if (created) {
                consecutiveRejectionsCount = 0
                deleteDeploymentWithCaution(NGINX_DEPLOYMENT)
            }
            else {
                consecutiveRejectionsCount++
            }
            assert !created
            assert consecutiveRejectionsCount == 5
        }

        then:
        "Creation should fail"
        assert !created

        and:
        "Creation should fail consistently"
        assert consecutiveRejectionsCount == 5

        cleanup:
        "Stop ChaosMonkey ASAP to not lose logs"
        if (chaosMonkey) {
            chaosMonkey.stop()
        }

        and:
        "Restore sensor"
        orchestrator.scaleDeployment("stackrox", "sensor", 1)
        orchestrator.waitForPodsReady("stackrox", ["app": "sensor"], 1, 30, 1)

        and:
        "Delete nginx deployment"
        if (created) {
            deleteDeploymentWithCaution(NGINX_DEPLOYMENT)
        }
    }
}
