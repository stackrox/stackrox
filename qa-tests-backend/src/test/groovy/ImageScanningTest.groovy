import static services.ClusterService.DEFAULT_CLUSTER_NAME
import common.Constants
import groups.BAT
import groups.Integration
import io.grpc.StatusRuntimeException
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.ImageIntegrationOuterClass
import io.stackrox.proto.storage.ImageOuterClass
import io.stackrox.proto.storage.Vulnerability
import objects.AzureRegistryIntegration
import objects.ClairScannerIntegration
import objects.Deployment
import objects.ECRRegistryIntegration
import objects.GCRImageIntegration
import objects.GoogleArtifactRegistry
import objects.QuayImageIntegration
import objects.Secret
import objects.StackroxScannerIntegration
import orchestratormanager.OrchestratorTypes
import org.junit.Assume
import org.junit.AssumptionViolatedException
import org.junit.experimental.categories.Category
import services.ClusterService
import services.ImageIntegrationService
import services.ImageService
import spock.lang.Shared
import spock.lang.Unroll
import util.Env
import util.Helpers
import util.Timer

class ImageScanningTest extends BaseSpecification {

    static final private String RHEL7_IMAGE =
            "richxsl/rhel7@sha256:8f3aae325d2074d2dc328cb532d6e7aeb0c588e15ddf847347038fe0566364d6"
    static final private String GCR_IMAGE   = "us.gcr.io/stackrox-ci/qa/registry-image:0.2"
    static final private String NGINX_IMAGE = "quay.io/rhacs-eng/qa:nginx-1-12-1"
    static final private String OCI_IMAGE   = "quay.io/rhacs-eng/qa:oci-manifest"
    static final private String AR_IMAGE    = "us-west1-docker.pkg.dev/stackrox-ci/artifact-registry-test1/nginx:1.17"
    static final private String CENTOS_IMAGE = "quay.io/rhacs-eng/qa:centos7-base"
    static final private String CENTOS_ECHO_IMAGE = "quay.io/rhacs-eng/qa:centos7-base-echo"

    // Amount of seconds to sleep to avoid race condition during on-going processing of images.
    static final private int SLEEP_DURING_PROCESSING = isRaceBuild() ? 25000 :
            ((Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT) ? 20000 : 15000)

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
        log.info "Post test cleanup:"
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
                // Sleep for 20s in order to avoid race condition with processing that is currently in progress
                log.info "Sleeping to avoid race condition with reprocessing"
                sleep(SLEEP_DURING_PROCESSING)
                ImageService.deleteImagesWithRetry(SearchServiceOuterClass.RawQuery.newBuilder()
                        .setQuery("Image:${imageToCleanup}").build(), true)
            } catch (e) {
                log.info "Image delete threw an exception: ${e}, this is OK for some retry cases."
            }
        }
        integrationIds.each { ImageIntegrationService.deleteImageIntegration(it) }

        if (deleteStackroxScanner) {
            ImageIntegrationService.deleteStackRoxScannerIntegrationIfExists()
        }
    }

    def cleanupSetupForRetry() {
        if (Helpers.getAttemptCount() > 1) {
            cleanup()
            setup()
        }
    }

    @Unroll
    @Category([BAT, Integration])
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
            withRetry(30, 2) {
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
        ImageService.scanImage(deployment.image)
        imageDetail = ImageService.getImage(ImageService.getImages().find { it.name == deployment.image }?.id)

        then:
        "validate scan results for the image"
        Timer t = new Timer(20, 3)
        while (imageDetail?.scan?.componentsCount == 0 && t.IsValid()) {
            log.info "waiting on scan details..."
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
            log.info "Validating existence of ${cve} cve..."
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
                41  | 181 | 28

        "gcr"                           | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },] |
                41  | 181 | 28

        "gcr-fully-qualified-endpoint"  | "gcr"  |
                [{ GCRImageIntegration.createCustomIntegration(endpoint: "https://us.gcr.io/") },]        |
                41  | 181 | 28

        "gcr-duplicate"                 | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },
                 { GCRImageIntegration.createCustomIntegration(name: "gcr-duplicate") },]                 |
                41  | 181 | 28

        "gcr-dupe-invalid"              | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },
                 {
            GCRImageIntegration.createCustomIntegration(
                             name: "gcr-no-access",
                             serviceAccount: Env.mustGet("GOOGLE_CREDENTIALS_GCR_NO_ACCESS_KEY"),
                             skipTestIntegration: true,
                     ) },]                                                                                          |
                41  | 181 | 28

        "gcr-and-other"                 | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },
                 { QuayImageIntegration.createDefaultIntegration() },]                                    |
                41  | 181 | 28

        cves = ["CVE-2016-2781", "CVE-2017-9614"]
    }

    @SuppressWarnings('LineLength')
    @Unroll
    @Category([BAT, Integration])
    def "Verify Image Scan Results - #scanner.name() - #component:#version - #image - #cve - #idx"() {
        Assume.assumeTrue(scanner.isTestable())
        cleanupSetupForRetry()

        when:
        "A registry is required"
        if (registry) {
            assert IMAGE_PULL_SECRETS.containsKey(registry)
            secret = IMAGE_PULL_SECRETS.get(registry)
            orchestrator.createImagePullSecret(secret)
            sleep 2000
            String autoCreatedIntegrationId = expectAutoGeneratedRegistry(secret)
            integrationIds.add(autoCreatedIntegrationId)
        }

        and:
        "Add scanner"
        String integrationId = scanner.createDefaultIntegration()
        assert integrationId
        integrationIds.add(integrationId)

        and:
        "Scan Image and verify results"
        ImageOuterClass.Image img = Services.scanImage(image)
        assert img.metadata.dataSource.id != ""
        assert img.metadata.dataSource.name != ""
        assert img.scan.dataSource.id != ""
        assert img.scan.dataSource.name != ""

        then:
        ImageOuterClass.EmbeddedImageScanComponent foundComponent =
                img.scan.componentsList.find {
                    c -> c.name == component && c.version == version && c.layerIndex == idx
                }
        foundComponent != null

        Vulnerability.EmbeddedVulnerability vuln =
                foundComponent.vulnsList.find { v -> v.cve == cve }

        vuln != null

        cleanup:
        if (scanner.isTestable()) {
            imageToCleanup = image
        }

        where:
        "Data inputs are: "

        scanner                          | component      | version            | idx | cve              | image       | registry
        new StackroxScannerIntegration() | "openssl-libs" | "1:1.0.1e-34.el7"  | 1   | "RHSA-2014:1052" | RHEL7_IMAGE | ""
        new StackroxScannerIntegration() | "openssl-libs" | "1:1.0.1e-34.el7"  | 1   | "CVE-2014-3509"  | RHEL7_IMAGE | ""
        new StackroxScannerIntegration() | "systemd"      | "229-4ubuntu21.29" | 0   | "CVE-2021-33910" | OCI_IMAGE   | ""
        new ClairScannerIntegration()    | "apt"          | "1.4.8"            | 0   | "CVE-2011-3374"  | NGINX_IMAGE | ""
        new ClairScannerIntegration()    | "bash"         | "4.4-5"            | 0   | "CVE-2019-18276" | NGINX_IMAGE | ""
    }

    @Unroll
    @Category([BAT, Integration])
    def "Verify Scan Results from Registries - #registry.name() - #component:#version - #image - #cve - #idx"() {
        cleanupSetupForRetry()
        ImageIntegrationService.addStackroxScannerIntegration()

        when:
        "Add scanner"
        String integrationId = registry.createDefaultIntegration()
        assert integrationId
        integrationIds.add(integrationId)

        and:
        "Scan Image and verify results"
        ImageOuterClass.Image img = Services.scanImage(image)
        assert img.metadata.dataSource.id != ""
        assert img.metadata.dataSource.name != ""
        assert img.scan.dataSource.id != ""
        assert img.scan.dataSource.name != ""

        then:
        ImageOuterClass.EmbeddedImageScanComponent foundComponent =
                img.scan.componentsList.find {
                    c -> c.name == component && c.version == version && c.layerIndex == idx
                }
        foundComponent != null

        Vulnerability.EmbeddedVulnerability vuln =
                foundComponent.vulnsList.find { v -> v.cve == cve }

        vuln != null

        cleanup:
        deleteStackroxScanner = true
        imageToCleanup = image

        where:
        "Data inputs are: "

        registry                     | component | version   | idx | cve              | image
        new GoogleArtifactRegistry() | "gcc-8"   | "8.3.0-6" | 0   | "CVE-2018-12886" | AR_IMAGE
    }

    static final private IMAGES_FOR_ERROR_TESTS = [
            "Clair Scanner"   : [
                    "image does not exist"     : "non-existent:image",
                    "missing required registry": GCR_IMAGE,
            ],
            "Stackrox Scanner": [
                    "image does not exist"     : "non-existent:image",
                    "no access"                : "quay.io/stackrox/testing:registry-image",
                    "missing required registry": GCR_IMAGE,
            ],
    ]

    @Unroll
    @Category(Integration)
    def "Verify image scan exceptions - #scanner.name() - #testAspect"() {
        Assume.assumeTrue(scanner.isTestable())
        cleanupSetupForRetry()

        when:
        "Add scanner"
        String integrationId = scanner.createDefaultIntegration()
        assert integrationId
        integrationIds.add(integrationId)

        and:
        "Scan image"
        String image = IMAGES_FOR_ERROR_TESTS[scanner.name()][testAspect]
        assert image
        Services.scanImage(image)

        then:
        "Verify image scan outcome"
        def error = thrown(expectedError)
        error.message =~ expectedMessage

        where:
        "tests are:"

        scanner                          | expectedMessage                      | testAspect
        new ClairScannerIntegration()    | /Failed to get the manifest digest/  | "image does not exist"
        new StackroxScannerIntegration() | /Failed to get the manifest digest/  | "image does not exist"
        new ClairScannerIntegration()    | /no matching image registries found/ | "missing required registry"
        new StackroxScannerIntegration() | /no matching image registries found/ | "missing required registry"
// This is not supported. Scanners get access to previous creds and can pull the images that way.
// https://stack-rox.atlassian.net/browse/ROX-5376
//        new StackroxScannerIntegration() | /status=401/ | "no access"

        expectedError = StatusRuntimeException
    }

    @Unroll
    @Category([BAT, Integration])
    def "Image metadata from registry test - #testName"() {
        Assume.assumeTrue(testName != "ecr-iam" || ClusterService.isEKS())
        cleanupSetupForRetry()

        secret = IMAGE_PULL_SECRETS.get(integration)
        deployment = DEPLOYMENTS.get(integration)
        deployment = deployment.clone()
        deployment.setName("${testName}--${deployment.name}")
        if (testName == "ecr-iam") {
            secret = null
            deployment.setImagePullSecret([])
        }

        when:
        "Image integration is configured"
        String integrationId
        if (imageIntegrationConfig) {
            integrationId = imageIntegrationConfig()
            integrationIds.add(integrationId)
        }

        // and/or:
        "A pull secret auto creates an integration"
        if (secret) {
            orchestrator.createImagePullSecret(secret)
            sleep(SLEEP_DURING_PROCESSING)
            String autoCreatedIntegrationId = expectAutoGeneratedRegistry(secret)
            if (deleteAutoRegistry) {
                ImageIntegrationService.deleteImageIntegration(autoCreatedIntegrationId)
            } else {
                integrationIds.add(autoCreatedIntegrationId)
            }
        }

        and:
        "A deployment from this registry is started"
        orchestrator.createDeployment(deployment)
        assert Services.waitForDeployment(deployment)

        then:
        "validate registry based image metadata"
        expectDigestedImage(deployment.image, source)

        and:
        "validate expected violations based on dockerfile"
        for (String policy : POLICIES) {
            assert Services.waitForViolation(deployment.name, policy, WAIT_FOR_VIOLATION_TIMEOUT)
        }

        where:
        testName                      | integration | deleteAutoRegistry | source                     |
                imageIntegrationConfig
        "ecr-iam"                     | "ecr"       | false              | /^ecr$/                    |
                { -> ECRRegistryIntegration.createCustomIntegration(useIam: true, endpoint: "") }
        "ecr-assume-role"             | "ecr"       | false              | /^ecr$/                    |
                { -> ECRRegistryIntegration.createCustomIntegration(useAssumeRole: true, endpoint: "") }
        "ecr-assume-role-external-id" | "ecr"       | false              | /^ecr$/                    |
                { -> ECRRegistryIntegration.createCustomIntegration(useAssumeRoleExternalId: true, endpoint: "") }
        "ecr-auto"                    | "ecr"       | false              | source(".*.amazonaws.com") |
                null
        "ecr-auto-and-config"         | "ecr"       | false              | /^ecr$/                    |
                { -> ECRRegistryIntegration.createDefaultIntegration() }
        "ecr-config-only"             | "ecr"       | true               | /^ecr$/                    |
                { -> ECRRegistryIntegration.createDefaultIntegration() }
        "acr-auto"            | "acr"       | false              | /Autogenerated .*.azurecr.io for cluster remote/ |
                null
        "acr-auto-and-config" | "acr"       | false              | /^acr$/                    |
                { -> AzureRegistryIntegration.createDefaultIntegration() }
        "acr-config-only"     | "acr"       | true               | /^acr$/                    |
                { -> AzureRegistryIntegration.createDefaultIntegration() }
        "quay-auto"           | "quay"      | false              | source(".*.quay.io")       | null
        "gcr-auto"            | "gcr"       | false              | source(".*.gcr.io")        | null
    }

    private static String source(String server) {
        return "Autogenerated ${server} for cluster ${DEFAULT_CLUSTER_NAME}"
    }

    @Unroll
    @Category(Integration)
    def "Image scanning test to check if scan time is not null #image from stackrox"() {
        cleanupSetupForRetry()

        when:
        "Add Stackrox scanner"
        String integrationId = StackroxScannerIntegration.createDefaultIntegration()
        assert integrationId
        integrationIds.add(integrationId)

        and:
        "Image is scanned"
        Services.scanImage(image)

        then:
        "get image by name"
        String id = Services.getImageIdByName(image)
        ImageOuterClass.Image img = Services.getImageById(id)

        and:
        "check scanned time is not null"
        assert img.scan.scanTime != null
        assert img.scan.hasScanTime() == true

        cleanup:
        imageToCleanup = image

        where:
        image                                              | registry
        "k8s.gcr.io/ip-masq-agent-amd64:v2.4.1"            | "gcr registry"
        "docker.io/jenkins/jenkins:lts"                    | "docker registry"
        "docker.io/jenkins/jenkins:2.220-alpine"           | "docker registry"
        "gke.gcr.io/heapster:v1.7.2"                       | "one from gke"
        "mcr.microsoft.com/dotnet/core/runtime:2.1-alpine" | "one from mcr"
    }

    def "Validate basic image details across all current images in StackRox"() {
        cleanupSetupForRetry()

        when:
        "get list of all images"
        List<ImageOuterClass.ListImage> images = ImageService.getImages()

        then:
        "validate details for each image"
        Map<ImageOuterClass.ImageName, List<Vulnerability.EmbeddedVulnerability>> missingValues = [:]
        for (ImageOuterClass.ListImage image : images) {
            ImageOuterClass.Image imageDetails = ImageService.getImage(image.id)

            if (imageDetails.hasScan()) {
                assert imageDetails.scan.scanTime
                for (ImageOuterClass.EmbeddedImageScanComponent component : imageDetails.scan.componentsList) {
                    for (Vulnerability.EmbeddedVulnerability vuln : component.vulnsList) {
                        // Removed summary due to GCR's lack of summary
                        if (0.0 > vuln.cvss || vuln.cvss > 10.0 ||
                                vuln.link == null || vuln.link == "") {
                            missingValues.containsKey(imageDetails.name) ?
                                    missingValues.get(imageDetails.name).add(vuln) :
                                    missingValues.put(imageDetails.name, [vuln])
                        }
                    }
                }
            }
            if (missingValues.containsKey(imageDetails.name)) {
                log.info "Failing image: ${imageDetails}"
            }
        }
        log.info missingValues.toString()
        assert missingValues.size() == 0
    }

    def "Validate image deletion does not affect other images"() {
        cleanupSetupForRetry()

        given:
        ImageIntegrationService.addStackroxScannerIntegration()

        when:
        "Scan CentOS image and derivative echo image (centos + touch file)"
        ImageService.scanImage(CENTOS_ECHO_IMAGE, false)
        def expectedDetails = ImageService.scanImage(CENTOS_IMAGE, false)

        and:
        "Delete CentOS image and ensure echo still same number of vulns"
        ImageService.deleteImages(
                SearchServiceOuterClass.RawQuery.newBuilder().setQuery("Image:${CENTOS_ECHO_IMAGE}").build(), true)
        def actualDetails = ImageService.getImage(expectedDetails.id)
        assert actualDetails.scan.componentsList.sum { it.vulnsList.size() } > 0

        then:
        "Delete CentOS image and ensure echo still same number of vulns"
        expectedDetails.scan.componentsList.size() == actualDetails.scan.componentsList.size()
        expectedDetails.scan.componentsList.sum { it.vulnsList.size() } ==
                actualDetails.scan.componentsList.sum { it.vulnsList.size() }

        cleanup:
        deleteStackroxScanner = true
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
        withRetry(30, 2) {
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
