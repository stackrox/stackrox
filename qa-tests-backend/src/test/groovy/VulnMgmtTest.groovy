import io.stackrox.proto.storage.Cve.VulnerabilitySeverity

import services.GraphQLService
import services.ImageService

import spock.lang.Tag
import spock.lang.Unroll

@Tag("BAT")
@Tag("PZ")
class VulnMgmtTest extends BaseSpecification {
    static final private String RHEL_IMAGE_DIGEST =
            "sha256:481960439934084fb041431f27cb98b89666e1a0daaeb2078bcbe1209790368c"
    static final private String RHEL_IMAGE =
            "quay.io/rhacs-eng/qa:ansibleplaybookbundle-"+
            "-gluster-s3object-apb-"+
            "-481960439934084fb041431f27cb98b89666e1a0daaeb2078bcbe1209790368c"

    static final private String UBUNTU_IMAGE_DIGEST =
            "sha256:74ee7a5d7a7172090162b1b5f8022b3b403b9f4ac677d325209c56483452f417"
    static final private String UBUNTU_IMAGE =
            "quay.io/rhacs-eng/qa:barchart-"+
            "-dockerup--ce6c28c63fa9a043214f4cccf036990dbd2bb0e47820af015de8dfb5dc68dd9a"

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
        ImageService.scanImage(RHEL_IMAGE)
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
    def "Verify severities and CVSS #imageDigest #component #severity #cvss"() {
        when:
        def gqlService = new GraphQLService()

        def query="CVE:CVE-2019-13012"

        // Fetch the component ID dynamically since IDs now include image ID and index
        def componentID = getComponentIDForImage(gqlService, imageDigest, component)

        def embeddedImageRes = gqlService.Call(getEmbeddedImageQuery(),
                [id: imageDigest, query: query])

        // Expanded instead of using hasErrors() for easier debugging if there are errors
        // as the test framework will actually print out the errors now
        assert embeddedImageRes.code == 200
        if (embeddedImageRes.getErrors() != null) {
            assert embeddedImageRes.getErrors().size() == 0
        }

        def embeddedImageResVuln = embeddedImageRes.value.result.scan.imageComponents[0].imageVulnerabilities[0]

        def topLevelImageRes = gqlService.Call(getTopLevelImageQuery(),
                [id: imageDigest, query: query])
        assert topLevelImageRes.hasNoErrors()
        def topLevelImageResVuln =  topLevelImageRes.value.result.vulns[0]

        def fixableCVEImageRes = gqlService.Call(getImageFixableCVEQuery(),
                [id: imageDigest, vulnQuery: query, scopeQuery: "Image SHA:${imageDigest}"])
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

        then:
        Math.round(embeddedImageResVuln.cvss * 10) / 10 == cvss
        embeddedImageResVuln.severity == severity.toString()

        Math.round(topLevelImageResVuln.cvss * 10) / 10 == cvss
        topLevelImageResVuln.severity == severity.toString()

        Math.round(fixableCVEImageResVuln.cvss * 10) / 10 == cvss
        fixableCVEImageResVuln.severity == severity.toString()

        Math.round(fixableCVEComponentResVuln.cvss * 10) / 10 == cvss
        fixableCVEComponentResVuln.severity == severity.toString()

        Math.round(subCVEComponentResVuln.cvss * 10) / 10 == cvss
        subCVEComponentResVuln.severity == severity.toString()

        where:
        "Data inputs are: "
        imageDigest | component | severity | cvss
        RHEL_IMAGE_DIGEST   | "glib2" |
                VulnerabilitySeverity.LOW_VULNERABILITY_SEVERITY | 4.4
        UBUNTU_IMAGE_DIGEST | "glib2.0" |
                VulnerabilitySeverity.MODERATE_VULNERABILITY_SEVERITY | 7.5
    }
}
