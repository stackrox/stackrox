import io.stackrox.proto.storage.Cve.VulnerabilitySeverity

import services.GraphQLService
import services.ImageService
import util.Env

import spock.lang.IgnoreIf
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
          components(query: \$query) {
            vulns(query: \$query) {
              ...cveFields
            }
          }
        }
      }
    }

fragment cveFields on EmbeddedVulnerability {
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

    def getRHELComponentID() {
        return "glib2#2.54.2-2.el7#centos:7"
    }

    def getUbuntuComponentID() {
        return "glib2.0#2.40.2-0ubuntu1#ubuntu:14.04"
    }

    @Unroll
    // TODO(ROX-29222): Fix the test for getting cves by component
    @IgnoreIf({ Env.ROX_FLATTEN_CVE_DATA == "true" })
    def "Verify severities and CVSS #imageDigest #component #severity #cvss"() {
        when:
        def gqlService = new GraphQLService()

        def query="CVE:CVE-2019-13012"

        def embeddedImageRes = gqlService.Call(getEmbeddedImageQuery(),
                [id: imageDigest, query: query])

        // Expanded instead of using hasErrors() for easier debugging if there are errors
        // as the test framework will actually print out the errors now
        assert embeddedImageRes.code == 200
        if (embeddedImageRes.getErrors() != null) {
            assert embeddedImageRes.getErrors().size() == 0
        }

        def embeddedImageResVuln = embeddedImageRes.value.result.scan.components[0].vulns[0]

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
        imageDigest | component | componentID | severity | cvss
        RHEL_IMAGE_DIGEST   | "glib2" | getRHELComponentID()   |
                VulnerabilitySeverity.LOW_VULNERABILITY_SEVERITY | 4.4
        UBUNTU_IMAGE_DIGEST | "glib2.0"      | getUbuntuComponentID() |
                VulnerabilitySeverity.MODERATE_VULNERABILITY_SEVERITY      | 7.5
    }
}
