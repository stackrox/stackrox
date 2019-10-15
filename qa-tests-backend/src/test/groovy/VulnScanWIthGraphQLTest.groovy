import groups.GraphQL
import objects.Deployment
import org.apache.commons.lang.StringUtils
import services.GraphQLService
import spock.lang.Shared
import spock.lang.Unroll
import org.junit.experimental.categories.Category

class VulnScanWIthGraphQLTest extends BaseSpecification {
    static final private String DEPLOYMENTNGINX_VULN_SCAN = "vuln-scan-deploymentnginx"
    static final private String STRUTSDEPLOYMENT_VULN_SCAN = "qastruts"
    static final private List<Deployment> DEPLOYMENTS = [
                    new Deployment()
                            .setName(DEPLOYMENTNGINX_VULN_SCAN)
                            .setImage("nginx:1.7.9")
                            .addPort(22, "TCP")
                            .addAnnotation("test", "annotation")
                            .setEnv(["CLUSTER_NAME": "main"])
                            .addLabel("app", "test"),
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
        def imageId = getImageIDFromDepId(uid)
        println "image id ..." + imageId
        def resultRet = gqlService.Call(GET_CVES_INFO_WITH_IMAGE_QUERY, [ id: imageId ])
        assert resultRet.getCode() == 200
        println "return code " + resultRet.getCode()
        then:
        println "image results " + resultRet.getValue().toString()
        assert !(StringUtils.isEmpty(resultRet.getValue().toString()))
        int cve
        def vulns
        def scan = resultRet.getValue().image.scan
        println " scan " + scan
        if (scan != null) {
            vulns = scan.components.vulns
            println "vulns " + vulns
        }
        if (vulns != null) {
            cve =  getCVEs(vulns)
        }
        assert cve >= vuln_cve
        where :
        "Data inputs are :"
        depName | vuln_cve
        DEPLOYMENTNGINX_VULN_SCAN | 0
        STRUTSDEPLOYMENT_VULN_SCAN | 219
    }

    private String getImageIDFromDepId(String id) {
        String depQuery = """query getDeployment(\$id: ID!) {
        deployment :
        deployment(id: \$id) {
             images {
             id }
        }
    }
"""
        def resultRet = gqlService.Call(depQuery, [ id: id ])
        println "code " + resultRet.getCode()
        assert resultRet.getCode() == 200
        String imageID =  resultRet.getValue().deployment.images.id
        println "image id " + imageID[imageID.indexOf('[')+1 .. imageID.indexOf(']')-1]
        return imageID[imageID.indexOf('[')+1 .. imageID.indexOf(']')-1]
    }

    private int getCVEs(List vulns) {
        int numCVEs = 0
        for (List cves : vulns.cve) {
            numCVEs += cves.size()
        }
        println "number of CVEs " + numCVEs
        return numCVEs
    }
}
