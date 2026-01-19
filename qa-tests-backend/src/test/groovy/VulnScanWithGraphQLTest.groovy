import org.apache.commons.lang3.StringUtils

import objects.Deployment
import services.GraphQLService
import util.Timer

import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Unroll

@Tag("BAT")
@Tag("GraphQL")
@Tag("PZ")
class VulnScanWithGraphQLTest extends BaseSpecification {
    static final private String STRUTSDEPLOYMENT_VULN_SCAN = "qastruts"
    static final private Deployment STRUTS_DEP = new Deployment()
            .setName (STRUTSDEPLOYMENT_VULN_SCAN)
            .setImage ("quay.io/rhacs-eng/qa-multi-arch:struts-app")
            .addLabel ("app", "test" )
    static final private List<Deployment> DEPLOYMENTS = [
    STRUTS_DEP,
    ]

    private static final String GET_CVES_INFO_WITH_IMAGE_QUERY = """
    query image(\$id: ID!) {
        image:
        image(id: \$id) {
           //cve_info here
           id
        lastUpdated
        deployments {
            id
            name
        }
         name {
            fullName
            registry
            remote
            tag
        }
        scan {
            imageComponents {
                name
                layerIndex
                version
                license {
                    name
                    type
                    url
                }
                imageVulnerabilities {
                    cve
                    cvss
                    link
                    summary
                }
            }
        }
        }
    }"""

    private static final String GET_IMAGE_INFO_FROM_VULN_QUERY = """
    query getCve(\$id: ID!) {
        result: imageVulnerability(id: \$id) {
        cve
        cvss
        scoreVersion
        link
        vectors {
          __typename
          ... on CVSSV2 {
            impactScore
            exploitabilityScore
            vector
          }
          ... on CVSSV3 {
            impactScore
            exploitabilityScore
            vector
          }
        }
        summary
        fixedByVersion
        isFixable
        lastScanned
        imageComponentCount
        imageCount
        deploymentCount
        images {
            id  name {fullName} scan {
                scanTime
            }}}
    }"""

    private static final String DEP_QUERY = """query getDeployment(\$id: ID!) {
        deployment :
        deployment(id: \$id) {
             images {
             scan
             {
             scanTime} id name {fullName}
             }
        }
    }
"""

    // Query to find CVE by name and get its actual ID
    // CVE IDs now include component ID and index, so they cannot be constructed manually
    private static final String GET_CVE_ID_BY_NAME_QUERY = """
    query getCveByName(\$query: String!) {
        result: imageVulnerabilities(query: \$query) {
            id
            cve
        }
    }
"""

    @Shared
    private  gqlService = new GraphQLService()

    def setupSpec() {
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment)
        }
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
    }

    @Unroll
    def "Verify image vuln,cves,cvss on #depName in GraphQL"() {
        when:
        "Fetch the results of the images from GraphQL "
        gqlService = new GraphQLService()
        String uid = DEPLOYMENTS.find { it.name == depName }.deploymentUid
        assert uid != null
        def imageId = waitForValidImageID(uid)
        log.info "image id ..." + imageId
        assert !StringUtils.isEmpty(imageId)
        def resultRet = gqlService.Call(GET_CVES_INFO_WITH_IMAGE_QUERY, [ id: imageId ])
        assert resultRet.getCode() == 200
        log.info "return code " + resultRet.getCode()
        then:
        assert resultRet.getValue() != null
        def image = resultRet.getValue().image
        assert image?.scan?.imageComponents?.imageVulnerabilities != null
        int cve =  getCVEs(image.scan.imageComponents.imageVulnerabilities)
        assert cve >= vuln_cve
        where:
        "Data inputs are :"
        depName | vuln_cve
        STRUTSDEPLOYMENT_VULN_SCAN | 138
    }

    @Unroll
    def "Verify image info from #CVEID in GraphQL"() {
        when:
        "Fetch the results of the CVE,image from GraphQL "
        GraphQLService.Response result2Ret = waitForImagesTobeFetched(CVEID)
        assert result2Ret.getValue()?.result?.images  != null
        then :
        List<Object> imagesReturned = result2Ret.getValue().result.images
        assert imagesReturned != null
        String imgName = imagesReturned.find { it.name.fullName == imageToBeVerified }
        assert !(StringUtils.isEmpty(imgName))
        where:
        "Data inputs are :"
        CVEID           | imageToBeVerified
        "CVE-2017-5638" | STRUTS_DEP.getImage()
    }

    /**
     * Fetches images associated with a CVE by first looking up the CVE's actual ID.
     * CVE IDs now include component ID and index, so they must be fetched from the API
     * rather than constructed manually.
     */
    private GraphQLService.Response waitForImagesTobeFetched(String cveName,
     int retries = 30, int interval = 4) {
        Timer t = new Timer(retries, interval)

        // First, find the actual CVE ID by querying for it by name
        String actualCveId = null
        while (t.IsValid() && actualCveId == null) {
            def cveQuery = "CVE:${cveName}"
            def cveResult = gqlService.Call(GET_CVE_ID_BY_NAME_QUERY, [query: cveQuery])
            assert cveResult.getCode() == 200
            def cves = cveResult.getValue()?.result
            if (cves != null && cves.size() > 0) {
                actualCveId = cves[0].id
                log.info "Found CVE ID: ${actualCveId} for CVE name: ${cveName}"
            } else {
                log.info "CVE ${cveName} not found yet, retrying..."
            }
        }

        if (actualCveId == null) {
            log.info "Unable to find CVE ${cveName} in ${t.SecondsSince()} seconds"
            return null
        }

        // Now use the actual CVE ID to get the vulnerability details with images
        while (t.IsValid()) {
            def result2Ret = gqlService.Call(GET_IMAGE_INFO_FROM_VULN_QUERY, [id: actualCveId])
            assert result2Ret.getCode() == 200
            if (result2Ret.getValue().result != null) {
                log.info "images fetched from cve"
                return result2Ret
            }
        }
        log.info "Unable to fetch images for ${cveName} in ${t.SecondsSince()} seconds"
        return null
    }

    private String getImageIDFromDepId(String id) {
        log.info "id " + id
        def resultRet = gqlService.Call(DEP_QUERY, [ id: id ])
        log.info "code " + resultRet.getCode()
        assert resultRet.getCode() == 200
        String imageID
        assert resultRet.getValue() != null
        def dep = resultRet.getValue().deployment
        if (dep != null && dep.images != null) {
            for (Object img : dep.images) {
                if (img.name != null && img.name.fullName.contains("struts") ) {
                    log.info " img.name ..." + img.name
                    imageID = img.id
                    break
                }
            }
        }
        return imageID
    }

    private int getCVEs(List vulns) {
        int numCVEs = 0
        for (List cves : vulns.cve) {
            numCVEs += cves.size()
        }
        log.info "number of CVEs " + numCVEs
        return numCVEs
    }

    private String waitForValidImageID(String depID, int retries = 30, int interval = 2) {
        Timer t = new Timer(retries, interval)
        String imageID
        while (t.IsValid()) {
            imageID = getImageIDFromDepId(depID)
            if (!StringUtils.isEmpty(imageID)) {
                log.info "imageID found using deployment query "
                return imageID
            }
            log.info "imageID not found for ${depID} yet "
        }
        log.info "could not find  imageID from  ${depID} in ${t.SecondsSince()} seconds"
        return ""
    }
}
