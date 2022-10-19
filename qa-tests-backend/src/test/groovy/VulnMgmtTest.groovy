import groups.BAT
import io.stackrox.proto.storage.Cve.VulnerabilitySeverity
import org.junit.experimental.categories.Category
import services.GraphQLService
import services.ImageIntegrationService
import services.ImageService
import spock.lang.IgnoreIf
import spock.lang.Unroll
import util.Env

@Category(BAT)
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
    imageVulnerabilities: vulns(
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
    imageVulnerabilities: vulns(
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
    imageVulns(query: \$query, scopeQuery: \$scopeQuery) {
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

    def setupSpec() {
        ImageIntegrationService.addStackroxScannerIntegration()
        ImageService.scanImage(RHEL_IMAGE)
        ImageService.scanImage(UBUNTU_IMAGE)
    }

    def cleanupSpec() {
        ImageIntegrationService.deleteStackRoxScannerIntegrationIfExists()
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

    def getVulnQuery(String suffix) {
        return isPostgresRun() ?
            "CVE:CVE-2017-10684#" + suffix :
            "CVE:CVE-2017-10684"
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

        def embeddedImageRes = gqlService.Call(EMBEDDED_IMAGE_QUERY,
                [id: imageDigest, query: getVulnQuery(cveSuffix)])
        assert embeddedImageRes.hasNoErrors()
        def embeddedImageResVuln = embeddedImageRes.value.result.scan.components[0].vulns[0]

        def topLevelImageRes = gqlService.Call(TOPLEVEL_IMAGE_QUERY,
                [id: imageDigest, query: getVulnQuery(cveSuffix)])
        assert topLevelImageRes.hasNoErrors()
        def topLevelImageResVuln = topLevelImageRes.value.result.vulns[0]

        def fixableCVEImageRes = gqlService.Call(getImageFixableCVEQuery(),
                [id: imageDigest, vulnQuery: getVulnQuery(cveSuffix), scopeQuery: "Image SHA:${imageDigest}"])
        assert fixableCVEImageRes.hasNoErrors()
        def fixableCVEImageResVuln = fixableCVEImageRes.value.result.vulnerabilities[0]

        def fixableCVEComponentRes = gqlService.Call(getComponentFixableCVEQuery(),
                [id: componentID, vulnQuery: getVulnQuery(cveSuffix), scopeQuery: "Image SHA:${imageDigest}"])
        assert fixableCVEComponentRes.hasNoErrors()
        def fixableCVEComponentResVuln = fixableCVEComponentRes.value.result.vulnerabilities[0]

        def subCVEComponentRes = gqlService.Call(getComponentSubCVEQuery(),
                [id: componentID, query: getVulnQuery(cveSuffix), scopeQuery: "Image SHA:${imageDigest}"])
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
        imageDigest | component | componentID | cveSuffix | severity | cvss
        RHEL_IMAGE_DIGEST   | "ncurses-base" | getRHELComponentID()   | "centos:7"     |
                VulnerabilitySeverity.MODERATE_VULNERABILITY_SEVERITY | 5.3
        UBUNTU_IMAGE_DIGEST | "ncurses"      | getUbuntuComponentID() | "ubuntu:14.04" |
                VulnerabilitySeverity.LOW_VULNERABILITY_SEVERITY      | 9.8
    }
}
