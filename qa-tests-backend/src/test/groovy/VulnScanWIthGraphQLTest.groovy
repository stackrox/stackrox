import groups.GraphQL
import objects.Deployment
import org.apache.commons.lang.StringUtils
import services.GraphQLService
import spock.lang.Shared
import spock.lang.Unroll
import org.junit.experimental.categories.Category
import util.Timer

class VulnScanWIthGraphQLTest extends BaseSpecification {
    static final private String STRUTSDEPLOYMENT_VULN_SCAN = "qastruts"
    static final private List<Deployment> DEPLOYMENTS = [
    new Deployment()
    .setName (STRUTSDEPLOYMENT_VULN_SCAN)
    .setImage ("apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
    .addLabel ("app", "test" ),
    ]

    private static final String GET_CVES_INFO_WITH_IMAGE_QUERY = """
    query image(\$id: ID!) {
        image:
        image(sha: \$id) {
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
            components {
                name
                layerIndex
                version
                license {
                    name
                    type
                    url
                }
                vulns {
                    cve
                    cvss
                    link
                    summary
                }
            }
        }
        }
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
    @Category(GraphQL)
    def "Verify image vuln,cves,cvss in GraphQL"() {
        when:
        "Fetch the results of the images from GraphQL "
        gqlService = new GraphQLService()
        String uid = DEPLOYMENTS.find { it.name == depName }.deploymentUid
        assert uid != null
        def imageId = waitForValidImageID(uid)
        println "image id ..." + imageId
        assert !StringUtils.isEmpty(imageId)
        def resultRet = gqlService.Call(GET_CVES_INFO_WITH_IMAGE_QUERY, [ id: imageId ])
        assert resultRet.getCode() == 200
        println "return code " + resultRet.getCode()
        then:
        assert resultRet.getValue() != null
        def image = resultRet.getValue().image
        assert image?.scan?.components?.vulns != null
        int cve =  getCVEs(image.scan.components.vulns)
        assert cve >= vuln_cve
        where :
        "Data inputs are :"
        depName | vuln_cve
        STRUTSDEPLOYMENT_VULN_SCAN | 219
    }

    private String getImageIDFromDepId(String id) {
        println "id " + id
        def resultRet = gqlService.Call(DEP_QUERY, [ id: id ])
        println "code " + resultRet.getCode()
        assert resultRet.getCode() == 200
        String imageID
        assert resultRet.getValue() != null
        def dep = resultRet.getValue().deployment
        if (dep != null && dep.images != null) {
            for (Object img : dep.images) {
                if (img.name != null && img.name.fullName.contains("struts") ) {
                    println " img.name ..." + img.name
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
        println "number of CVEs " + numCVEs
        return numCVEs
    }

    private String waitForValidImageID(String depID, int iterations = 30, int interval = 2) {
        Timer t = new Timer(iterations, interval)
        String imageID
        while (t.IsValid()) {
            imageID = getImageIDFromDepId(depID)
            if (!StringUtils.isEmpty(imageID)) {
                println "imageID found using deployment query "
                return imageID
            }
            println "imageID not found for ${depID} yet "
        }
        println "could not find  imageID from  ${depID} in ${iterations * interval} seconds"
        return ""
    }
}
