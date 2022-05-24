import groups.BAT
import io.stackrox.proto.storage.ClusterOuterClass.AdmissionControllerConfig
import io.stackrox.proto.storage.PolicyOuterClass
import objects.Deployment
import objects.GCRImageIntegration
import org.junit.experimental.categories.Category
import services.ClusterService
import services.ImageIntegrationService
import spock.lang.Shared
import spock.lang.Unroll
import util.Timer

class AdmissionControllerNoImageScanTest extends BaseSpecification {
    @Shared
    private List<PolicyOuterClass.EnforcementAction> noImageScansEnforcements
    @Shared
    private boolean noImageScansPolicyWasDisabled

    private final static String NO_IMAGE_SCANS = "Images with no scans"

    def setupSpec() {
        noImageScansEnforcements = Services.updatePolicyEnforcement(
                NO_IMAGE_SCANS,
                [PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,]
        )

        def noImageScansPolicy = Services.getPolicyByName(NO_IMAGE_SCANS)
        noImageScansPolicyWasDisabled = noImageScansPolicy.disabled
        def updatedPolicy = PolicyOuterClass.Policy.newBuilder(noImageScansPolicy)
            .setDisabled(false).build()
        Services.updatePolicy(updatedPolicy)
    }

    def cleanupSpec() {
        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(false)
                .build()

        assert ClusterService.updateAdmissionController(ac)

        Services.updatePolicyEnforcement(NO_IMAGE_SCANS, noImageScansEnforcements)
        def noImageScansPolicy = Services.getPolicyByName(NO_IMAGE_SCANS)
        def updatedPolicy = PolicyOuterClass.Policy.newBuilder(noImageScansPolicy)
                .setDisabled(noImageScansPolicyWasDisabled).build()
        Services.updatePolicy(updatedPolicy)
    }

    @Category([BAT])
    def "Verify Admission Controller Behavior for No Image Scans Policy"() {
        String gcrId

        // Note: This test is intentionally not using @Unroll in order to depend on the order of
        // operations.

        when:
        def ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(true)
                .setScanInline(false)
                .setTimeoutSeconds(30)

        assert ClusterService.updateAdmissionController(ac)
        // Maximum time to wait for propagation to sensor
        sleep 5000

        then:
        "Create deployment with a non-scannable image and inline scans disabled"
        assert launchDeploymentWithImage("non-existent:image")

        and:
        "Create deployment with a scannable image and inline scans disabled"
        assert launchDeploymentWithImage("us.gcr.io/stackrox-ci/nginx:1.12")

        when:
        "Enable inline scans"
        ac.scanInline = true
        assert ClusterService.updateAdmissionController(ac)
        sleep 5000

        and:
        "Enable registry integration"
        gcrId = GCRImageIntegration.createDefaultIntegration()
        assert gcrId != ""

        then:
        "Create deployment with a scannable image and inline scans enabled (w/ long timeout)"
        assert launchDeploymentWithImage("us.gcr.io/stackrox-ci/nginx:1.12")

        and:
        "Create deployment with a non-scannable image and inline scans enabled (w/ long timeout)"
        assert !launchDeploymentWithImage("non-existent:image")

        when:
        "Disable inline scans again"
        ac.scanInline = false
        assert ClusterService.updateAdmissionController(ac)
        sleep 5000

        then:
        "Create deployment with a non-scannable image and inline scans disabled"
        assert launchDeploymentWithImage("non-existent:image")

        and:
        "Create deployment with a scannable image and inline scans disabled"
        assert launchDeploymentWithImage("us.gcr.io/stackrox-ci/nginx:1.12")

        cleanup:
        if (gcrId) {
            ImageIntegrationService.deleteImageIntegration(gcrId)
        }
    }

    private boolean launchDeploymentWithImage(String img) {
        def deployment = new Deployment()
                .setName("adm-ctrl-img-scan-test")
                .setImage(img)
                .addLabel("app", "adm-ctrl-img-scan-test")

        def created = orchestrator.createDeploymentNoWait(deployment)
        if (created) {
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
            orchestrator.waitForDeploymentDeletion(deployment)
        }
        return created
    }
}
