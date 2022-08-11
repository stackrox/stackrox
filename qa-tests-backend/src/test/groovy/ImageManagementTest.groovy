import groups.BAT
import groups.Integration
import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.PolicyServiceOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import objects.Deployment
import objects.GenericNotifier
import org.junit.experimental.categories.Category
import services.CVEService
import services.ImageService
import services.PolicyService
import spock.lang.Unroll
import util.Env

class ImageManagementTest extends BaseSpecification {

    @Unroll
    @Category([BAT, Integration])
    def "Verify CI/CD Integration Endpoint - #policy - #imageRegistry #note"() {
        when:
        "Update Policy to build time"
        def startStages = Services.updatePolicyLifecycleStage(policy, [LifecycleStage.BUILD, LifecycleStage.DEPLOY])

        and:
        "Update Policy to be enabled"
        def policyEnabled = Services.setPolicyDisabled(policy, false)

        and:
        "Request Image Scan"
        def scanResults = Services.requestBuildImageScan(imageRegistry, imageRemote, imageTag)

        then:
        "verify policy exists in response"
        assert scanResults.getAlertsList().findAll { it.getPolicy().name == policy }.size() == 1

        cleanup:
        "Revert Policy"
        Services.updatePolicyLifecycleStage(policy, startStages)
        if (policyEnabled) {
            Services.setPolicyDisabled(policy, true)
        }

        where:
        "Data inputs are: "

        policy                            | imageRegistry | imageRemote              | imageTag     | note
        "Latest tag"                      | "docker.io"   | "library/nginx"          | "latest"     | ""
        //intentionally use the same policy twice to make sure alert count does not increment
        "Latest tag"                      | "docker.io"   | "library/nginx"          | "latest"     | "(repeat)"
        "90-Day Image Age"                | "quay.io"   | "rhacs-eng/qa"            | "struts-app" | ""
        // verify Azure registry
        // "90-Day Image Age"                | "stackroxacr.azurecr.io" | "nginx"                  | "1.12"   | ""
        "Ubuntu Package Manager in Image" | "quay.io"   | "rhacs-eng/qa"            | "struts-app" | ""
        "Curl in Image"                   | "quay.io"   | "rhacs-eng/qa"            | "struts-app" | ""
        "Fixable CVSS >= 7"               | "us.gcr.io"   | "stackrox-ci/nginx"      | "1.11"       | ""
        "Wget in Image"                   | "quay.io"   | "rhacs-eng/qa"            | "struts-app" | ""
        "Apache Struts: CVE-2017-5638"    | "quay.io"   | "rhacs-eng/qa"            | "struts-app" | ""
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
        "alpine:3.10.0"         | "docker.io"   | "library/alpine"  | "3.10.0"         | "alpine:v3.10"
        "busybox:1.32.0"        | "docker.io"   | "library/busybox" | "1.32.0"         | "busybox:1.32.0"
        "centos:centos8.2.2004" | "docker.io"   | "library/centos"  | "centos8.2.2004" | "centos:8"
        // We explicitly do not support Fedora at this time.
        "fedora:33"             | "docker.io"   | "library/fedora"  | "33"             | "unknown"
        "nginx:1.10"            | "docker.io"   | "library/nginx"   | "1.10"           | "debian:8"
        "nginx:1.19"            | "docker.io"   | "library/nginx"   | "1.19"           | "debian:10"
        // TODO: Add check for RHEL
        "ubuntu:14.04"          | "docker.io"   | "library/ubuntu"  | "14.04"          | "ubuntu:14.04"
    }

    @Unroll
    @Category([BAT, Integration])
    def "Verify CI/CD Integration Endpoint excluded scopes - #policy - #excludedscopes"() {
        when:
        "Update Policy to build time and mark policy excluded scope"
        def startStages = Services.updatePolicyLifecycleStage(policy, [LifecycleStage.BUILD, LifecycleStage.DEPLOY])
        Services.updatePolicyImageExclusion(policy, excludedscopes)

        and:
        "Request Image Scan"
        def scanResults = Services.requestBuildImageScan(imageRegistry, imageRemote, imageTag)

        then:
        "verify violation matches expected violation status"
        assert expectedViolation == (scanResults.getAlertsList().findAll { it.getPolicy().name == policy }.size() == 1)

        cleanup:
        "Revert Policy"
        Services.updatePolicyLifecycleStage(policy, startStages)
        Services.updatePolicyImageExclusion(policy, [])

        where:
        "Data inputs are: "

        policy       | imageRegistry | imageRemote       | imageTag | excludedscopes | expectedViolation
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
                .addPolicySections(
                        PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                PolicyOuterClass.PolicyGroup.newBuilder()
                                        .setFieldName("CVE")
                                        .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue("CVE-2019-14697")
                                                .build()).build()
                        ).build()
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
        def cve = "CVE-2019-14697"
        CVEService.suppressImageCVE(cve)
        scanResults = Services.requestBuildImageScan("docker.io", "docker/kube-compose-controller", "v0.4.23")
        assert scanResults.alertsList.find { y -> y.policy.id == policy.id } == null

        and:
        "Unsuppress CVE"
        CVEService.unsuppressImageCVE(cve)
        scanResults = Services.requestBuildImageScan("docker.io", "docker/kube-compose-controller", "v0.4.23")

        then:
        "Verify unsuppressing lets the CVE show up again"
        assert scanResults.alertsList.find { z -> z.policy.id == policy.id } != null

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
                .setSkipReplicaWait(Env.CI_JOBNAME && Env.CI_JOBNAME.contains("openshift-crio"))

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

        def cve = "CVE-2010-0928"
        CVEService.suppressImageCVE(cve)

        when:
        def scanIncludeSnoozed = ImageService.scanImage("library/nginx:1.10", true)
        assert hasOpenSSLVuln(scanIncludeSnoozed)

        def scanExcludedSnoozed = ImageService.scanImage("library/nginx:1.10", false)
        assert !hasOpenSSLVuln(scanExcludedSnoozed)

        def getIncludeSnoozed  = ImageService.getImage(image.id, true)
        assert hasOpenSSLVuln(getIncludeSnoozed)

        def getExcludeSnoozed  = ImageService.getImage(image.id, false)
        assert !hasOpenSSLVuln(getExcludeSnoozed)

        CVEService.unsuppressImageCVE(cve)

        def unsuppressedScan = ImageService.scanImage("library/nginx:1.10", false)
        def unsuppressedGet  = ImageService.getImage(image.id, false)

        then:

        assert hasOpenSSLVuln(unsuppressedScan)
        assert hasOpenSSLVuln(unsuppressedGet)

        cleanup:
        // Should be able to call this multiple times safely in case of any failures previously
        CVEService.unsuppressImageCVE(cve)
    }

    @Category([BAT, Integration])
    def "Verify CI/CD Integration Endpoint with notifications"() {
        when:
        "Update policy to build time, create notifier and add it to policy"
        def notifier = new GenericNotifier()
        notifier.createNotifier()
        assert notifier.id

        def policyName = "Latest tag"
        def startPolicy = Services.getPolicyByName(policyName)
        assert startPolicy

        def newPolicy = PolicyOuterClass.Policy.newBuilder(startPolicy)
            .clearLifecycleStages()
            .addLifecycleStages(LifecycleStage.BUILD)
            .addNotifiers(notifier.id)
            .build()
        Services.updatePolicy(newPolicy)

        and:
        "Request Image Scan with sendNotifications"
        def scanResults = Services.requestBuildImageScan("docker.io", "library/busybox", "latest", true)

        then:
        "verify violation matches expected violation status and notification sent"
        assert scanResults.getAlertsList().findAll { it.getPolicy().name == policyName }.size() == 1
        withRetry(2, 3) {
            def genericViolation = GenericNotifier.getMostRecentViolationAndValidateCommonFields()
            log.info "Most recent violation sent: ${genericViolation}"
            def alert = genericViolation["data"]["alert"]
            assert alert != null
            assert alert["policy"]["name"] == policyName
            assert alert["image"] != null
            assert alert["deployment"] == null
            assert alert["image"]["name"]["fullName"] == "docker.io/library/busybox:latest"
            assert alert["image"]["name"]["registry"] == "docker.io"
            assert alert["image"]["name"]["remote"] == "library/busybox"
            assert alert["image"]["name"]["tag"] == "latest"
        }

        cleanup:
        "Revert policy and clean up notifier"
        if (startPolicy != null) {
            Services.updatePolicy(startPolicy)
        }
        notifier.deleteNotifier()
    }

}
