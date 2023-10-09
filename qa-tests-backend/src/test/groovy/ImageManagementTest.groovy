import static util.Helpers.withRetry

import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import io.stackrox.proto.storage.ScopeOuterClass

import objects.Deployment
import objects.GenericNotifier
import services.CVEService
import services.ImageService
import services.PolicyService

import spock.lang.Tag
import spock.lang.Unroll
import spock.lang.IgnoreIf
import util.Env

@Tag("Parallel")
class ImageManagementTest extends BaseSpecification {

    private static final String TEST_NAMESPACE = "qa-image-management"

    private static final String FEDORA_28 = "fedora-6fb84ba634fe68572a2ac99741062695db24b921d0aa72e61ee669902f88c187"
    private static final String WGET_IMAGE_NS = ((Env.REMOTE_CLUSTER_ARCH == "x86_64") ?
        "rhacs-eng/qa":"rhacs-eng/qa-multi-arch")
    private static final String WGET_IMAGE_TAG = ((Env.REMOTE_CLUSTER_ARCH == "x86_64") ?
        "struts-app":"trigger-policy-violations-most")

    def cleanupSpec() {
        orchestrator.deleteNamespace(TEST_NAMESPACE)
    }

    @Unroll
    @Tag("BAT")
    @Tag("Integration")
    def "Verify CI/CD Integration Endpoint - #policyName - #imageRegistry #note"() {
        when:
        "Clone and scope the policy for test"
        Policy clone = PolicyService.clonePolicyAndScopeByNamespace(policyName, TEST_NAMESPACE)

        and:
        "Update Policy to build time"
        Services.updatePolicyLifecycleStage(clone.name, [LifecycleStage.BUILD, LifecycleStage.DEPLOY])

        and:
        "Update Policy to be enabled"
        Services.setPolicyDisabled(clone.name, false)

        and:
        "Request Image Scan"
        def scanResults = Services.requestBuildImageScan(imageRegistry, imageRemote, imageTag)

        then:
        "verify policy exists in response"
        assert scanResults.getAlertsList().findAll { it.getPolicy().name == clone.name }.size() == 1

        cleanup:
        "Delete policy clone"
        if (clone) {
            PolicyService.deletePolicy(clone.id)
        }

        where:
        "Data inputs are: "

        policyName                        | imageRegistry | imageRemote                      | imageTag     | note
        "Latest tag"                      | "quay.io"     | "rhacs-eng/qa-multi-arch-nginx"  | "latest"     | ""
        //intentionally use the same policy twice to make sure alert count does not increment
        "Latest tag"                      | "quay.io"     | "rhacs-eng/qa-multi-arch-nginx"  | "latest"     | "(repeat)"
        "90-Day Image Age"                | "quay.io"     | "rhacs-eng/qa-multi-arch"        | "struts-app" | ""
        // verify Azure registry
        // "90-Day Image Age"             | "stackroxacr.azurecr.io" | "nginx"               | "1.12"       | ""
        "Ubuntu Package Manager in Image" | "quay.io"     | "rhacs-eng/qa-multi-arch"        | "struts-app" | ""
        "Curl in Image"                   | "quay.io"     | "rhacs-eng/qa-multi-arch"        | "struts-app" | ""
        "Fixable CVSS >= 7"               | "quay.io"     | "rhacs-eng/qa-multi-arch"        | "nginx-1.12" | ""
        "Wget in Image"                   | "quay.io"     | WGET_IMAGE_NS                  | WGET_IMAGE_TAG | ""
        "Apache Struts: CVE-2017-5638"    | "quay.io"     | "rhacs-eng/qa-multi-arch"        | "struts-app" | ""
    }

    @Tag("BAT")
    def "Verify two consecutive latest tag image have different scans"() {
        given:
        // Scan an ubuntu 14:04 image we're pretending is latest
        def img = ImageService.scanImage(
            "quay.io/rhacs-eng/qa-multi-arch:ubuntu-latest" +
                "@sha256:64483f3496c1373bfd55348e88694d1c4d0c9b660dee6bfef5e12f43b9933b30", false) // 14.04
        assert img.scan.componentsList.stream().find { x -> x.name == "eglibc" } != null

        img = ImageService.scanImage(
            "quay.io/rhacs-eng/qa-multi-arch:ubuntu-latest" +
                "@sha256:1f1a2d56de1d604801a9671f301190704c25d604a416f59e03c04f5c6ffee0d6", false) // 16.04

        expect:
        assert img.scan != null
        assert img.scan.componentsList.stream().find { x -> x.name == "eglibc" } == null
    }

    @Unroll
    @Tag("BAT")
    def "Verify image scan finds correct base OS - #qaImageTag"() {
        when:
        def img = ImageService.scanImage("quay.io/rhacs-eng/qa:$qaImageTag", false)
        then:
        assert img.scan.operatingSystem == expected
        where:
        "Data inputs are: "

        qaImageTag             | expected
        "nginx-1.19-alpine"    | "alpine:v3.13"
        "busybox-1-30"         | "busybox:1.30.1"
        "centos7-base"         | "centos:7"
        // We explicitly do not support Fedora at this time.
        FEDORA_28              | "unknown"
        "nginx-1-9"            | "debian:8"
        "nginx-1-17-1"         | "debian:9"
        "ubi9-slf4j"           | "rhel:9"
        "apache-server"        | "ubuntu:14.04"
        "ubuntu-22.10-openssl" | "ubuntu:22.10"
    }

    @Unroll
    @Tag("BAT")
    @Tag("Integration")
    def "Verify CI/CD Integration Endpoint excluded scopes - #policyName - #excludedscopes"() {
        when:
        "Clone and scope the policy for test"
        Policy clone = PolicyService.clonePolicyAndScopeByNamespace(policyName, TEST_NAMESPACE)

        and:
        "Update Policy to build time and mark policy excluded scope"
        Services.updatePolicyLifecycleStage(clone.name, [LifecycleStage.BUILD, LifecycleStage.DEPLOY])
        Services.updatePolicyImageExclusion(clone.name, excludedscopes)

        and:
        "Request Image Scan"
        def scanResults = Services.requestBuildImageScan(imageRegistry, imageRemote, imageTag)

        then:
        "verify violation matches expected violation status"
        assert expectedViolation == (
            scanResults.getAlertsList().findAll { it.getPolicy().name == clone.name }.size() == 1
        )

        cleanup:
        "Delete policy clone"
        if (clone) {
            PolicyService.deletePolicy(clone.id)
        }

        where:
        "Data inputs are: "

        policyName   | imageRegistry | imageRemote                       | imageTag | excludedscopes | expectedViolation
        "Latest tag" | "quay.io"     | "rhacs-eng/qa-multi-arch-busybox" | "latest" | ["quay.io"]           | false
        "Latest tag" | "quay.io"     | "rhacs-eng/qa-multi-arch-busybox" | "latest" | ["quay.io/rhacs-eng"] | false
        "Latest tag" | "quay.io"     | "rhacs-eng/qa-multi-arch-busybox" | "latest" |
                      ["quay.io/rhacs-eng/qa-multi-arch-busybox"]        | false
        "Latest tag" | "quay.io"     | "rhacs-eng/qa-multi-arch-busybox" | "latest" |
                      ["quay.io/rhacs-eng/qa-multi-arch-busybox:latest"] | false
        "Latest tag" | "quay.io"     | "rhacs-eng/qa-multi-arch-busybox" | "latest" | ["other"]             | true
        "Latest tag" | "quay.io"     | "rhacs-eng/qa-multi-arch-busybox" | "latest" |
                      ["quay.io/rhacs-eng/qa-multi-arch-busybox:1.30"]   | true
        "Latest tag" | "quay.io"     | "rhacs-eng/qa-multi-arch-busybox" | "latest" |
                      ["rhacs-eng/qa-multi-arch-busybox:1.30"]           | true
    }

    @Tag("Integration")
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
    @Tag("BAT")
    def "Verify CVE snoozing applies to build time detection"() {
        given:
        "Create policy looking for a specific CVE applying to build time"
        PolicyOuterClass.Policy policy = PolicyOuterClass.Policy.newBuilder()
                .setName("Matching CVE (CVE-2019-14697)")
                .addLifecycleStages(LifecycleStage.BUILD)
                .addCategories("Testing")
                .setSeverity(PolicyOuterClass.Severity.HIGH_SEVERITY)
                .addPolicySections(
                        PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                PolicyOuterClass.PolicyGroup.newBuilder()
                                        .setFieldName("CVE")
                                        .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue("CVE-2019-14697")
                                                .build()).build()
                        ).build()
                )
                .clearScope()
                .addScope(ScopeOuterClass.Scope.newBuilder().setNamespace(TEST_NAMESPACE))
                .build()
        policy = PolicyService.createAndFetchPolicy(policy)
        def scanResults = Services.requestBuildImageScan("quay.io", "rhacs-eng/qa", "kube-compose-controller-v0.4.23")
        assert scanResults.alertsList.find { x -> x.policy.id == policy.id } != null

        when:
        "Suppress CVE and check that it violates"
        def cve = "CVE-2019-14697"
        CVEService.suppressImageCVE(cve)
        scanResults = Services.requestBuildImageScan("quay.io", "rhacs-eng/qa", "kube-compose-controller-v0.4.23")
        assert scanResults.alertsList.find { y -> y.policy.id == policy.id } == null

        and:
        "Unsuppress CVE"
        CVEService.unsuppressImageCVE(cve)
        scanResults = Services.requestBuildImageScan("quay.io", "rhacs-eng/qa", "kube-compose-controller-v0.4.23")

        then:
        "Verify unsuppressing lets the CVE show up again"
        assert scanResults.alertsList.find { z -> z.policy.id == policy.id } != null

        cleanup:
        "Delete policy"
        PolicyService.deletePolicy(policy.id)
    }

    @Unroll
    @Tag("BAT")
    def "Verify risk is properly being attributed to scanned images"() {
        when:
        "Scan an image and then grab the image data"
        ImageService.scanImage(
            "quay.io/rhacs-eng/qa-multi-arch-nginx@" +
            "sha256:6650513efd1d27c1f8a5351cbd33edf85cc7e0d9d0fcb4ffb23d8fa89b601ba8")

        then:
        "Assert that riskScore is non-zero"
        withRetry(10, 3) {
            def image = ImageService.getImage(
                    "sha256:6650513efd1d27c1f8a5351cbd33edf85cc7e0d9d0fcb4ffb23d8fa89b601ba8")
            assert image != null && image.riskScore != 0
        }
    }

    @Unroll
    @Tag("BAT")
    def "Verify risk is properly being attributed to run images"() {
        when:
        "Create deployment that runs an image and verify that image has a non-zero riskScore"
        def deployment = new Deployment()
                .setName("risk-image")
                .setNamespace(TEST_NAMESPACE)
                .setReplicas(1)
                .setImage("quay.io/rhacs-eng/qa-multi-arch-nginx" +
                    "@sha256:6650513efd1d27c1f8a5351cbd33edf85cc7e0d9d0fcb4ffb23d8fa89b601ba8")
                .setCommand(["sleep", "60000"])
                .setSkipReplicaWait(false)

        orchestrator.createDeployment(deployment)

        then:
        "Assert that riskScore is non-zero"
        withRetry(10, 3) {
            def image = ImageService.getImage(
                    "sha256:6650513efd1d27c1f8a5351cbd33edf85cc7e0d9d0fcb4ffb23d8fa89b601ba8")
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
    @Tag("BAT")
    def "Verify image scan results when CVEs are suppressed: "() {
        given:
        "Scan image"
        def image = ImageService.scanImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1.12", true)
        assert hasOpenSSLVuln(image)

        image = ImageService.getImage(image.id, true)
        assert hasOpenSSLVuln(image)

        def cve = "CVE-2010-0928"
        CVEService.suppressImageCVE(cve)

        when:
        def scanIncludeSnoozed = ImageService.scanImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1.12", true)
        assert hasOpenSSLVuln(scanIncludeSnoozed)

        def scanExcludedSnoozed = ImageService.scanImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1.12", false)
        assert !hasOpenSSLVuln(scanExcludedSnoozed)

        def getIncludeSnoozed  = ImageService.getImage(image.id, true)
        assert hasOpenSSLVuln(getIncludeSnoozed)

        def getExcludeSnoozed  = ImageService.getImage(image.id, false)
        assert !hasOpenSSLVuln(getExcludeSnoozed)

        CVEService.unsuppressImageCVE(cve)

        def unsuppressedScan = ImageService.scanImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1.12", false)
        def unsuppressedGet  = ImageService.getImage(image.id, false)

        then:

        assert hasOpenSSLVuln(unsuppressedScan)
        assert hasOpenSSLVuln(unsuppressedGet)

        cleanup:
        // Should be able to call this multiple times safely in case of any failures previously
        CVEService.unsuppressImageCVE(cve)
    }

    @Tag("BAT")
    @Tag("Integration")
    @IgnoreIf({ Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x" })
    def "Verify CI/CD Integration Endpoint with notifications"() {
        when:
        "Clone and scope the policy for test"
        Policy clone = PolicyService.clonePolicy("Latest tag", "Latest tag - ${TEST_NAMESPACE}")

        and:
        "Create a notifier"
        def notifier = new GenericNotifier("Generic Notifier - ${TEST_NAMESPACE}")
        notifier.createNotifier()
        assert notifier.id

        and:
        "Update policy to build time and add the notifier"
        def update = clone.toBuilder()
            .clearLifecycleStages()
            .addLifecycleStages(LifecycleStage.BUILD)
            .addNotifiers(notifier.id)
            .addCategories("DevOps Best Practices") // required for putPolicy
            .clearExclusions()
            .build()
        Services.updatePolicy(update)

        and:
        "Request Image Scan with sendNotifications"
        def scanResults = Services.requestBuildImageScan("quay.io", "quay/busybox", "latest", true)

        then:
        "verify violation matches expected violation status and notification sent"
        assert scanResults.getAlertsList().findAll { it.getPolicy().name == clone.name }.size() == 1
        withRetry(2, 3) {
            def genericViolation = GenericNotifier.getMostRecentViolationAndValidateCommonFields()
            log.info "Most recent violation sent: ${genericViolation}"
            def alert = genericViolation["data"]["alert"]
            assert alert != null
            assert alert["policy"]["name"] == clone.name
            assert alert["image"] != null
            assert alert["deployment"] == null
            assert alert["image"]["name"]["fullName"] == "quay.io/quay/busybox:latest"
            assert alert["image"]["name"]["registry"] == "quay.io"
            assert alert["image"]["name"]["remote"] == "quay/busybox"
            assert alert["image"]["name"]["tag"] == "latest"
        }

        cleanup:
        "Revert policy and clean up notifier"
        if (clone) {
            PolicyService.deletePolicy(clone.id)
        }
        if (notifier) {
            notifier.deleteNotifier()
        }
    }
}
