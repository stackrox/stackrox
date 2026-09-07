import io.stackrox.proto.storage.ClusterOuterClass.AdmissionControllerConfig
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.SignatureIntegrationOuterClass

import objects.Deployment
import objects.GCRImageIntegration
import services.ClusterService
import services.ImageIntegrationService
import services.PolicyService
import services.SignatureIntegrationService
import util.Timer

import spock.lang.Ignore
import spock.lang.Shared
import spock.lang.Tag

@Ignore("ROX-29720 - gcr.io sunset, tests need migrated to Artifacts Registry results.")
@Tag("PZ")
class AdmissionControllerNoImageScanTest extends BaseSpecification {
    @Shared
    private List<PolicyOuterClass.EnforcementAction> noImageScansEnforcements
    @Shared
    private boolean noImageScansPolicyWasDisabled
    @Shared
    private String imageSignaturePolicyID
    @Shared
    private String imageSignatureIntegrationID

    // This key correlates to https://github.com/GoogleContainerTools/distroless/blob/main/cosign.pub.
    private final static String PUBLIC_KEY = """\
-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEWZzVzkb8A+DbgDpaJId/bOmV8n7Q
OqxYbK0Iro6GzSmOzxkn+N2AKawLyXi84WSwJQBK//psATakCgAQKkNTAA==
-----END PUBLIC KEY-----"""
    private final static String NO_IMAGE_SCANS = "Images with no scans"
    private final static String IMAGE_SIGNATURE = "Image Signature Test"

    private final static String NON_EXISTENT_IMAGE = "non-existent:image"
    private final static String IMAGE_WITH_SCANS = "us.gcr.io/acs-san-stackroxci/nginx:1.12"

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

        imageSignaturePolicyID = createImageSignaturePolicy()
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

        PolicyService.deletePolicy(imageSignaturePolicyID)

        SignatureIntegrationService.deleteSignatureIntegration(imageSignatureIntegrationID)
    }

    @Tag("BAT")
    def "Verify Admission Controller Behavior for No Image Scans Policy"() {
        String gcrId

        when:
        "Enable admission controller enforcement"
        def ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(true)
                .setTimeoutSeconds(30)

        assert ClusterService.updateAdmissionController(ac)
        sleep 5000

        and:
        "Enable registry integration"
        gcrId = GCRImageIntegration.createDefaultIntegration()
        assert gcrId != ""

        and:
        "Disable image signature policy"
        updateImageSignaturePolicy(true)
        sleep 5000

        then:
        "Create deployment with a scannable image"
        assert launchDeploymentWithImage(IMAGE_WITH_SCANS)

        and:
        "Create deployment with a non-scannable image"
        assert !launchDeploymentWithImage(NON_EXISTENT_IMAGE)

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
                orchestrator.deleteDeployment(deployment)
                deleted = true
            }
            if (!deleted) {
                log.warn "Failed to delete deployment. Subsequent tests may be affected ..."
            }
            orchestrator.waitForDeploymentDeletion(deployment)
        }
        return created
    }

    private String createImageSignaturePolicy() {
        String signatureIntegrationID = SignatureIntegrationService.createSignatureIntegration(
                SignatureIntegrationOuterClass.SignatureIntegration.newBuilder()
                        .setName("TEST")
                        .setCosign(SignatureIntegrationOuterClass.CosignPublicKeyVerification.newBuilder()
                                .addPublicKeys(SignatureIntegrationOuterClass.CosignPublicKeyVerification.PublicKey.
                                        newBuilder().setName("key").setPublicKeyPemEnc(PUBLIC_KEY).build())
                                .build()
                        )
                        .build()
        )
        assert signatureIntegrationID
        imageSignatureIntegrationID = signatureIntegrationID
        def policyValue = PolicyOuterClass.PolicyValue.newBuilder()
                .setValue(imageSignatureIntegrationID)
                .build()
        def policyGroup = PolicyOuterClass.PolicyGroup.newBuilder()
                .setFieldName("Image Signature Verified By")
                .setBooleanOperator(PolicyOuterClass.BooleanOperator.OR)
                .addValues(policyValue)
                .setNegate(false)
                .build()
        def policy = PolicyOuterClass.Policy.newBuilder()
            .addLifecycleStages(PolicyOuterClass.LifecycleStage.DEPLOY)
            .addCategories("Test")
            .setDisabled(false)
            .setSeverityValue(2)
            .setName(IMAGE_SIGNATURE)
            .addPolicySections(PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(policyGroup))
            .build()

        String policyID = PolicyService.createNewPolicy(policy)
        assert policyID

        Services.updatePolicyEnforcement(IMAGE_SIGNATURE,
                [PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,])

        return policyID
    }

    private static updateImageSignaturePolicy(boolean disabled) {
        def imageSignaturePolicy = Services.getPolicyByName(IMAGE_SIGNATURE)
        def updatedImageSignaturePolicy = PolicyOuterClass.Policy.newBuilder(imageSignaturePolicy)
                .setDisabled(disabled).build()
        Services.updatePolicy(updatedImageSignaturePolicy)
    }
}
