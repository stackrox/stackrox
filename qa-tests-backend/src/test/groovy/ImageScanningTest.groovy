import common.Constants
import groups.BAT
import groups.Integration
import io.grpc.StatusRuntimeException
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.ImageIntegrationOuterClass
import io.stackrox.proto.storage.ImageOuterClass
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
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ClusterService
import services.ImageIntegrationService
import services.ImageService
import services.PolicyService
import spock.lang.Shared
import spock.lang.Unroll
import util.Env
import util.Timer

class ImageScanningTest extends BaseSpecification {

    static final private String RHEL7_IMAGE =
            "richxsl/rhel7@sha256:8f3aae325d2074d2dc328cb532d6e7aeb0c588e15ddf847347038fe0566364d6"
    static final private String GCR_IMAGE   = "us.gcr.io/stackrox-ci/qa/registry-image:0.2"
    static final private String NGINX_IMAGE = "nginx:1.12.1"
    static final private String AR_IMAGE    = "us-west1-docker.pkg.dev/stackrox-ci/artifact-registry-test1/nginx:1.17"

    static final private List<String> POLICIES = [
            "ADD Command used instead of COPY",
            "Secure Shell (ssh) Port Exposed in Image",
    ]

    static final private Map<String, Deployment> DEPLOYMENTS = [
            "quay": new Deployment()
                    .setName("quay-image-scanning-test")
                    .setImage("quay.io/stackrox/testing:registry-image")
                    .addLabel("app", "quay-image-scanning-test")
                    .addImagePullSecret("quay-image-scanning-test"),
            "gcr": new Deployment()
                    .setName("gcr-image-scanning-test")
                    .setImage("us.gcr.io/stackrox-ci/qa/registry-image:0.2")
                    .addLabel("app", "gcr-image-scanning-test")
                    .addImagePullSecret("gcr-image-scanning-test"),
            "ecr": new Deployment()
                    .setName("ecr-image-registry-test")
                    .setImage("${Env.mustGetAWSECRRegistryID()}.dkr.ecr.${Env.mustGetAWSECRRegistryRegion()}." +
                            "amazonaws.com/stackrox-qa-ecr-test:registry-image")
                    .addLabel("app", "ecr-image-registry-test")
                    .addImagePullSecret("ecr-image-registry-test"),
            "acr": new Deployment()
                    .setName("acr-image-registry-test")
                    .setImage("stackroxci.azurecr.io/stackroxci/registry-image:0.2")
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
                    server: "https://stackroxci.azurecr.io")
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
        ImageIntegrationService.handleUnreliableGCRAutoGenerate()

        for (String policy : UPDATED_POLICIES) {
            Services.setPolicyDisabled(policy, true)
        }
    }

    @Unroll
    @Category([BAT, Integration])
    def "Verify Image Registry+Scanner Integrations: #testName"() {
        given:
        "Get deployment details used to test integration"
        Deployment deployment = null
        if (IMAGE_PULL_SECRETS.containsKey(integration)) {
            orchestrator.createImagePullSecret(IMAGE_PULL_SECRETS.get(integration))
        }
        if (DEPLOYMENTS.containsKey(integration)) {
            deployment = DEPLOYMENTS.get(integration)
            deployment = deployment.clone()
            deployment.setName("${testName}--${deployment.name}")
            orchestrator.createDeployment(deployment)
            assert Services.waitForDeployment(deployment)
        }
        assert deployment

        expect:
        "validate auto-generated registry was created"
        def autogeneratedId
        if (IMAGE_PULL_SECRETS.containsKey(integration)) {
            autogeneratedId = expectAutoGeneratedRegistry(IMAGE_PULL_SECRETS.get(integration))
        }

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
            Assume.assumeNoException("Failed to pull the image using ${integration}. Skipping test!", e)
        }
        ImageOuterClass.Image imageDetail = ImageService.getImage(imageDigest?.id)
        assert imageDetail.metadata?.v1?.layersCount >= 1
        assert imageDetail.metadata?.layerShasCount >= 1

        and:
        "validate expected violations based on dockerfile"
        for (String policy : POLICIES) {
            assert Services.waitForViolation(deployment.name, policy)
        }

        when:
        "Add scanner integration"
        def integrationIds = []
        addIntegrationClosure.each {
            def id = it()
            integrationIds.add(id) }
        PolicyService.reassessPolicies()
        ImageService.scanImage(deployment.image)
        imageDetail = ImageService.getImage(ImageService.getImages().find { it.name == deployment.image }?.id)

        then:
        "validate scan results for the image"
        Timer t = new Timer(20, 3)
        while (imageDetail.scan?.componentsCount == 0 && t.IsValid()) {
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
            Assume.assumeNoException("Failed to scan the image using ${integration}. Skipping test!", e)
        }

        and:
        "validate the existence of expected CVEs"
        for (String cve : cves) {
            println "Validating existence of ${cve} cve..."
            ImageOuterClass.EmbeddedImageScanComponent component = imageDetail.scan.componentsList.find {
                component -> component.vulnsList.find { vuln -> vuln.cve == cve }
            }
            assert component
            ImageOuterClass.EmbeddedVulnerability vuln = component.vulnsList.find { it.cve == cve }
            assert vuln

            assert vuln.summary && vuln.summary != ""
            assert 0.0 <= vuln.cvss && vuln.cvss <= 10.0
            assert vuln.link && vuln.link != ""
        }
        assert imageDetail.components >= components
        assert imageDetail.cves >= totalCves
        assert imageDetail.fixableCves >= fixable

        cleanup:
        "Remove deployment and integrations"
        if (IMAGE_PULL_SECRETS.containsKey(integration)) {
            def s = IMAGE_PULL_SECRETS.get(integration)
            orchestrator.deleteSecret(s.name, s.namespace)
        }
        if (DEPLOYMENTS.containsKey(integration)) {
            orchestrator.deleteAndWaitForDeploymentDeletion(deployment)
        }
        integrationIds.each { ImageIntegrationService.deleteImageIntegration(it) }
        if (deleteAutoGenerated && autogeneratedId != null) {
            ImageIntegrationService.deleteImageIntegration(autogeneratedId)
        }
        ImageService.clearImageCaches()
        ImageService.deleteImagesWithRetry(SearchServiceOuterClass.RawQuery.newBuilder()
                .setQuery("Image:${deployment.image}").build(), true)

        where:
        "Data inputs:"

        testName                        | integration |
                addIntegrationClosure                                                                             |
                components | totalCves | fixable

        "quay-keep-autogenerated"       | "quay" |
                [{ QuayImageIntegration.createDefaultIntegration() },] |
                165 | 184 | 28

        "quay"                          | "quay" |
                [{ QuayImageIntegration.createDefaultIntegration() },]                                      |
                165 | 184 | 28

        "quay-fully-qualified-endpoint" | "quay" |
                [{ QuayImageIntegration.createCustomIntegration(endpoint: "https://quay.io/") },]           |
                165 | 184 | 28

        "quay-insecure"                 | "quay" |
                [{ QuayImageIntegration.createCustomIntegration(insecure: true) },]                         |
                165 | 184 | 28

        "quay-duplicate"                | "quay" |
                [{ QuayImageIntegration.createDefaultIntegration() },
                 { QuayImageIntegration.createCustomIntegration(name: "quay-duplicate") },]                 |
                165 | 184 | 28

        "quay-dupe-invalid"             | "quay" |
                [{ QuayImageIntegration.createDefaultIntegration() },
                 {
            QuayImageIntegration.createCustomIntegration(
                             name: "quay-duplicate",
                             oauthToken: Env.mustGet("QUAY_SECONDARY_BEARER_TOKEN"),
                     )
                 },]                               |
                165 | 184 | 28

        "quay-and-other"                | "quay" |
                [{ GCRImageIntegration.createDefaultIntegration() },
                 { QuayImageIntegration.createDefaultIntegration() },]                                    |
                165 | 184 | 28

        "gcr-keep-autogenerated"        | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },]                                     |
                44  | 204 | 49

        "gcr"                           | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },] |
                44  | 204 | 49

        "gcr-fully-qualified-endpoint"  | "gcr"  |
                [{ GCRImageIntegration.createCustomIntegration(endpoint: "https://us.gcr.io/") },]        |
                44  | 204 | 49

        "gcr-duplicate"                 | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },
                 { GCRImageIntegration.createCustomIntegration(name: "gcr-duplicate") },]                 |
                44  | 204 | 49

        "gcr-dupe-invalid"              | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },
                 {
            GCRImageIntegration.createCustomIntegration(
                             name: "gcr-no-access",
                             serviceAccount: Env.mustGet("GOOGLE_CREDENTIALS_GCR_NO_ACCESS_KEY"),
                             skipTestIntegration: true,
                     ) },]                                                                                          |
                44  | 204 | 49

        "gcr-and-other"                 | "gcr"  |
                [{ GCRImageIntegration.createDefaultIntegration() },
                 { QuayImageIntegration.createDefaultIntegration() },]                                    |
                44  | 204 | 49

        deleteAutoGenerated = !testName.contains("keep-autogenerated")
        cves = ["CVE-2016-2781", "CVE-2017-9614"]
    }

    @Unroll
    @Category([BAT, Integration])
    def "Verify Image Scan Results - #scanner.name() - #component:#version - #image - #cve - #idx"() {
        Assume.assumeTrue(scanner.isTestable())

        when:
        "Add scanner"
        def integrationId = scanner.createDefaultIntegration()
        assert integrationId

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

        ImageOuterClass.EmbeddedVulnerability vuln =
                foundComponent.vulnsList.find { v -> v.cve == cve }

        vuln != null

        cleanup:
        "Remove scanner and delete image"
        integrationId ? ImageIntegrationService.deleteImageIntegration(integrationId) : null
        ImageService.clearImageCaches()
        ImageService.deleteImagesWithRetry(SearchServiceOuterClass.RawQuery.newBuilder()
                .setQuery("Image:${image}").build(), true)

        where:
        "Data inputs are: "

        scanner                          | component      | version            | idx | cve              | image
        new StackroxScannerIntegration() | "openssl-libs" | "1:1.0.1e-34.el7"  | 1   | "RHSA-2014:1052" | RHEL7_IMAGE
        new StackroxScannerIntegration() | "openssl-libs" | "1:1.0.1e-34.el7"  | 1   | "CVE-2014-3509"  | RHEL7_IMAGE
        new AnchoreScannerIntegration()  | "openssl"      | "1.0.1t-1+deb8u12" | 0   | "CVE-2010-0928"  | GCR_IMAGE
        new AnchoreScannerIntegration()  | "perl"         | "5.20.2-3+deb8u12" | 0   | "CVE-2011-4116"  | GCR_IMAGE
        new ClairScannerIntegration()    | "apt"          | "1.4.8"            | 0   | "CVE-2011-3374"  | NGINX_IMAGE
        new ClairScannerIntegration()    | "bash"         | "4.4-5"            | 0   | "CVE-2019-18276" | NGINX_IMAGE
    }

    @Unroll
    @Category([BAT, Integration])
    def "Verify Scan Results from Registries - #registry.name() - #component:#version - #image - #cve - #idx"() {
        ImageIntegrationService.addStackroxScannerIntegration()

        when:
        "Add scanner"
        def integrationId = registry.createDefaultIntegration()
        assert integrationId

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

        ImageOuterClass.EmbeddedVulnerability vuln =
                foundComponent.vulnsList.find { v -> v.cve == cve }

        vuln != null

        cleanup:
        "Remove scanner and delete image"
        integrationId ? ImageIntegrationService.deleteImageIntegration(integrationId) : null
        ImageIntegrationService.deleteStackRoxScannerIntegrationIfExists()
        ImageService.clearImageCaches()
        ImageService.deleteImagesWithRetry(SearchServiceOuterClass.RawQuery.newBuilder()
                .setQuery("Image:${image}").build(), true)

        where:
        "Data inputs are: "

        registry                          | component      | version            | idx | cve              | image
        new GoogleArtifactRegistry()     | "gcc-8"        | "8.3.0-6"          | 0   | "CVE-2018-12886" | AR_IMAGE
    }

    static final private IMAGES_FOR_ERROR_TESTS = [
            "Anchore Scanner": [
                    "image does not exist": "non-existent:image",
                    "no access": "quay.io/stackrox/testing:registry-image"
            ],
            "Clair Scanner": [
                    "image does not exist": "non-existent:image",
            ],
            "Stackrox Scanner": [
                    "image does not exist": "non-existent:image",
                    "no access": "quay.io/stackrox/testing:registry-image"
            ],
    ]

    @Unroll
    @Category(Integration)
    def "Verify image scan exceptions - #scanner.name() - #testAspect"() {
        Assume.assumeTrue(scanner.isTestable())

        when:
        "Add scanner"
        def integrationId = scanner.createDefaultIntegration()
        assert integrationId

        and:
        "Scan image"
        String image = IMAGES_FOR_ERROR_TESTS[scanner.name()][testAspect]
        assert image
        Services.scanImage(image)

        then:
        "Verify image scan outcome"
        def error = thrown(expectedError)
        error.message =~ expectedMessage

        cleanup:
        "Remove scanner and delete image"
        integrationId ? ImageIntegrationService.deleteImageIntegration(integrationId) : null
        ImageService.clearImageCaches()
        ImageService.deleteImages(SearchServiceOuterClass.RawQuery.newBuilder()
                .setQuery("Image:${image}").build(), true)

        where:
        "tests are:"

        scanner                          | expectedMessage                     | testAspect
        new AnchoreScannerIntegration()  | /Failed to get the manifest digest/ | "image does not exist"
        new ClairScannerIntegration()    | /Failed to get the manifest digest/ | "image does not exist"
        new StackroxScannerIntegration() | /Failed to get the manifest digest/ | "image does not exist"
// This is not supported. Scanners get access to previous creds and can pull the images that way.
// https://stack-rox.atlassian.net/browse/ROX-5376
//        new AnchoreScannerIntegration() | /access to the requested resource is not authorized/ | "no access"
//        new StackroxScannerIntegration() | /status=401/ | "no access"

        expectedError = StatusRuntimeException
    }

    @Unroll
    @Category([BAT, Integration])
    def "Image metadata from registry test - #testName"() {
        Assume.assumeTrue(testName != "ecr-iam" || ClusterService.isEKS())

        Secret secret = IMAGE_PULL_SECRETS.get(integration)
        Deployment deployment = DEPLOYMENTS.get(integration)
        deployment = deployment.clone()
        deployment.setName("${testName}--${deployment.name}")
        if (testName == "ecr-iam") {
            secret = null
            deployment.setImagePullSecret([])
        }

        when:
        "Image integration is configured"
        def integrationId
        if (imageIntegrationConfig) {
            integrationId = imageIntegrationConfig()
        }

        // and/or:
        "A pull secret auto creates an integration"
        String autoCreatedIntegrationId
        if (secret) {
            orchestrator.createImagePullSecret(secret)
            autoCreatedIntegrationId = expectAutoGeneratedRegistry(secret)
            if (deleteAutoRegistry) {
                ImageIntegrationService.deleteImageIntegration(autoCreatedIntegrationId)
                autoCreatedIntegrationId = null
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
            assert Services.waitForViolation(deployment.name, policy)
        }

        cleanup:
        if (integrationId) {
            ImageIntegrationService.deleteImageIntegration(integrationId)
        }
        if (secret) {
            orchestrator.deleteSecret(secret.name, secret.namespace)
        }
        if (autoCreatedIntegrationId) {
            ImageIntegrationService.deleteImageIntegration(autoCreatedIntegrationId)
        }
        if (deployment) {
            orchestrator.deleteAndWaitForDeploymentDeletion(deployment)
        }
        ImageService.clearImageCaches()
        ImageService.deleteImagesWithRetry(SearchServiceOuterClass.RawQuery.newBuilder()
                .setQuery("Image:${deployment.image}").build(), true)

        where:
        testName              | integration | deleteAutoRegistry | source |
                imageIntegrationConfig
        "ecr-iam"             | "ecr"       | false              | /^ecr$/ |
                { -> ECRRegistryIntegration.createCustomIntegration(useIam: true) }
        "ecr-auto"            | "ecr"       | false              | /Autogenerated .*.amazonaws.com for cluster remote/ |
                null
        "ecr-auto-and-config" | "ecr"       | false              | /^ecr$/ |
                { -> ECRRegistryIntegration.createDefaultIntegration() }
        "ecr-config-only"     | "ecr"       | true               | /^ecr$/  |
                { -> ECRRegistryIntegration.createDefaultIntegration() }
        "acr-auto"            | "acr"       | false              | /Autogenerated .*.azurecr.io for cluster remote/ |
                null
        "acr-auto-and-config" | "acr"       | false              | /^acr$/ |
                { -> AzureRegistryIntegration.createDefaultIntegration() }
        "acr-config-only"     | "acr"       | true               | /^acr$/  |
                { -> AzureRegistryIntegration.createDefaultIntegration() }
        "quay-auto"           | "quay"      | false              | /Autogenerated .*.quay.io for cluster remote/ |
                null
        "gcr-auto"            | "gcr"       | false              | /Autogenerated .*.gcr.io for cluster remote/ |
                null
    }

    @Unroll
    @Category(Integration)
    def "Image scanning test to check if scan time is not null #image from stackrox"() {
        when:
        "Add Stackrox scanner"
        def integrationId = StackroxScannerIntegration.createDefaultIntegration()
        assert integrationId

        and:
        "Image is scanned"
        def imageName = image
        Services.scanImage(imageName)

        then:
        "get image by name"
        String id = Services.getImageIdByName(imageName)
        ImageOuterClass.Image img = Services.getImageById(id)

        and:
        "check scanned time is not null"
        assert img.scan.scanTime != null
        assert img.scan.hasScanTime() == true

        cleanup:
        "Remove stackrox scanner and clear"
        ImageIntegrationService.deleteImageIntegration(integrationId)

        where:
        image                                    | registry
        "k8s.gcr.io/ip-masq-agent-amd64:v2.4.1"  | "gcr registry"
        "docker.io/jenkins/jenkins:lts"          | "docker registry"
        "docker.io/jenkins/jenkins:2.220-alpine" | "docker registry"
        "gke.gcr.io/heapster:v1.7.2"             | "one from gke"
        "mcr.microsoft.com/dotnet/core/runtime:2.1-alpine" | "one from mcr"
    }

    def "Validate basic image details across all current images in StackRox"() {
        when:
        "This is still flaky - disable for now until we get the issue resolved (ROX-4619)"
        Assume.assumeTrue(false)

        and:
        "get list of all images"
        List<ImageOuterClass.ListImage> images = ImageService.getImages()

        then:
        "validate details for each image"
        Map<ImageOuterClass.ImageName, List<ImageOuterClass.EmbeddedVulnerability>> missingValues = [:]
        for (ImageOuterClass.ListImage image : images) {
            ImageOuterClass.Image imageDetails = ImageService.getImage(image.id)

            if (imageDetails.hasScan()) {
                assert imageDetails.scan.scanTime
                for (ImageOuterClass.EmbeddedImageScanComponent component : imageDetails.scan.componentsList) {
                    for (ImageOuterClass.EmbeddedVulnerability vuln : component.vulnsList) {
                        if (vuln.summary == null || vuln.summary == "" ||
                                0.0 > vuln.cvss || vuln.cvss > 10.0 ||
                                vuln.link == null || vuln.link == "") {
                            missingValues.containsKey(imageDetails.name) ?
                                    missingValues.get(imageDetails.name).add(vuln) :
                                    missingValues.put(imageDetails.name, [vuln])
                        }
                    }
                }
            }
        }
        println missingValues
        assert missingValues.size() == 0
    }

    private static String expectAutoGeneratedRegistry(Secret secret) {
        ImageIntegrationOuterClass.ImageIntegration autoGenerated = null
        withRetry(5, 2) {
            autoGenerated =
                    ImageIntegrationService.getImageIntegrationByName(
                            "Autogenerated ${secret.server} for cluster remote"
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
