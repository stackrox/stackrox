import groups.BAT
import groups.Integration
import org.junit.experimental.categories.Category
import spock.lang.Shared
import spock.lang.Unroll
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage

class ImageManagementTest extends BaseSpecification {
    @Shared
    private String gcrId
    @Shared
    private String azureId

    def setupSpec() {
        gcrId = Services.addGcrRegistryAndScanner()
        assert gcrId != null

        azureId = Services.addAzureACRRegistry()
        assert azureId != null
    }

    def cleanupSpec() {
        assert Services.deleteGcrRegistryAndScanner(gcrId)
        assert Services.deleteImageIntegration(azureId)
    }

    @Unroll
    @Category([BAT, Integration])
    def "Verify CI/CD Integration Endpoint - #policy - #imageRegistry"() {
        when:
        "Update Policy to build time"
        def startStages = Services.updatePolicyLifecycleStage(policy, [LifecycleStage.BUILD,])

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

        policy                                        | imageRegistry            | imageRemote              | imageTag
        "Latest tag"                                  | "apollo-dtr.rox.systems" | "legacy-apps/struts-app" | "latest"
        //intentionally use the same policy twice to make sure alert count does not increment
        "Latest tag"                                  | "apollo-dtr.rox.systems" | "legacy-apps/struts-app" | "latest"
        "90-Day Image Age"                            | "apollo-dtr.rox.systems" | "legacy-apps/struts-app" | "latest"
        // verify Azure registry
        "90-Day Image Age"                            | "stackroxacr.azurecr.io" | "nginx" | "1.12"
        "Ubuntu Package Manager in Image"             | "apollo-dtr.rox.systems" | "legacy-apps/struts-app" | "latest"
        "Curl in Image"                               | "apollo-dtr.rox.systems" | "legacy-apps/struts-app" | "latest"
        "Fixable CVSS >= 7"                           | "us.gcr.io" | "stackrox-ci/nginx" | "1.11"
        "Wget in Image"                               | "apollo-dtr.rox.systems" | "legacy-apps/struts-app" | "latest"
        "Apache Struts: CVE-2017-5638"                | "apollo-dtr.rox.systems" | "legacy-apps/struts-app" | "latest"
    }

    @Unroll
    @Category([BAT, Integration])
    def "Verify CI/CD Integration Endpoint Whitelists - #policy"() {
        when:
        "Update Policy to build time and mark policy whitelist"
        def startStages = Services.updatePolicyLifecycleStage(policy, [LifecycleStage.BUILD,])
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
}
