import io.stackrox.proto.storage.ImageOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.ScopeOuterClass

import objects.Deployment
import services.ClusterService
import services.ImageService
import services.PolicyService
import util.Timer

import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Unroll

@Tag("PZ")
class AdmissionControllerTest extends BaseSpecification {
    @Shared
    private String clusterId
    @Shared
    private List<String> createdPolicyIds

    static final private String TEST_NAMESPACE = "qa-admission-controller-test"

    static final private String SCAN_INLINE_DEPLOYMENT_NAME = "scan-inline"
    // An image name with @sha... appended is used for admission control that
    // requires inline scanning. This ensures that metadata gets cached after initial scan,
    // and therefore central does not connect to an external registry during test, which avoids flakes.
    static final private String SCAN_INLINE_IMAGE_NAME_WITH_SHA = TEST_IMAGE_NAME_WITH_SHA
    static final private String SCAN_INLINE_IMAGE_SHA = TEST_IMAGE_SHA

    static final private String BUSYBOX_NO_BYPASS        = "busybox-no-bypass"
    static final private String BUSYBOX_BYPASS           = "busybox-bypass"
    static final private String BUSYBOX_LATEST_TAG_IMAGE = "quay.io/rhacs-eng/qa-multi-arch-busybox:latest"

    private final static String CLONED_POLICY_SUFFIX = "(${TEST_NAMESPACE})"
    private final static String LATEST_TAG = "Latest tag"
    private final static String LATEST_TAG_FOR_TEST = "Latest tag ${CLONED_POLICY_SUFFIX}"
    private final static String SEVERITY = "Fixable Severity at least Important"

    static final private Deployment SCAN_INLINE_DEPLOYMENT = new Deployment()
            .setName(SCAN_INLINE_DEPLOYMENT_NAME)
            .setNamespace(TEST_NAMESPACE)
            .setImagePrefetcherAffinity()
            .setImage(SCAN_INLINE_IMAGE_NAME_WITH_SHA)
            .addLabel("app", "test")

    static final private Deployment BUSYBOX_NO_BYPASS_DEPLOYMENT = new Deployment()
            .setName(BUSYBOX_NO_BYPASS)
            .setNamespace(TEST_NAMESPACE)
            .setImagePrefetcherAffinity()
            .setImage(BUSYBOX_LATEST_TAG_IMAGE)
            .addLabel("app", "test")

    static final private Deployment BUSYBOX_BYPASS_DEPLOYMENT = new Deployment()
            .setName(BUSYBOX_BYPASS)
            .setNamespace(TEST_NAMESPACE)
            .setImagePrefetcherAffinity()
            .setImage(BUSYBOX_LATEST_TAG_IMAGE)
            .addLabel("app", "test")
            .addAnnotation("admission.stackrox.io/break-glass", "yay")

    def setupSpec() {
        clusterId = ClusterService.getClusterId()
        assert clusterId

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
        // Wait for policy propagation to sensor and admission controller
        sleep(10000 * (ClusterService.isOpenShift4() ? 4 : 1))

        // Pre-scan the image so Central has cached scan results for CVE-based policy evaluation.
        ImageService.scanImage(SCAN_INLINE_IMAGE_NAME_WITH_SHA)

        ImageOuterClass.Image image = ImageService.getImage(SCAN_INLINE_IMAGE_SHA, false)
        assert image
        assert !image.getNotesList().contains(ImageOuterClass.Image.Note.MISSING_METADATA)

        orchestrator.ensureNamespaceExists(TEST_NAMESPACE)
    }

    def cleanupSpec() {
        orchestrator.deleteNamespace(TEST_NAMESPACE)

        for (policyID in createdPolicyIds) {
            PolicyService.deletePolicy(policyID)
        }
    }

    @Unroll
    @Tag("BAT")
    @Tag("Parallel")
    def "Verify admission controller enforcement on create: #desc"() {
        when:
        "Create a deployment that violates an enforced policy"
        def created = orchestrator.createDeploymentNoWait(deployment)

        then:
        "Verify the admission controller allows or blocks based on policy and bypass annotation"
        assert created == launched

        cleanup:
        if (created) {
            deleteDeploymentWithCaution(deployment)
        }

        where:
        deployment                   | launched | desc
        BUSYBOX_NO_BYPASS_DEPLOYMENT | false    | "blocked by enforced latest tag policy"
        BUSYBOX_BYPASS_DEPLOYMENT    | true     | "allowed with bypass annotation"
        SCAN_INLINE_DEPLOYMENT       | false    | "blocked by enforced severity policy (cached scan)"
    }

    @Unroll
    @Tag("BAT")
    @Tag("Parallel")
    def "Verify admission controller enforcement on update: #desc"() {
        when:
        "Create a deployment with a non-violating image"
        def modDeployment = deployment.clone()
        modDeployment.image = "quay.io/rhacs-eng/qa-multi-arch:busybox-1-28"
        def created = orchestrator.createDeploymentNoWait(modDeployment)
        assert created

        then:
        "Update to a violating image and verify enforcement"
        def updated = orchestrator.updateDeploymentNoWait(deployment)
        assert updated == success

        cleanup:
        if (created) {
            deleteDeploymentWithCaution(deployment)
        }

        where:
        deployment                   | success | desc
        BUSYBOX_NO_BYPASS_DEPLOYMENT | false   | "blocked by enforced latest tag policy"
        BUSYBOX_BYPASS_DEPLOYMENT    | true    | "allowed with bypass annotation"
    }

    @Unroll
    @Tag("BAT")
    @Tag("Parallel")
    def "Verify admission controller enforcement respects Cluster/Namespace scopes: match: #clusterMatch/#nsMatch"() {
        when:
        "Update latest tag policy scope"
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

        // Wait for policy propagation to sensor
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
        if (created) {
            deleteDeploymentWithCaution(deployment)
        }
        Services.updatePolicy(latestTagPolicy)

        where:
        clusterMatch | nsMatch
        false        | false
        false        | true
        true         | false
        true         | true
    }

    def deleteDeploymentWithCaution(Deployment deployment) {
        def timer = new Timer(30, 1)
        def deleted = false
        while (!deleted && timer.IsValid()) {
            orchestrator.deleteDeployment(deployment)
            deleted = true
        }
        if (!deleted) {
            log.warn "Failed to delete deployment. Subsequent tests may be affected ..."
        }
    }

}
