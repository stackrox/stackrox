import groups.BAT
import groups.Integration
import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.PolicyServiceOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import objects.Deployment
import org.junit.experimental.categories.Category
import services.CVEService
import services.ImageIntegrationService
import services.ImageService
import services.PolicyService
import spock.lang.Shared
import spock.lang.Unroll
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage

class ImageManagementTest extends BaseSpecification {
    @Shared
    private String azureId
    @Shared
    private static final boolean CHECK_AZURE = false

    def setupSpec() {
        if (CHECK_AZURE) {
            azureId = ImageIntegrationService.addAzureRegistry()
            assert azureId != ""
        }
    }

    def cleanupSpec() {
        if (CHECK_AZURE) {
            assert ImageIntegrationService.deleteImageIntegration(azureId)
        }
    }

    @Unroll
    @Category([BAT, Integration])
    def "Verify CI/CD Integration Endpoint - #policy - #imageRegistry #note"() {
        when:
        "Update Policy to build time"
        def startStages = Services.updatePolicyLifecycleStage(policy, [LifecycleStage.BUILD, LifecycleStage.DEPLOY])

        and:
        "Request Image Scan"
        def scanResults = Services.requestBuildImageScan(imageRegistry, imageRemote, imageTag)

        then:
        "verify policy exists in response"
        assert scanResults.getAlertsList().findAll { it.getPolicy().name == policy }.size() == 1

        cleanup:
        "Revert Policy"
        Services.updatePolicyLifecycleStage(policy, startStages)

        where:
        "Data inputs are: "

        policy                            | imageRegistry | imageRemote              | imageTag     | note
        "Latest tag"                      | "docker.io"   | "library/nginx"          | "latest"     | ""
        //intentionally use the same policy twice to make sure alert count does not increment
        "Latest tag"                      | "docker.io"   | "library/nginx"          | "latest"     | "(repeat)"
        "90-Day Image Age"                | "docker.io"   | "stackrox/qa"            | "struts-app" | ""
        // verify Azure registry
        // "90-Day Image Age"                | "stackroxacr.azurecr.io" | "nginx"                  | "1.12"   | ""
        "Ubuntu Package Manager in Image" | "docker.io"   | "stackrox/qa"            | "struts-app" | ""
        "Curl in Image"                   | "docker.io"   | "stackrox/qa"            | "struts-app" | ""
        "Fixable CVSS >= 7"               | "us.gcr.io"   | "stackrox-ci/nginx"      | "1.11"       | ""
        "Wget in Image"                   | "docker.io"   | "stackrox/qa"            | "struts-app" | ""
        "Apache Struts: CVE-2017-5638"    | "docker.io"   | "stackrox/qa"            | "struts-app" | ""
    }

    @Category(BAT)
    def "Verify two consecutive latest tag image have different scans"() {
        given:
        // Scan an ubuntu 14:04 image we're pretending is latest
        def img = Services.scanImage(
            "docker.io/library/ubuntu:latest@sha256:ffc76f71dd8be8c9e222d420dc96901a07b61616689a44c7b3ef6a10b7213de4")
        assert img.scan.componentsList.stream().find { x -> x.name == "eglibc" } != null

        img = Services.scanImage(
             "docker.io/library/ubuntu:latest@sha256:3235326357dfb65f1781dbc4df3b834546d8bf914e82cce58e6e6b676e23ce8f")

        expect:
        assert img.scan != null
        assert img.scan.componentsList.stream().find { x -> x.name == "eglibc" } == null
    }

    @Unroll
    @Category(BAT)
    def "Verify image scan finds correct base OS - #imageName"() {
        when:
        def img = Services.scanImage(imageRegistry + "/" + imageRemote + ":" + imageTag)
        then:
        assert img.scan.operatingSystem == expected
        where:
        "Data inputs are: "

        imageName               | imageRegistry | imageRemote       | imageTag         | expected
        "ubuntu:14.04"          | "docker.io"   | "library/ubuntu"  | "14.04"          | "ubuntu:14.04"
        "busybox:1.32.0"        | "docker.io"   | "library/busybox" | "1.32.0"         | "unknown"
        "alpine:3.10.1"         | "docker.io"   | "library/alpine"  | "3.10.0"         | "alpine:v3.10"
        "nginx:1.10"            | "docker.io"   | "library/nginx"   | "1.10"           | "debian:8"
        "nginx:1.19"            | "docker.io"   | "library/nginx"   | "1.19"           | "debian:10"
        "centos:centos8.2.2004" | "docker.io"   | "library/centos"  | "centos8.2.2004" | "centos:8"
        // This is due to weird Scanner functionality. Making this a test, in case we decide to come back to Scanner
        // and change this. Best to know what our expectations are.
        "fedora:33"             | "docker.io"   | "library/fedora"  | "33"             | "unknown"
    }

    @Unroll
    @Category([BAT, Integration])
    def "Verify CI/CD Integration Endpoint Whitelists - #policy - #whitelists"() {
        when:
        "Update Policy to build time and mark policy whitelist"
        def startStages = Services.updatePolicyLifecycleStage(policy, [LifecycleStage.BUILD, LifecycleStage.DEPLOY])
        Services.updatePolicyImageWhitelist(policy, whitelists)

        and:
        "Request Image Scan"
        def scanResults = Services.requestBuildImageScan(imageRegistry, imageRemote, imageTag)

        then:
        "verify violation matches expected violation status"
        assert expectedViolation == (scanResults.getAlertsList().findAll { it.getPolicy().name == policy }.size() == 1)

        cleanup:
        "Revert Policy"
        Services.updatePolicyLifecycleStage(policy, startStages)
        Services.updatePolicyImageWhitelist(policy, [])

        where:
        "Data inputs are: "

        policy       | imageRegistry | imageRemote       | imageTag | whitelists | expectedViolation
        "Latest tag" | "docker.io"   | "library/busybox" | "latest" | ["docker.io"]                         | false
        "Latest tag" | "docker.io"   | "library/busybox" | "latest" | ["docker.io/library"]                 | false
        "Latest tag" | "docker.io"   | "library/busybox" | "latest" | ["docker.io/library/busybox"]         | false
        "Latest tag" | "docker.io"   | "library/busybox" | "latest" | ["docker.io/library/busybox:latest"]  | false
        "Latest tag" | "docker.io"   | "library/busybox" | "latest" | ["other"]                             | true
        "Latest tag" | "docker.io"   | "library/busybox" | "latest" | ["docker.io/library/busybox:1.10"]    | true
        "Latest tag" | "docker.io"   | "library/busybox" | "latest" | ["library/busybox:1.10"]              | true
    }

    @Category(Integration)
    def "Verify lifecycle Stage can only be build time for policies with image criteria"() {
        when:
        "Update Policy to build time"
        def startStages = Services.updatePolicyLifecycleStage(
                "No resource requests or limits specified",
                [LifecycleStage.BUILD,]
        )

        then:
        "assert startStage is null"
        assert startStages == []
    }

    @Unroll
    @Category([BAT])
    def "Verify CVE snoozing applies to build time detection"() {
        given:
        "Create policy looking for a specific CVE applying to build time"
        PolicyOuterClass.Policy policy = PolicyOuterClass.Policy.newBuilder()
                .setName("Matching CVE (CVE-2019-14697)")
                .addLifecycleStages(LifecycleStage.BUILD)
                .addCategories("Testing")
                .setSeverity(PolicyOuterClass.Severity.HIGH_SEVERITY)
                .setFields(
                        PolicyOuterClass.PolicyFields.newBuilder().setCve("CVE-2019-14697").build()
                ).build()
        policy = PolicyService.policyClient.postPolicy(
                PolicyServiceOuterClass.PostPolicyRequest.newBuilder()
                    .setPolicy(policy)
                    .build()
        )
        def scanResults = Services.requestBuildImageScan("docker.io", "docker/kube-compose-controller", "v0.4.23")
        assert scanResults.alertsList.find { x -> x.policy.id == policy.id } != null

        when:
        "Suppress CVE and check that it violates"
        CVEService.suppressCVE("CVE-2019-14697")
        scanResults = Services.requestBuildImageScan("docker.io", "docker/kube-compose-controller", "v0.4.23")
        assert scanResults.alertsList.find { x -> x.policy.id == policy.id } == null

        and:
        "Unsuppress CVE"
        CVEService.unsuppressCVE("CVE-2019-14697")
        scanResults = Services.requestBuildImageScan("docker.io", "docker/kube-compose-controller", "v0.4.23")

        then:
        "Verify unsuppressing lets the CVE show up again"
        assert scanResults.alertsList.find { x -> x.policy.id == policy.id } != null

        cleanup:
        "Delete policy"
        PolicyService.policyClient.deletePolicy(Common.ResourceByID.newBuilder().setId(policy.id).build())
    }

    @Unroll
    @Category([BAT])
    def "Verify risk is properly being attributed to scanned images"() {
        when:
        "Scan an image and then grab the image data"
        ImageService.scanImage(
          "mysql@sha256:de2913a0ec53d98ced6f6bd607f487b7ad8fe8d2a86e2128308ebf4be2f92667")

        then:
        "Assert that riskScore is non-zero"
        withRetry(10, 3) {
            def image = ImageService.getImage(
                    "sha256:de2913a0ec53d98ced6f6bd607f487b7ad8fe8d2a86e2128308ebf4be2f92667")
            assert image != null && image.riskScore != 0
        }
    }

    @Unroll
    @Category([BAT])
    def "Verify risk is properly being attributed to run images"() {
        when:
        "Create deployment that runs an image and verify that image has a non-zero riskScore"
        def deployment = new Deployment()
                .setName("risk-image")
                .setReplicas(1)
                .setImage("mysql@sha256:f7985e36c668bb862a0e506f4ef9acdd1254cdf690469816f99633898895f7fa")
                .setCommand(["sleep", "60000"])

        orchestrator.createDeployment(deployment)

        then:
        "Assert that riskScore is non-zero"
        withRetry(10, 3) {
            def image = ImageService.getImage(
                    "sha256:f7985e36c668bb862a0e506f4ef9acdd1254cdf690469816f99633898895f7fa")
            assert image != null && image.riskScore != 0
        }

        cleanup:
        orchestrator.deleteDeployment(deployment)
    }

    def hasOpenSSLVuln(image) {
        return image?.getScan()?.getComponentsList().
                find { it.name == "openssl" }?.
                getVulnsList().find { it.cve == "CVE-2010-0928" } != null
    }

    @Unroll
    @Category([BAT])
    def "Verify image scan results when CVEs are suppressed: "() {
        given:
        "Scan image"
        def image = ImageService.scanImage("library/nginx:1.10", true)
        assert hasOpenSSLVuln(image)

        image = ImageService.getImage(image.id, true)
        assert hasOpenSSLVuln(image)

        CVEService.suppressCVE("CVE-2010-0928")

        when:
        def scanIncludeSnoozed = ImageService.scanImage("library/nginx:1.10", true)
        assert hasOpenSSLVuln(scanIncludeSnoozed)

        def scanExcludedSnoozed = ImageService.scanImage("library/nginx:1.10", false)
        assert !hasOpenSSLVuln(scanExcludedSnoozed)

        def getIncludeSnoozed  = ImageService.getImage(image.id, true)
        assert hasOpenSSLVuln(getIncludeSnoozed)

        def getExcludeSnoozed  = ImageService.getImage(image.id, false)
        assert !hasOpenSSLVuln(getExcludeSnoozed)

        CVEService.unsuppressCVE("CVE-2010-0928")

        def unsuppressedScan = ImageService.scanImage("library/nginx:1.10", false)
        def unsuppressedGet  = ImageService.getImage(image.id, false)

        then:

        assert hasOpenSSLVuln(unsuppressedScan)
        assert hasOpenSSLVuln(unsuppressedGet)

        cleanup:
        // Should be able to call this multiple times safely in case of any failures previously
        CVEService.unsuppressCVE("CVE-2010-0928")
    }

}
