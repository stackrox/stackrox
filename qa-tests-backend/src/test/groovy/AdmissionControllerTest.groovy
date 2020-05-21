import common.Constants
import groups.BAT
import io.fabric8.kubernetes.api.model.Pod
import io.stackrox.proto.storage.ClusterOuterClass.AdmissionControllerConfig
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.ScopeOuterClass
import objects.Deployment
import orchestratormanager.OrchestratorTypes
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ClusterService
import services.FeatureFlagService
import services.ImageIntegrationService
import spock.lang.Retry
import spock.lang.Shared
import spock.lang.Timeout
import spock.lang.Unroll
import util.Env
import util.Timer

import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.locks.ReentrantLock

class AdmissionControllerTest extends BaseSpecification {
    @Shared
    private List<PolicyOuterClass.EnforcementAction> latestTagEnforcements
    @Shared
    private List<PolicyOuterClass.EnforcementAction> cvssEnforcements
    @Shared
    private String gcrId
    @Shared
    private String clusterId

    private ChaosMonkey chaosMonkey

    static final private String GCR_NGINX         = "qagcrnginx"
    static final private String BUSYBOX_NO_BYPASS = "busybox-no-bypass"
    static final private String BUSYBOX_BYPASS    = "busybox-bypass"

    private final static String LATEST_TAG = "Latest tag"
    private final static String CVSS = "Fixable CVSS >= 7"

    static final private String ADMISSION_CONTROLLER_APP_NAME = "admission-control"

    static final private Deployment GCR_NGINX_DEPLOYMENT = new Deployment()
            .setName(GCR_NGINX)
            .setImage("us.gcr.io/stackrox-ci/nginx:1.10")
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
        .setImage("busybox:1.30")
        .addLabel("app", "random-busybox")

    def setupSpec() {
        Assume.assumeFalse(Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT)

        clusterId = ClusterService.getClusterId()
        assert clusterId

        latestTagEnforcements = Services.updatePolicyEnforcement(
                LATEST_TAG,
                [PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,]
        )

        cvssEnforcements = Services.updatePolicyEnforcement(
                CVSS,
                [PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,]
        )

        gcrId = ImageIntegrationService.addGcrRegistry()
        assert gcrId != null
    }

    def setup() {
        if (FeatureFlagService.isFeatureFlagEnabled("ROX_ADMISSION_CONTROL_SERVICE")) {
            // By default, operate with a chaos monkey that keeps one ready replica alive and deletes with a 10s grace
            // period, which should be sufficient for K8s to pick up readiness changes and update endpoints.
            chaosMonkey = new ChaosMonkey(1, 10L)
            chaosMonkey.waitForEffect()
        }
    }

    def cleanup() {
        if (chaosMonkey) {
            chaosMonkey.stop()
            chaosMonkey.waitForReady()
        }
    }

    def cleanupSpec() {
        Assume.assumeFalse(Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT)

        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(false)
                .build()

        assert ClusterService.updateAdmissionController(ac)

        Services.updatePolicyEnforcement(
                LATEST_TAG,
                latestTagEnforcements
        )

        Services.updatePolicyEnforcement(
                CVSS,
                cvssEnforcements
        )
        assert ImageIntegrationService.deleteImageIntegration(gcrId)
    }

    @Unroll
    @Category([BAT])
    def "Verify Admission Controller Config (#desc)"() {
        when:
        Assume.assumeFalse(Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT)

        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                                .setEnabled(true)
                                .setDisableBypass(!bypassable)
                                .setScanInline(scan)
                                .setTimeoutSeconds(timeout)
                            .build()

        assert ClusterService.updateAdmissionController(ac)
        // Maximum time to wait for propagation to sensor
        sleep 5000

        then:
        "Run deployment request"
        def created = orchestrator.createDeploymentNoWait(deployment)
        assert created == launched

        cleanup:
        "Revert Cluster"
        if (created) {
            def timer = new Timer(30, 1)
            def deleted = false
            while (!deleted && timer.IsValid()) {
                try {
                    orchestrator.deleteDeployment(deployment)
                    deleted = true
                } catch (NullPointerException ignore) {
                    println "Caught NPE while deleting deployment, retrying in 1s..."
                }
            }
            if (!deleted) {
                println "Warning: failed to delete deployment. Subsequent tests may be affected ..."
            }
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
    def "Verify Admission Controller Enforcement on Updates (#desc)"() {
        when:
        Assume.assumeFalse(Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT)
        Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled("ROX_ADMISSION_CONTROL_SERVICE"))
        Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled("ROX_ADMISSION_CONTROL_ENFORCE_ON_UPDATE"))

        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(true)
                .setEnforceOnUpdates(true)
                .setDisableBypass(!bypassable)
                .setScanInline(scan)
                .setTimeoutSeconds(timeout)
                .build()

        assert ClusterService.updateAdmissionController(ac)
        // Maximum time to wait for propagation to sensor
        sleep 5000

        and:
        "Create the deployment with a harmless image"
        def modDeployment = deployment.clone()
        modDeployment.image = "busybox:1.28"
        def created = orchestrator.createDeploymentNoWait(modDeployment)
        assert created

        then:
        "Verify that the admission controller reacts to an update"
        def updated = orchestrator.updateDeploymentNoWait(deployment)
        assert updated == success

        cleanup:
        "Revert Cluster"
        if (created) {
            def timer = new Timer(30, 1)
            def deleted = false
            while (!deleted && timer.IsValid()) {
                try {
                    orchestrator.deleteDeployment(deployment)
                    deleted = true
                } catch (NullPointerException ignore) {
                    println "Caught NPE while deleting deployment, retrying in 1s..."
                }
            }
            if (!deleted) {
                println "Warning: failed to delete deployment. Subsequent tests may be affected ..."
            }
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
        Assume.assumeFalse(Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT)

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
        sleep 5000

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
            def timer = new Timer(30, 1)
            def deleted = false
            while (!deleted && timer.IsValid()) {
                try {
                    orchestrator.deleteDeployment(deployment)
                    deleted = true
                } catch (NullPointerException ignore) {
                    println "Caught NPE while deleting deployment, retrying in 1s..."
                }
            }
            if (!deleted) {
                println "Warning: failed to delete deployment. Subsequent tests may be affected ..."
            }
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
        Assume.assumeFalse(Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT)
        Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled("ROX_ADMISSION_CONTROL_SERVICE"))

        and:
        "Stop the regular chaos monkey"
        chaosMonkey.stop()
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
        sleep 5000

        and:
        "Start a chaos monkey thread that kills _all_ ready admission control replicas with a short grace period"
        def killAllChaosMonkey = new ChaosMonkey(0, 1L)

        then:
        "Verify deployment can be created"
        def deployment = MISC_DEPLOYMENT.clone()
        def created = orchestrator.createDeploymentNoWait(deployment)
        assert created

        and:
        "Verify deployment can be modified reliably"
        for (int i = 0; i < 45; i++) {
            sleep 1000
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
            def timer = new Timer(30, 1)
            def deleted = false
            while (!deleted && timer.IsValid()) {
                try {
                    orchestrator.deleteDeployment(deployment)
                    deleted = true
                } catch (NullPointerException ignore) {
                    println "Caught NPE while deleting deployment, retrying in 1s..."
                }
            }
            if (!deleted) {
                println "Warning: failed to delete deployment. Subsequent tests may be affected ..."
            }
        }
    }

    class ChaosMonkey {
        def stopFlag = new AtomicBoolean()
        def lock = new ReentrantLock()
        def effectCond = lock.newCondition()

        Thread thread

        ChaosMonkey(int minReadyReplicas, Long gracePeriod) {
            def pods = orchestrator.getPods(Constants.STACKROX_NAMESPACE, ADMISSION_CONTROLLER_APP_NAME)
            assert pods.size() > 0, "There are no ${ADMISSION_CONTROLLER_APP_NAME} pods. " +
                "Did you enable ADMISSION_CONTROLLER when deploying?"

            thread = Thread.start {
                while (!stopFlag.get()) {
                    // Get the current ready, non-deleted pod replicas
                    def admCtrlPods = new ArrayList<Pod>(orchestrator.getPods(
                            Constants.STACKROX_NAMESPACE, ADMISSION_CONTROLLER_APP_NAME))
                    admCtrlPods.removeIf { it?.status?.containerStatuses[0]?.ready }

                    if (admCtrlPods.size() <= minReadyReplicas) {
                        lock.lock()
                        effectCond.signalAll()
                        lock.unlock()
                    }

                    admCtrlPods.removeIf { it?.metadata?.deletionTimestamp }

                    // If there are more than the minimum number of ready replicas, randomly pick some to delete
                    if (admCtrlPods.size() > minReadyReplicas) {
                        Collections.shuffle(admCtrlPods)
                        def podsToDelete = admCtrlPods.drop(minReadyReplicas)
                        podsToDelete.forEach {
                            orchestrator.deletePod(it.metadata.namespace, it.metadata.name, gracePeriod)
                        }
                    }
                    sleep 1000
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
                sleep 1000

                def admCtrlPods = orchestrator.getPods(Constants.STACKROX_NAMESPACE, ADMISSION_CONTROLLER_APP_NAME)
                if (admCtrlPods.size() < 3) {
                    continue
                }
                allReady = true
                for (def pod : admCtrlPods) {
                    if (!pod.status.containerStatuses[0].ready) {
                        allReady = false
                        break
                    }
                }
            }
            println "All admission control pod replicas ready"
        }
    }
}
