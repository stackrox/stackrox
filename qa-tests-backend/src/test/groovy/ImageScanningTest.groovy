import static services.ClusterService.DEFAULT_CLUSTER_NAME

import io.grpc.StatusRuntimeException

import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.ImageIntegrationOuterClass
import io.stackrox.proto.storage.ImageOuterClass
import io.stackrox.proto.storage.Vulnerability

import common.Constants
import groups.BAT
import groups.TARGET
import groups.Integration
import objects.AnchoreScannerIntegration
import objects.ClairScannerIntegration
import objects.Deployment
import objects.AzureRegistryIntegration
import objects.ECRRegistryIntegration
import objects.GCRImageIntegration
import objects.GoogleArtifactRegistry
import objects.QuayImageIntegration
import objects.Secret
import objects.StackroxScannerIntegration
import services.ClusterService
import services.ImageIntegrationService
import services.ImageService
import services.PolicyService
import util.Env
import util.Helpers
import util.Timer

import org.junit.Assume
import org.junit.AssumptionViolatedException
import org.junit.experimental.categories.Category
import spock.lang.Shared
import spock.lang.Unroll

class ImageScanningTest extends BaseSpecification {

    static final private String RHEL7_IMAGE =
            "richxsl/rhel7@sha256:8f3aae325d2074d2dc328cb532d6e7aeb0c588e15ddf847347038fe0566364d6"
    static final private String GCR_IMAGE   = "us.gcr.io/stackrox-ci/qa/registry-image:0.2"
    static final private String NGINX_IMAGE = "nginx:1.12.1"
    static final private String OCI_IMAGE   = "quay.io/rhacs-eng/qa:oci-manifest"
    static final private String AR_IMAGE    = "us-west1-docker.pkg.dev/stackrox-ci/artifact-registry-test1/nginx:1.17"
    static final private String CENTOS_IMAGE = "quay.io/rhacs-eng/qa:centos7-base"
    static final private String CENTOS_ECHO_IMAGE = "quay.io/rhacs-eng/qa:centos7-base-echo"

    static final private List<String> POLICIES = [
            "ADD Command used instead of COPY",
            "Secure Shell (ssh) Port Exposed in Image",
    ]

    static final private Integer WAIT_FOR_VIOLATION_TIMEOUT = isRaceBuild() ? 450 : 30

    static final private Map<String, Deployment> DEPLOYMENTS = [
            "quay": new Deployment()
                    .setName("quay-image-scanning-test")
                    .setImage("quay.io/stackrox/testing:registry-image-no-secrets")
                    .addLabel("app", "quay-image-scanning-test")
                    .addImagePullSecret("quay-image-scanning-test"),
            "gcr": new Deployment()
                    .setName("gcr-image-scanning-test")
                    .setImage("us.gcr.io/stackrox-ci/qa/registry-image:0.3")
                    .addLabel("app", "gcr-image-scanning-test")
                    .addImagePullSecret("gcr-image-scanning-test"),
            "ecr": new Deployment()
                    .setName("ecr-image-registry-test")
                    .setImage("${Env.mustGetAWSECRRegistryID()}.dkr.ecr.${Env.mustGetAWSECRRegistryRegion()}." +
                            "amazonaws.com/stackrox-qa-ecr-test:registry-image-no-secrets")
                    .addLabel("app", "ecr-image-registry-test")
                    .addImagePullSecret("ecr-image-registry-test"),
            "acr": new Deployment()
                    .setName("acr-image-registry-test")
                    .setImage("stackroxci.azurecr.io/stackroxci/registry-image:0.3")
                    .addLabel("app", "acr-image-registry-test")
                    .addImagePullSecret("acr-image-registry-test"),
    ]

    static final private Map<String, Secret> IMAGE_PULL_SECRETS = [
            "quay": new Secret(
                    name: "quay-image-scanning-test",
                    namespace: Constants.ORCHESTRATOR_NAMESPACE,
                    username: "stackrox+circleci_apollo",
                    password: Env.mustGet("QUAY_PASSWORD"),
                    server: "https://quay.io"),
            "gcr": new Secret(
                    name: "gcr-image-scanning-test",
                    namespace: Constants.ORCHESTRATOR_NAMESPACE,
                    username: "_json_key",
                    password: Env.mustGet("GOOGLE_CREDENTIALS_GCR_SCANNER"),
                    server: "https://us.gcr.io"),
            "ecr": new Secret(
                    name: "ecr-image-registry-test",
                    namespace: Constants.ORCHESTRATOR_NAMESPACE,
                    username: "AWS",
                    password: Env.mustGetAWSECRDockerPullPassword(),
                    server: "https://${Env.mustGetAWSECRRegistryID()}.dkr.ecr."+
                            "${Env.mustGetAWSECRRegistryRegion()}.amazonaws.com"),
            "acr": new Secret(
                    name: "acr-image-registry-test",
                    namespace: Constants.ORCHESTRATOR_NAMESPACE,
                    username: "stackroxci",
                    password: Env.mustGet("AZURE_REGISTRY_PASSWORD"),
                    server: "https://stackroxci.azurecr.io"),
    ]

    @Shared
    static final private List<String> UPDATED_POLICIES = []

    def setupSpec() {
        ImageIntegrationService.deleteStackRoxScannerIntegrationIfExists()
        removeGCRImagePullSecret()
        ImageIntegrationService.deleteAutoRegisteredGCRIntegrationIfExists()

        // Enable specific policies to test image integrations
        for (String policy : POLICIES) {
            if (Services.setPolicyDisabled(policy, false)) {
                UPDATED_POLICIES.add(policy)
            }
        }
    }

    def cleanupSpec() {
        ImageIntegrationService.addStackroxScannerIntegration()
        addGCRImagePullSecret()

        for (String policy : UPDATED_POLICIES) {
            Services.setPolicyDisabled(policy, true)
        }
    }

    private Secret secret
    private Deployment deployment
    private List<String> integrationIds
    private String imageToCleanup
    private Boolean deleteStackroxScanner

    def setup() {
        secret = null
        deployment = null
        integrationIds = new ArrayList<String>()
        imageToCleanup = null
        deleteStackroxScanner = false
    }

    def cleanup() {
        println "Post test cleanup:"
        if (secret != null) {
            orchestrator.deleteSecret(secret.name, secret.namespace)
        }
        if (deployment != null) {
            orchestrator.deleteAndWaitForDeploymentDeletion(deployment)
            imageToCleanup = deployment.image
        }
        if (imageToCleanup != null) {
            ImageService.clearImageCaches()
            try {
                ImageService.deleteImagesWithRetry(SearchServiceOuterClass.RawQuery.newBuilder()
                        .setQuery("Image:${imageToCleanup}").build(), true)
            } catch (e) {
                println "Image delete threw an exception: ${e}, this is OK for some retry cases."
            }
        }
        integrationIds.each { ImageIntegrationService.deleteImageIntegration(it) }

        if (deleteStackroxScanner) {
            ImageIntegrationService.deleteStackRoxScannerIntegrationIfExists()
        }
    }

    def cleanupSetupForRetry() {
        if (Helpers.getAttemptCount() > 1) {
            println "Cleaning up"
            cleanup()
            println "Done cleaning up and sleeping"
            sleep(10000)
            println "Setting up"
            setup()
            println "Done setting up"
        }
    }

    @Unroll
    @Category([BAT, Integration, TARGET])
    def "Verify Image Registry+Scanner Integrations: #testName"() {
        cleanupSetupForRetry()

        given:
        "Get deployment details used to test integration"
        assert IMAGE_PULL_SECRETS.containsKey(integration)
        secret = IMAGE_PULL_SECRETS.get(integration)
        orchestrator.createImagePullSecret(secret)

        when:
        "validate auto-generated registry was created"
        def autogeneratedId = expectAutoGeneratedRegistry(secret)
        if (!testName.contains("keep-autogenerated")) {
            integrationIds.add(autogeneratedId)
        }

        assert DEPLOYMENTS.containsKey(integration)
        deployment = DEPLOYMENTS.get(integration)
        deployment = deployment.clone()
        deployment.setName("${testName}--${deployment.name}")
        orchestrator.createDeployment(deployment)

        then:
        assert Services.waitForDeployment(deployment)
        assert deployment

        and:
        "validate registry based image metadata"
        def imageDigest
        try {
            withRetry(15, 2) {
                imageDigest = ImageService.getImages().find { it.name == deployment.image }
                assert imageDigest?.id
            }
        } catch (Exception e) {
            if (strictIntegrationTesting) {
                throw (e)
            }
            throw new AssumptionViolatedException("Failed to pull the image using ${integration}. Skipping test!", e)
        }
        ImageOuterClass.Image imageDetail = ImageService.getImage(imageDigest?.id)
        assert imageDetail.metadata?.v1?.layersCount >= 1
        assert imageDetail.metadata?.layerShasCount >= 1

        and:
        "validate expected violations based on dockerfile"
        for (String policy : POLICIES) {
            assert Services.waitForViolation(deployment.name, policy, WAIT_FOR_VIOLATION_TIMEOUT)
        }

        when:
        "Add scanner integration"
        addIntegrationClosure.each {
            def id = it()
            integrationIds.add(id) }
        PolicyService.reassessPolicies()
        ImageService.scanImage(deployment.image)
        imageDetail = ImageService.getImage(ImageService.getImages().find { it.name == deployment.image }?.id)

        then:
        "validate scan results for the image"
        Timer t = new Timer(20, 3)
        while (imageDetail?.scan?.componentsCount == 0 && t.IsValid()) {
            println "waiting on scan details..."
            sleep 3000
            ImageService.scanImage(deployment.image)
            imageDetail = ImageService.getImage(ImageService.getImages().find { it.name == deployment.image }?.id)
        }
        assert imageDetail.metadata.dataSource.id != ""
        assert imageDetail.metadata.dataSource.name != ""
        assert imageDetail.scan.dataSource.id != ""
        assert imageDetail.scan.dataSource.name != ""
        try {
            assert imageDetail.scan.componentsCount > 0
        } catch (Exception e) {
            if (strictIntegrationTesting) {
                throw (e)
            }
            throw new AssumptionViolatedException("Failed to scan the image using ${integration}. Skipping test!", e)
        }

        and:
        "validate the existence of expected CVEs"
        for (String cve : cves) {
            println "Validating existence of ${cve} cve..."
            ImageOuterClass.EmbeddedImageScanComponent component = imageDetail.scan.componentsList.find {
                component -> component.vulnsList.find { vuln -> vuln.cve == cve }
            }
            assert component
            Vulnerability.EmbeddedVulnerability vuln = component.vulnsList.find { it.cve == cve }
            assert vuln

            assert vuln.summary && vuln.summary != ""
            assert 0.0 <= vuln.cvss && vuln.cvss <= 10.0
            assert vuln.link && vuln.link != ""
        }
        assert imageDetail.components >= components
        assert imageDetail.cves >= totalCves
        assert imageDetail.fixableCves >= fixable

        where:
        "Data inputs:"

        testName                        | integration |
                addIntegrationClosure                                                                             |
                components | totalCves | fixable

        // ROX-9448 - disable Quay until scanning is fixed
        // "quay-keep-autogenerated"       | "quay" |
        //         [{ QuayImageIntegration.createDefaultIntegration() },] |
        //         165 | 182 | 28

        // "quay"                          | "quay" |
        //         [{ QuayImageIntegration.createDefaultIntegration() },]                                      |
        //         165 | 182 | 28

        // "quay-fully-qualified-endpoint" | "quay" |
        //         [{ QuayImageIntegration.createCustomIntegration(endpoint: "https://quay.io/") },]           |
        //         165 | 182 | 28

        // "quay-insecure"                 | "quay" |
        //         [{ QuayImageIntegration.createCustomIntegration(insecure: true) },]                         |
        //         165 | 182 | 28

        // "quay-duplicate"                | "quay" |
        //         [{ QuayImageIntegration.createDefaultIntegration() },
        //          { QuayImageIntegration.createCustomIntegration(name: "quay-duplicate") },]                 |
        //         165 | 182 | 28

        // "quay-dupe-invalid"             | "quay" |
        //         [{ QuayImageIntegration.createDefaultIntegration() },
        //          {
        //     QuayImageIntegration.createCustomIntegration(
        //                      name: "quay-duplicate",
        //                      oauthToken: Env.mustGet("QUAY_SECONDARY_BEARER_TOKEN"),
        //              )
        //          },]                               |
        //         165 | 182 | 28

        // "quay-and-other"                | "quay" |
        //         [{ GCRImageIntegration.createDefaultIntegration() },
        //          { QuayImageIntegration.createDefaultIntegration() },]                                    |
        //         165 | 182 | 28

        "gcr-keep-autogenerated"        | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },]                                     |
                41  | 182 | 28

        "gcr"                           | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },] |
                41  | 182 | 28

        cves = ["CVE-2016-2781", "CVE-2017-9614"]
    }

    private static String expectAutoGeneratedRegistry(Secret secret) {
        ImageIntegrationOuterClass.ImageIntegration autoGenerated = null
        withRetry(5, 2) {
            autoGenerated =
                    ImageIntegrationService.getImageIntegrationByName(
                            "Autogenerated ${secret.server} for cluster ${DEFAULT_CLUSTER_NAME}"
                    )
            assert autoGenerated
        }
        assert autoGenerated
        assert autoGenerated.categoriesCount == 1
        assert autoGenerated.categoriesList.contains(ImageIntegrationOuterClass.ImageIntegrationCategory.REGISTRY)
        assert autoGenerated.docker.username == secret.username
        assert autoGenerated.docker.endpoint == secret.server
        return autoGenerated.id
    }

    private static ImageOuterClass.Image expectDigestedImage(String imageName, String source) {
        def imageDigest
        withRetry(15, 2) {
            imageDigest = ImageService.getImages().find { it.name == imageName }
            assert imageDigest?.id
        }
        ImageOuterClass.Image imageDetail = ImageService.getImage(imageDigest?.id)
        assert imageDetail.metadata?.v1?.layersCount >= 1
        assert imageDetail.metadata?.layerShasCount >= 1
        assert imageDetail.metadata.dataSource.id != ""
        assert imageDetail.metadata.dataSource.name =~ source

        return imageDetail
    }
}
