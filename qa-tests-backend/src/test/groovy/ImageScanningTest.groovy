import groups.BAT
import groups.Integration
import io.stackrox.proto.storage.ImageOuterClass
import org.junit.experimental.categories.Category
import services.ImageIntegrationService
import spock.lang.Unroll

class ImageScanningTest extends BaseSpecification {

    def setupSpec() {
        ImageIntegrationService.addStackroxScannerIntegration()
    }

    def cleanupSpec() {
        ImageIntegrationService.deleteAutoRegisteredStackRoxScannerIntegrationIfExists()
    }

    @Unroll
    @Category([BAT, Integration])
    def "Verify Image Scan Results - #image - #component:#version - #cve - #layerIdx"() {
        when:
        "Scan Image and verify results"
        ImageOuterClass.Image img = Services.scanImage(image)

        then:
        ImageOuterClass.EmbeddedImageScanComponent foundComponent =
                img.scan.componentsList.find {
                    c -> c.name == component && c.version == version && c.layerIndex == layerIdx
                }
        foundComponent != null

        ImageOuterClass.EmbeddedVulnerability vuln =
                foundComponent.vulnsList.find { v -> v.cve == cve }

        vuln != null

        where:
        "Data inputs are: "

        image                                                                                   | component      |
                version | layerIdx | cve
        "richxsl/rhel7@sha256:8f3aae325d2074d2dc328cb532d6e7aeb0c588e15ddf847347038fe0566364d6" | "openssl-libs" |
                "1:1.0.1e-34.el7"      | 1        | "RHSA-2014:1052"

        "richxsl/rhel7@sha256:8f3aae325d2074d2dc328cb532d6e7aeb0c588e15ddf847347038fe0566364d6" | "openssl-libs" |
                "1:1.0.1e-34.el7"      | 1        | "CVE-2014-3509"
    }

    @Unroll
    def "Image scanning test to check if scan time is not null #image from stackrox"() {
        when:
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

        where:
        image                                    | registry
        "k8s.gcr.io/ip-masq-agent-amd64:v2.4.1"  | "gcr registry"
        "docker.io/jenkins/jenkins:lts"          | "docker registry"
        "docker.io/jenkins/jenkins:2.220-alpine" | "docker registry"
        "gke.gcr.io/heapster:v1.7.2"             | "one from gke"
    }
}
