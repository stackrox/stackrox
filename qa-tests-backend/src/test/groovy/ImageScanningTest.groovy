import static services.ClusterService.DEFAULT_CLUSTER_NAME
import static util.Helpers.withRetry

import io.grpc.StatusRuntimeException
import orchestratormanager.OrchestratorTypes

import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.ImageIntegrationOuterClass
import io.stackrox.proto.storage.ImageOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import io.stackrox.proto.storage.ScopeOuterClass.Scope
import io.stackrox.proto.storage.Vulnerability

import objects.AzureRegistryIntegration
import objects.ClairScannerIntegration
import objects.ClairV4ScannerIntegration
import objects.Deployment
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
import util.Timer

import org.junit.Assume
import org.junit.AssumptionViolatedException
import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Unroll
import spock.lang.IgnoreIf

@Tag("PZDebug")
@Tag("PZ")
class ImageScanningTest extends BaseSpecification {
    static final private String TEST_NAMESPACE = "qa-image-scanning-test"
    private final static String CLONED_POLICY_SUFFIX = "(${TEST_NAMESPACE})"

    static final private String UBI8_0_IMAGE = "registry.access.redhat.com/ubi8:8.0-208"
    static final private String RHEL7_IMAGE = "quay.io/rhacs-eng/qa-multi-arch:rhel7-minimal-7.5-422"
    static final private String QUAY_IMAGE_WITH_CLAIR_SCAN_DATA = "quay.io/rhacs-eng/qa:struts-app"
    static final private String GCR_IMAGE   = "us.gcr.io/stackrox-ci/qa-multi-arch/registry-image:0.2"
    static final private String NGINX_IMAGE = "quay.io/rhacs-eng/qa:nginx-1-12-1"
    static final private String OCI_IMAGE   = "quay.io/rhacs-eng/qa:oci-manifest"
    static final private String LIST_IMAGE_OCI_MANIFEST = "quay.io/rhacs-eng/qa:list-image-oci-manifest"
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

    @Shared
    private List<Policy> policiesScopedForTest

    static final private Integer WAIT_FOR_VIOLATION_TIMEOUT = isRaceBuild() ? 450 : 30

    static final private Map<String, Deployment> DEPLOYMENTS = [
            "quay": new Deployment()
                    .setName("quay-image-scanning-test")
                    .setNamespace(TEST_NAMESPACE)
                    // same image as us.gcr.io/stackrox-ci/qa/registry-image:0.3 but just retagged
                    // Alternatively can use quay.io/rhacs-eng/qa:struts-app but that doesn't have as many
                    // dockerfile violations
                    .setImage("quay.io/rhacs-eng/qa:registry-image-0-3")
                    .addLabel("app", "quay-image-scanning-test")
                    .addImagePullSecret("quay-image-scanning-test"),
            "gcr": new Deployment()
                    .setName("gcr-image-scanning-test")
                    .setNamespace(TEST_NAMESPACE)
                    .setImage("us.gcr.io/stackrox-ci/qa/registry-image:0.3")
                    .addLabel("app", "gcr-image-scanning-test")
                    .addImagePullSecret("gcr-image-scanning-test"),
            "ecr": new Deployment()
                    .setName("ecr-image-registry-test")
                    .setNamespace(TEST_NAMESPACE)
                    .setImage("${Env.mustGetAWSECRRegistryID()}.dkr.ecr.${Env.mustGetAWSECRRegistryRegion()}." +
                            "amazonaws.com/stackrox-qa-ecr-test:registry-image-no-secrets")
                    .addLabel("app", "ecr-image-registry-test")
                    .addImagePullSecret("ecr-image-registry-test"),
            "acr": new Deployment()
                    .setName("acr-image-registry-test")
                    .setNamespace(TEST_NAMESPACE)
                    .setImage("stackroxci.azurecr.io/stackroxci/registry-image:0.3")
                    .addLabel("app", "acr-image-registry-test")
                    .addImagePullSecret("acr-image-registry-test"),
    ]

    static final private Map<String, Secret> IMAGE_PULL_SECRETS = [
            "quay": new Secret(
                    name: "quay-image-scanning-test",
                    namespace: TEST_NAMESPACE,
                    username: Env.mustGet("QUAY_RHACS_ENG_RO_USERNAME"),
                    password: Env.mustGet("QUAY_RHACS_ENG_RO_PASSWORD"),
                    server: "https://quay.io"),
            "gcr": new Secret(
                    name: "gcr-image-scanning-test",
                    namespace: TEST_NAMESPACE,
                    username: "_json_key",
                    password: Env.mustGet("GOOGLE_CREDENTIALS_GCR_SCANNER"),
                    server: "https://us.gcr.io"),
            "ecr": new Secret(
                    name: "ecr-image-registry-test",
                    namespace: TEST_NAMESPACE,
                    username: "AWS",
                    password: Env.mustGetAWSECRDockerPullPassword(),
                    server: "https://${Env.mustGetAWSECRRegistryID()}.dkr.ecr."+
                            "${Env.mustGetAWSECRRegistryRegion()}.amazonaws.com"),
            "acr": new Secret(
                    name: "acr-image-registry-test",
                    namespace: TEST_NAMESPACE,
                    username: "stackroxci",
                    password: Env.mustGet("AZURE_REGISTRY_PASSWORD"),
                    server: "https://stackroxci.azurecr.io"),
    ]

    def setupSpec() {
        ImageIntegrationService.deleteStackRoxScannerIntegrationIfExists()
        removeGCRImagePullSecret()
        ImageIntegrationService.deleteAutoRegisteredGCRIntegrationIfExists()

        // Create namespace scoped policies for test.
        policiesScopedForTest = []
        for (String policyName : POLICIES) {
            Policy policy = Services.getPolicyByName(policyName)
            Policy scopedPolicyForTest = policy.toBuilder()
                .clearId()
                .setName(policy.getName() + " ${CLONED_POLICY_SUFFIX}")
                .setDisabled(false)
                .clearScope()
                .addScope(Scope.newBuilder().setNamespace(TEST_NAMESPACE))
                .build()
            Policy created = PolicyService.createAndFetchPolicy(scopedPolicyForTest)
            assert created
            policiesScopedForTest.add(created)
        }

        orchestrator.ensureNamespaceExists(TEST_NAMESPACE)
    }

    def cleanupSpec() {
        orchestrator.deleteNamespace(TEST_NAMESPACE)

        ImageIntegrationService.addStackroxScannerIntegration()
        addGCRImagePullSecret()

        for (Policy policy : policiesScopedForTest) {
            PolicyService.deletePolicy(policy.getId())
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

    @Unroll
    @Tag("BAT")
    @Tag("Integration")
    // GCR doesn't have MA images to verify the GCR-image-integrations on P/Z
    @IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
    def "Verify Image Registry+Scanner Integrations: #testName"() {
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
        for (Policy policy : policiesScopedForTest) {
            assert Services.waitForViolation(deployment.name, policy.getName(), WAIT_FOR_VIOLATION_TIMEOUT)
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
        assert ((imageDetail.cves - 20)..(imageDetail.cves + 20)).contains(totalCves)
        assert imageDetail.fixableCves >= fixable

        where:
        "Data inputs:"

        testName                        | integration |
                addIntegrationClosure                                                                     |
                components | totalCves | fixable

        // ROX-9448 - disable Quay until scanning is fixed
        // "quay-keep-autogenerated"       | "quay" |
        //         [{ QuayImageIntegration.createCustomIntegration(
        //                 [oauthToken: Env.mustGet("QUAY_RHACS_ENG_BEARER_TOKEN")]) },] |
        //         41  | 181 | 28

        // "quay"                          | "quay" |
        //         [{ QuayImageIntegration.createCustomIntegration(
        //                 [oauthToken: Env.mustGet("QUAY_RHACS_ENG_BEARER_TOKEN")]) },]                   |
        //         41  | 181 | 28

        // "quay-fully-qualified-endpoint" | "quay" |
        //         [{ QuayImageIntegration.createCustomIntegration(
        //                 oauthToken: Env.mustGet("QUAY_RHACS_ENG_BEARER_TOKEN"), endpoint: "https://quay.io/") },]  |
        //         41  | 181 | 28

        // "quay-insecure"                 | "quay" |
        //         [{ QuayImageIntegration.createCustomIntegration(
        //         oauthToken: Env.mustGet("QUAY_RHACS_ENG_BEARER_TOKEN"), insecure: true) },]            |
        //          41  | 181 | 28

        // "quay-duplicate"                | "quay" |
        //         [{ QuayImageIntegration.createCustomIntegration(
        //                 [oauthToken: Env.mustGet("QUAY_RHACS_ENG_BEARER_TOKEN")]) },
        //          { QuayImageIntegration.createCustomIntegration(
        //          oauthToken: Env.mustGet("QUAY_RHACS_ENG_BEARER_TOKEN"), name: "quay-duplicate") },]   |
        //          41  | 181 | 28

        // "quay-dupe-invalid"             | "quay" |
        //          [{ QuayImageIntegration.createCustomIntegration(
        //                 [oauthToken: Env.mustGet("QUAY_RHACS_ENG_BEARER_TOKEN")]) },
        //          {
        //      QuayImageIntegration.createCustomIntegration(
        //                      name: "quay-duplicate",
        //                     oauthToken: Env.mustGet("QUAY_SECONDARY_BEARER_TOKEN"),
        //              )
        //          },]                               |
        //         41  | 181 | 28

        // "quay-and-other"                | "quay" |
        //         [{ GCRImageIntegration.createDefaultIntegration() },
        //          { QuayImageIntegration.createCustomIntegration(
        //                   [oauthToken: Env.mustGet("QUAY_RHACS_ENG_BEARER_TOKEN")]) },]                |
        //         41  | 181 | 28

        "gcr-keep-autogenerated"        | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },]                                     |
                41  | 170 | 28

        "gcr"                           | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },] |
                41  | 170 | 28

        "gcr-fully-qualified-endpoint"  | "gcr"  |
                [{ GCRImageIntegration.createCustomIntegration(endpoint: "https://us.gcr.io/") },]        |
                41  | 170 | 28

        "gcr-duplicate"                 | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },
                 { GCRImageIntegration.createCustomIntegration(name: "gcr-duplicate") },]                 |
                41  | 170 | 28

        "gcr-dupe-invalid"              | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },
                 {
            GCRImageIntegration.createCustomIntegration(
                             name: "gcr-no-access",
                             serviceAccount: Env.mustGet("GOOGLE_CREDENTIALS_GCR_NO_ACCESS_KEY"),
                             skipTestIntegration: true,
                     ) },]                                                                                          |
                41  | 170 | 28

        "gcr-and-other"                 | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },
                 { QuayImageIntegration.createDefaultIntegration() },]                                    |
                41  | 170 | 28

        cves = ["CVE-2016-2781", "CVE-2017-9614"]
    }

    @SuppressWarnings('LineLength')
    @Unroll
    @Tag("BAT")
    @Tag("Integration")
    def "Verify Image Scan Results - #scanner.name() - #component:#version - #image - #cve - #idx"() {
        Assume.assumeTrue(scanner.isTestable())

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
        ImageOuterClass.Image img = ImageService.scanImage(image, false)
        assert img.metadata.dataSource.id != ""
        assert img.metadata.dataSource.name != ""
        assert img.scan.dataSource.id != ""
        assert img.scan.dataSource.name != ""

        then:
        ImageOuterClass.EmbeddedImageScanComponent foundComponent =
                img.scan.componentsList.find {
                    c -> c.name == component && c.version == version && c.layerIndex == idx
                }
        if (Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x") {
            // some breather for few arches
            sleep(10000)
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

        scanner                          | component      | version            | idx | cve              | image        | registry
        new StackroxScannerIntegration() | "openssl-libs"        | "1:1.0.2k-12.el7"  | 0   | "RHSA-2019:0483" | RHEL7_IMAGE  | ""
        new StackroxScannerIntegration() | "openssl-libs"        | "1:1.0.2k-12.el7"  | 0   | "CVE-2018-0735"  | RHEL7_IMAGE  | ""
        new StackroxScannerIntegration() | "systemd"             | "229-4ubuntu21.29" | 0   | "CVE-2021-33910" | OCI_IMAGE    | ""
        new StackroxScannerIntegration() | "glibc"               | "2.35-0ubuntu3.1"  | 4   | "CVE-2016-20013" | LIST_IMAGE_OCI_MANIFEST | ""
        new ClairScannerIntegration()    | "apt"                 | "1.4.8"            | 0   | "CVE-2011-3374"  | NGINX_IMAGE  | ""
        new ClairScannerIntegration()    | "bash"                | "4.4-5"            | 0   | "CVE-2019-18276" | NGINX_IMAGE  | ""
        new ClairV4ScannerIntegration()  | "openssl-libs"        | "1:1.1.1-8.el8"    | 0   | "RHSA-2021:1024" | UBI8_0_IMAGE | ""
        new ClairV4ScannerIntegration()  | "platform-python-pip" | "9.0.3-13.el8"     | 0   | "RHSA-2020:4432" | UBI8_0_IMAGE | ""
    }

    @Unroll
    @Tag("BAT")
    @Tag("Integration")
    def "Verify Scan Results from Registries - #registry.name() - #component:#version - #image - #cve - #idx"() {
        ImageIntegrationService.addStackroxScannerIntegration()

        when:
        "Add scanner"
        String integrationId = registry.createDefaultIntegration()
        assert integrationId
        integrationIds.add(integrationId)

        and:
        "Scan Image and verify results"
        ImageOuterClass.Image img = ImageService.scanImage(image, false)
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
    @Tag("Integration")
    def "Verify image scan exceptions - #scanner.name() - #testAspect"() {
        Assume.assumeTrue(scanner.isTestable())

        when:
        "Add scanner"
        String integrationId = scanner.createDefaultIntegration()
        assert integrationId
        integrationIds.add(integrationId)

        and:
        "Scan image"
        String image = IMAGES_FOR_ERROR_TESTS[scanner.name()][testAspect]
        assert image
        ImageService.scanImageNoRetry(image, false)

        then:
        "Verify image scan outcome"
        def error = thrown(expectedError)
        error.message =~ expectedMessage

        where:
        "tests are:"

        scanner                          | expectedMessage                      | testAspect
        new ClairScannerIntegration()    | /failed to get the manifest digest/  | "image does not exist"
        new StackroxScannerIntegration() | /failed to get the manifest digest/  | "image does not exist"
        new ClairScannerIntegration()    | /no matching image registries found/ | "missing required registry"
        new StackroxScannerIntegration() | /no matching image registries found/ | "missing required registry"
// This is not supported. Scanners get access to previous creds and can pull the images that way.
// https://stack-rox.atlassian.net/browse/ROX-5376
//        new StackroxScannerIntegration() | /status=401/ | "no access"

        expectedError = StatusRuntimeException
    }

    @Unroll
    @Tag("BAT")
    @Tag("Integration")
    // ACR, ECR, GCR don't have MA images to verify the the integrations on P/Z
    @IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
    def "Image metadata from registry test - #testName"() {
        Assume.assumeTrue(testName != "ecr-iam" || ClusterService.isEKS())

        if (coreImageIntegrationId != null && integration == "quay") {
            // For this test we don't want it
            // This conflicts with the autogenerated quay integration because they use the same creds
            // TODO: Switch this test to use a different image repo and token that only has access to that repo.
            //  That way the core integration and the auto-generated ones don't conflict.
            ImageIntegrationService.deleteImageIntegration(coreImageIntegrationId)
            coreImageIntegrationId = null
        }

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
        for (Policy policy : policiesScopedForTest) {
            assert Services.waitForViolation(deployment.name, policy.getName(), WAIT_FOR_VIOLATION_TIMEOUT)
        }

        cleanup:
        if (coreImageIntegrationId == null) {
            // Add it back as the rest of the test suite depends on this existing
            setupCoreImageIntegration()
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
        // TODO: append " from .*" once SourcedAutogeneratedIntegrations is enabled.
        return "Autogenerated ${server} for cluster ${DEFAULT_CLUSTER_NAME}"
    }

    @Unroll
    @Tag("Integration")
    def "Image scanning test to check if scan time is not null #image from stackrox"() {
        when:
        "Add Stackrox scanner"
        String integrationId = StackroxScannerIntegration.createDefaultIntegration()
        assert integrationId
        integrationIds.add(integrationId)

        and:
        "Image is scanned"
        ImageService.scanImage(image, false)

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
        "registry.k8s.io/ip-masq-agent-amd64:v2.4.1"       | "gcr registry"
        "quay.io/rhacs-eng/qa:alpine-3.16.0"               | "quay registry"
        "quay.io/stackrox-io/scanner:2.27.3"               | "quay registry"
        "gke.gcr.io/heapster:v1.7.2"                       | "one from gke"
        "mcr.microsoft.com/dotnet/core/runtime:2.1-alpine" | "one from mcr"
    }

    def "Validate basic image details across all current images in StackRox"() {
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

    @Unroll
    @Tag("BAT")
    @Tag("Integration")
    def "Quay registry and scanner supports token and/or robot credentials - #testName"() {
        if (coreImageIntegrationId != null) {
            // For this test we don't want it
            // This conflicts with the autogenerated quay integration because they use the same creds
            // TODO: Switch this test to use a different image repo and token that only has access to that repo.
            //  That way the core integration and the auto-generated ones don't conflict.
            ImageIntegrationService.deleteImageIntegration(coreImageIntegrationId)
            coreImageIntegrationId = null
        }

        when:
        "Image registry and scanner integrations are configured"
        if (scannerName == "Stackrox Scanner") {
            ImageIntegrationService.addStackroxScannerIntegration()
            deleteStackroxScanner = true
        }

        String integrationId
        integrationId = imageIntegrationConfig()
        integrationIds.add(integrationId)

        then:
        "Validate registry based image metadata"
        def imageDetail = expectedDigestImageFromScan(QUAY_IMAGE_WITH_CLAIR_SCAN_DATA, integrationName)

        and:
        "Validate image scan details"
        assert imageDetail.scan.dataSource.id != ""
        assert imageDetail.scan.dataSource.name == scannerName

        try {
            assert imageDetail.scan.componentsCount > 0
            assert imageDetail.scan.componentsList.size() > 0
            assert imageDetail.scan.componentsList.vulnsCount.sum { it as Integer } > 0
        } catch (Exception e) {
            if (strictIntegrationTesting) {
                throw (e)
            }
            throw new AssumptionViolatedException("Failed to scan the image using ${scannerName}. Skipping test!", e)
        }

        cleanup:
        if (scannerName == "Stackrox Scanner") {
            deleteStackroxScanner = true
        }

        if (coreImageIntegrationId == null) {
            // Add it back as the rest of the test suite depends on this existing
            setupCoreImageIntegration()
        }

        where:
        testName                                           | integrationName  | scannerName |
                imageIntegrationConfig
        "quay registry with token"                         | "quay"           | "Stackrox Scanner" |
                { -> QuayImageIntegration.createCustomIntegration(
                        [oauthToken: Env.mustGet("QUAY_RHACS_ENG_BEARER_TOKEN"), includeScanner: false,]) }
        "quay with robot creds only"                      | "quay"    |  "Stackrox Scanner" |
                { -> QuayImageIntegration.createCustomIntegration(
                        [oauthToken: "", useRobotCreds: true, includeScanner: false,]) }

        "quay registry+scanner with token"                  | "quay"   | "quay" |
                { -> QuayImageIntegration.createCustomIntegration(
                        [oauthToken: Env.mustGet("QUAY_RHACS_ENG_BEARER_TOKEN"), includeScanner: true,]) }

        "quay registry+scanner with token and robot creds"  | "quay"   | "quay" |
                { -> QuayImageIntegration.createCustomIntegration(
                        [oauthToken: Env.mustGet("QUAY_RHACS_ENG_BEARER_TOKEN"), useRobotCreds: true,
                         includeScanner: true,]) }
    }

    @SuppressWarnings('LineLength')
    private static String expectAutoGeneratedRegistry(Secret secret) {
        ImageIntegrationOuterClass.ImageIntegration autoGenerated = null
        withRetry(5, 2) {
            // TODO: append " from ${secret.namespace}/${secret.name}" once SourcedAutogeneratedIntegrations is enabled.
            autoGenerated = ImageIntegrationService.getImageIntegrationByName(
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
            def images = ImageService.getImages()
            imageDigest = images.find { it.name == imageName }
            assert imageDigest?.id, "image ${imageName} not found among ${images*.name}"
        }
        ImageOuterClass.Image imageDetail
        withRetry(10, 20) {
            imageDetail = ImageService.getImage(imageDigest?.id)
            validateImageMetadata(imageDetail, source)
        }
        return imageDetail
    }

    private static ImageOuterClass.Image expectedDigestImageFromScan(String imageName, String source) {
        ImageOuterClass.Image imageDetail = null
        withRetry(30, 2) {
            imageDetail = ImageService.scanImage(imageName, false, true)
        }
        validateImageMetadata(imageDetail, source)
        return imageDetail
    }

    private static ImageOuterClass.Image validateImageMetadata(ImageOuterClass.Image imageDetail, String source) {
        assert imageDetail.metadata?.v1?.layersCount >= 1
        assert imageDetail.metadata?.layerShasCount >= 1
        assert imageDetail.metadata.dataSource.id != ""
        assert imageDetail.metadata.dataSource.name =~ source
    }
}
