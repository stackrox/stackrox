import io.stackrox.proto.storage.Cve.VulnerabilitySeverity

import services.GraphQLService
import services.ImageService

import spock.lang.Tag
import spock.lang.Unroll

@Tag("BAT")

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
      result: image(id: \$id) {
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

    private static final EMBEDDED_IMAGE_POSTGRES_QUERY = """
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
        vulns(query: \$query) {
          ...cveFields
        }
      }
    }

    fragment cveFields on EmbeddedVulnerability {
      cvss
      severity
    }
    """

    private static final TOPLEVEL_IMAGE_POSTGRES_QUERY = """
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
    vulnerabilities: vulns(
      query: \$vulnQuery
      scopeQuery: \$scopeQuery
    ) {
      ...cveFields
      __typename
    }
    __typename
  }
}

fragment cveFields on EmbeddedVulnerability {
  cve
  cvss
  severity
}
"""

    private static final IMAGE_FIXABLE_CVE_POSTGRES_QUERY = """
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
  result: component(id: \$id) {
    vulnerabilities: vulns(
      query: \$vulnQuery
      scopeQuery: \$scopeQuery
    ) {
      ...cveFields
      __typename
    }
    __typename
  }
}

fragment cveFields on EmbeddedVulnerability {
  cve
  cvss
  severity
}
"""

    private static final COMPONENT_FIXABLE_CVE_POSTGRES_QUERY = """
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
  result: component(id: \$id) {
    vulns(query: \$query, scopeQuery: \$scopeQuery) {
      ...cveFields
    }
    __typename
  }
}

fragment cveFields on EmbeddedVulnerability {
  cvss
  severity
}
"""

    private static final COMPONENT_SUBCVE_POSTGRES_QUERY = """
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
        return isPostgresRun() ? EMBEDDED_IMAGE_POSTGRES_QUERY : EMBEDDED_IMAGE_QUERY
    }

    def getTopLevelImageQuery() {
        return isPostgresRun() ? TOPLEVEL_IMAGE_POSTGRES_QUERY : TOPLEVEL_IMAGE_QUERY
    }

    def getImageFixableCVEQuery() {
        return isPostgresRun() ? IMAGE_FIXABLE_CVE_POSTGRES_QUERY : IMAGE_FIXABLE_CVE_QUERY
    }

    def getComponentFixableCVEQuery() {
        return isPostgresRun() ? COMPONENT_FIXABLE_CVE_POSTGRES_QUERY : COMPONENT_FIXABLE_CVE_QUERY
    }

    def getComponentSubCVEQuery() {
        return isPostgresRun() ? COMPONENT_SUBCVE_POSTGRES_QUERY : COMPONENT_SUBCVE_QUERY
    }

    def getRHELComponentID() {
        return isPostgresRun() ?
            "ncurses-base#5.9-14.20130511.el7_4#centos:7" :
            "bmN1cnNlcy1iYXNl:NS45LTE0LjIwMTMwNTExLmVsN180"
    }

    def getUbuntuComponentID() {
        return isPostgresRun() ?
            "ncurses#5.9+20140118-1ubuntu1#ubuntu:14.04" :
            "bmN1cnNlcw:NS45KzIwMTQwMTE4LTF1YnVudHUx"
    }

    @Unroll
    def "Verify severities and CVSS #imageDigest #component #severity #cvss"() {
        when:
        def gqlService = new GraphQLService()

        def embeddedImageRes = gqlService.Call(getEmbeddedImageQuery(),
                [id: imageDigest, query: "CVE:CVE-2017-10684"])

        // Expanded instead of using hasErrors() for easier debugging if there are errors
        // as the test framework will actually print out the errors now
        assert embeddedImageRes.code == 200
        if (embeddedImageRes.getErrors() != null) {
            assert embeddedImageRes.getErrors().size() == 0
        }

        def embeddedImageResVuln = embeddedImageRes.value.result.scan.components[0].vulns[0]

        def topLevelImageRes = gqlService.Call(getTopLevelImageQuery(),
                [id: imageDigest, query: "CVE:CVE-2017-10684"])
        assert topLevelImageRes.hasNoErrors()
        def topLevelImageResVuln =  topLevelImageRes.value.result.vulns[0]

        def fixableCVEImageRes = gqlService.Call(getImageFixableCVEQuery(),
                [id: imageDigest, vulnQuery: "CVE:CVE-2017-10684", scopeQuery: "Image SHA:${imageDigest}"])
        assert fixableCVEImageRes.hasNoErrors()
        def fixableCVEImageResVuln = fixableCVEImageRes.value.result.vulnerabilities[0]

        def fixableCVEComponentRes = gqlService.Call(getComponentFixableCVEQuery(),
                [id: componentID, vulnQuery: "CVE:CVE-2017-10684", scopeQuery: "Image SHA:${imageDigest}"])
        assert fixableCVEComponentRes.hasNoErrors()
        def fixableCVEComponentResVuln = fixableCVEComponentRes.value.result.vulnerabilities[0]

        def subCVEComponentRes = gqlService.Call(getComponentSubCVEQuery(),
                [id: componentID, query: "CVE:CVE-2017-10684", scopeQuery: "Image SHA:${imageDigest}"])
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
        RHEL_IMAGE_DIGEST   | "ncurses-base" | getRHELComponentID()   |
                VulnerabilitySeverity.MODERATE_VULNERABILITY_SEVERITY | 5.3
        UBUNTU_IMAGE_DIGEST | "ncurses"      | getUbuntuComponentID() |
                VulnerabilitySeverity.LOW_VULNERABILITY_SEVERITY      | 9.8
    }
}
