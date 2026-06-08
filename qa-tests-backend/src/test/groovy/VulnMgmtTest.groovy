import io.stackrox.proto.storage.Cve.VulnerabilitySeverity

import org.junit.Assume
import services.GraphQLService
import services.ImageService
import util.Helpers

import spock.lang.Tag
import spock.lang.Unroll

@Tag("BAT")
@Tag("PZ")
class VulnMgmtTest extends BaseSpecification {
    static final private String RHEL_IMAGE_DIGEST =
            "sha256:a3fb564e8be461d5bf8344996eb3ef6eb24a4b8c9333053fe4f3e782657591d3"
    static final private String RHEL_IMAGE =
            "quay.io/rhacs-eng/qa:ubi9-9.7-1769417801-amd64"

    // Used by StackRox Scanner tests - the legacy scanner may misidentify UBI9 as centos:9
    // due to a scanner-db namespace detection issue, so we use an older RHEL image instead.
    static final private String RHEL_IMAGE_LEGACY_DIGEST =
            "sha256:481960439934084fb041431f27cb98b89666e1a0daaeb2078bcbe1209790368c"
    static final private String RHEL_IMAGE_LEGACY =
            "quay.io/rhacs-eng/qa:ansibleplaybookbundle-" +
            "-gluster-s3object-apb-" +
            "-481960439934084fb041431f27cb98b89666e1a0daaeb2078bcbe1209790368c"

    static final private String UBUNTU_IMAGE_DIGEST =
            "sha256:c9672795a48854502d9dc0f1b719ac36dd99259a2f8ce425904a5cb4ae0d60d2"
    static final private String UBUNTU_IMAGE =
            "quay.io/rhacs-eng/qa:ubuntu-22.04-amd64"

    static final private MODERATE = VulnerabilitySeverity.MODERATE_VULNERABILITY_SEVERITY
    static final private LOW = VulnerabilitySeverity.LOW_VULNERABILITY_SEVERITY

    private static final EMBEDDED_IMAGE_QUERY = """
    query getImage(\$id: ID!, \$query: String) {
      result: fullImage(id: \$id) {
        scan {
          imageComponents(query: \$query) {
            imageVulnerabilities(query: \$query) {
              ...cveFields
            }
          }
        }
      }
    }

fragment cveFields on ImageVulnerability {
  cve
  cvss
  severity
}
"""

    private static final TOPLEVEL_IMAGE_QUERY = """
    query getImage(\$id: ID!, \$query: String) {
      result: image(id: \$id) {
        vulns: imageVulnerabilities(query: \$query) {
          ...cveFields
        }
      }
    }

    fragment cveFields on ImageVulnerability {
      cvss
      severity
    }
    """

    private static final IMAGE_FIXABLE_CVE_QUERY = """
query getFixableCvesForEntity(\$id: ID!, \$scopeQuery: String, \$vulnQuery: String) {
  result: image(id: \$id) {
    vulnerabilities: imageVulnerabilities(
      query: \$vulnQuery
      scopeQuery: \$scopeQuery
    ) {
      ...cveFields
      __typename
    }
    __typename
  }
}

fragment cveFields on ImageVulnerability {
  cvss
  severity
}
"""

    private static final COMPONENT_FIXABLE_CVE_QUERY = """
query getFixableCvesForEntity(\$id: ID!, \$scopeQuery: String, \$vulnQuery: String) {
  result: imageComponent(id: \$id) {
    vulnerabilities: imageVulnerabilities(
      query: \$vulnQuery
      scopeQuery: \$scopeQuery
    ) {
      ...cveFields
      __typename
    }
    __typename
  }
}

fragment cveFields on ImageVulnerability {
  cve
  cvss
  severity
}
"""

    private static final COMPONENT_SUBCVE_QUERY = """
query getComponentSubEntityCVE(\$id: ID!, \$query: String, \$scopeQuery: String) {
  result: imageComponent(id: \$id) {
    vulns: imageVulnerabilities(query: \$query, scopeQuery: \$scopeQuery) {
      ...cveFields
    }
    __typename
  }
}

fragment cveFields on ImageVulnerability {
  cvss
  severity
}
"""

    // Query to fetch component ID by name from an image
    private static final GET_COMPONENT_ID_QUERY = """
query getComponentId(\$imageId: ID!, \$componentQuery: String) {
  result: image(id: \$imageId) {
    imageComponents(query: \$componentQuery) {
      id
      name
      version
      operatingSystem
    }
  }
}
"""

    def setupSpec() {
        if (scannerV4Enabled) {
            ImageService.scanImage(RHEL_IMAGE)
        } else {
            ImageService.scanImage(RHEL_IMAGE_LEGACY)
        }
        ImageService.scanImage(UBUNTU_IMAGE)
    }

    def getEmbeddedImageQuery() {
        return EMBEDDED_IMAGE_QUERY
    }

    def getTopLevelImageQuery() {
        return TOPLEVEL_IMAGE_QUERY
    }

    def getImageFixableCVEQuery() {
        return IMAGE_FIXABLE_CVE_QUERY
    }

    def getComponentFixableCVEQuery() {
        return COMPONENT_FIXABLE_CVE_QUERY
    }

    def getComponentSubCVEQuery() {
        return COMPONENT_SUBCVE_QUERY
    }

    /**
     * Fetches the component ID dynamically from the API by querying the image's components.
     * Component IDs now include the image ID and index, so they cannot be constructed manually.
     */
    private String getComponentIDForImage(GraphQLService gqlService, String imageDigest, String componentName) {
        def componentQuery = "Component:${componentName}"
        def result = gqlService.Call(GET_COMPONENT_ID_QUERY,
                [imageId: imageDigest, componentQuery: componentQuery])
        assert result.code == 200
        if (result.getErrors() != null) {
            assert result.getErrors().size() == 0
        }
        def components = result.value.result.imageComponents
        assert components != null && components.size() > 0 : "No component found with name ${componentName}"
        return components[0].id
    }

    @Unroll
    def "Verify severities and CVSS - StackRox Scanner - #cve #imageDigest #component #severity #cvss"() {
        given:
        Assume.assumeFalse(scannerV4Enabled)

        expect:
        verifySeveritiesAndCvss(imageDigest, imageName, component, cve, severity, cvss)

        where:
        "Data inputs are: "

        imageDigest               | imageName           | component | cve              | severity | cvss
        RHEL_IMAGE_LEGACY_DIGEST  | RHEL_IMAGE_LEGACY   | "glib2"   | "CVE-2019-13012" | LOW      | 4.4
        UBUNTU_IMAGE_DIGEST       | UBUNTU_IMAGE        | "systemd"  | "CVE-2023-7008" | LOW      | 5.9
    }

    @Unroll
    def "Verify severities and CVSS - Scanner V4 - #cve #imageDigest #component #severity #cvss"() {
        given:
        Assume.assumeTrue(scannerV4Enabled)

        expect:
        verifySeveritiesAndCvss(imageDigest, imageName, component, cve, severity, cvss)

        where:
        "Data inputs are: "

        imageDigest         | imageName    | component | cve              | severity | cvss
        RHEL_IMAGE_DIGEST   | RHEL_IMAGE   | "python3" | "CVE-2025-11468" | MODERATE | 4.5
        UBUNTU_IMAGE_DIGEST | UBUNTU_IMAGE | "gpgv"    | "CVE-2022-3219"  | LOW      | 3.3
    }

    private void verifySeveritiesAndCvss(String imageDigest, String imageName, String component, String cve,
                                          VulnerabilitySeverity severity, double cvss) {
        def gqlService = new GraphQLService()

        def query = "CVE:${cve}"
        def imageId = flattenImageDataEnabled ? Helpers.generateImageV2ID(imageName, imageDigest) : imageDigest

        // Fetch the component ID dynamically since IDs now include image ID and index
        def componentID = getComponentIDForImage(gqlService, imageId, component)

        def embeddedImageRes = gqlService.Call(getEmbeddedImageQuery(),
                [id: imageId, query: query])

        // Expanded instead of using hasErrors() for easier debugging if there are errors
        // as the test framework will actually print out the errors now
        assert embeddedImageRes.code == 200
        if (embeddedImageRes.getErrors() != null) {
            assert embeddedImageRes.getErrors().size() == 0
        }

        def embeddedImageResVuln = embeddedImageRes.value.result.scan.imageComponents[0].imageVulnerabilities[0]

        def topLevelImageRes = gqlService.Call(getTopLevelImageQuery(),
                [id: imageId, query: query])
        assert topLevelImageRes.hasNoErrors()
        def topLevelImageResVuln = topLevelImageRes.value.result.vulns[0]

        def fixableCVEImageRes = gqlService.Call(getImageFixableCVEQuery(),
                [id: imageId, vulnQuery: query, scopeQuery: "Image SHA:${imageDigest}"])
        assert fixableCVEImageRes.hasNoErrors()
        def fixableCVEImageResVuln = fixableCVEImageRes.value.result.vulnerabilities[0]

        def fixableCVEComponentRes = gqlService.Call(getComponentFixableCVEQuery(),
                [id: componentID, vulnQuery: query, scopeQuery: "Image SHA:${imageDigest}"])
        assert fixableCVEComponentRes.hasNoErrors()
        def fixableCVEComponentResVuln = fixableCVEComponentRes.value.result.vulnerabilities[0]

        def subCVEComponentRes = gqlService.Call(getComponentSubCVEQuery(),
                [id: componentID, query: query, scopeQuery: "Image SHA:${imageDigest}"])
        assert subCVEComponentRes.hasNoErrors()
        def subCVEComponentResVuln = subCVEComponentRes.value.result.vulns[0]

        assert Math.round(embeddedImageResVuln.cvss * 10) / 10 == cvss
        assert embeddedImageResVuln.severity == severity.toString()

        assert Math.round(topLevelImageResVuln.cvss * 10) / 10 == cvss
        assert topLevelImageResVuln.severity == severity.toString()

        assert Math.round(fixableCVEImageResVuln.cvss * 10) / 10 == cvss
        assert fixableCVEImageResVuln.severity == severity.toString()

        assert Math.round(fixableCVEComponentResVuln.cvss * 10) / 10 == cvss
        assert fixableCVEComponentResVuln.severity == severity.toString()

        assert Math.round(subCVEComponentResVuln.cvss * 10) / 10 == cvss
        assert subCVEComponentResVuln.severity == severity.toString()
    }
}
