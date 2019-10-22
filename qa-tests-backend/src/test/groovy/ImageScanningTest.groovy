import groups.BAT
import groups.Integration
import io.stackrox.proto.storage.ImageOuterClass
import org.junit.experimental.categories.Category
import spock.lang.Unroll

class ImageScanningTest extends BaseSpecification {

    def setupSpec() {
        Services.addStackroxScannerIntegration()
    }

    def cleanupSpec() {
        Services.deleteAutoRegisteredStackRoxScannerIntegrationIfExists()
    }

    @Unroll
    @Category([BAT, Integration])
    def "Verify Image Scan Results - #image - #component:#version - #cve - #layer idx"() {
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

        image           | component                | version                  | layerIdx | cve
        "richxsl/rhel7@sha256:8f3aae325d2074d2dc328cb532d6e7aeb0c588e15ddf847347038fe0566364d6" | "openssl-libs" |
                "1:1.0.1e-34.el7" | 1 | "RHSA-2014:1052"

        "richxsl/rhel7@sha256:8f3aae325d2074d2dc328cb532d6e7aeb0c588e15ddf847347038fe0566364d6" | "openssl-libs" |
                "1:1.0.1e-34.el7" | 1 | "CVE-2014-3509"
    }

}
