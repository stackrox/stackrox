import services.ProcessService

import groups.BAT
import groups.RUNTIME
import groups.SMOKE
import spock.lang.Unroll
import objects.Deployment
import org.junit.experimental.categories.Category
import util.Timer

class ProcessVisualizationTest extends BaseSpecification {
    // Deployment names
    static final private String NGINXDEPLOYMENT = "qanginx"
    static final private String STRUTSDEPLOYMENT = "qastruts"
    static final private String CENTOSDEPLOYMENT = "centosdeployment"
    static final private String FEDORADEPLOYMENT = "fedoradeployment"
    static final private String ELASTICDEPLOYMENT = "elasticdeployment"
    static final private String REDISDEPLOYMENT = "redisdeployment"
    static final private String MONGODEPLOYMENT = "mongodeployment"

    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                .setName (NGINXDEPLOYMENT)
                .setImage ("nginx:1.14-alpine")
                .addLabel ( "app", "test" ),
            new Deployment()
                .setName (STRUTSDEPLOYMENT)
                .setImage ("apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (CENTOSDEPLOYMENT)
                .setImage ("centos@sha256:fc2476ccae2a5186313f2d1dadb4a969d6d2d4c6b23fa98b6c7b0a1faad67685")
                .setCommand(["/bin/sh", "-c", "/bin/sleep 600"])
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (FEDORADEPLOYMENT)
                .setImage ("fedora@sha256:6fb84ba634fe68572a2ac99741062695db24b921d0aa72e61ee669902f88c187")
                .setCommand(["/bin/sh", "-c", "/bin/sleep 600"])
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (ELASTICDEPLOYMENT)
                .setImage ("elasticsearch@sha256:cdeb134689bb0318a773e03741f4414b3d1d0ee443b827d5954f957775db57eb")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (REDISDEPLOYMENT)
                .setImage ("redis@sha256:96be1b5b6e4fe74dfe65b2b52a0fee254c443184b34fe448f3b3498a512db99e")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (MONGODEPLOYMENT)
                .setImage ("mongo@sha256:dec7f10108a87ff660a0d56cb71b0c5ae1f33cba796a33c88b50280fc0707116")
                .addLabel ("app", "test" ),
     ]

    static final private MAX_SLEEP_TIME = 180000
    static final private SLEEP_INCREMENT = 5000

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

    @Category([BAT, SMOKE, RUNTIME])
    @Unroll
    def "Verify process visualization on default: #depName"()  {
        when:
        "Get Process IDs running on deployment: #depName"
        String uid = DEPLOYMENTS.find { it.name == depName }.deploymentUid
        assert uid != null

        Set<String> receivedProcessPaths
        int retries = MAX_SLEEP_TIME / SLEEP_INCREMENT
        int delaySeconds = SLEEP_INCREMENT / 1000
        Timer t = new Timer(retries, delaySeconds)
        while (t.IsValid()) {
            receivedProcessPaths = ProcessService.getUniqueProcessPaths(uid)
            if (receivedProcessPaths.containsAll(expectedFilePaths)) {
                break
            }
            println "Didn't find all the expected processes, retrying..."
        }
        println "ProcessVisualizationTest: Dep: " + depName + " Processes: " + receivedProcessPaths

        then:
        "Verify process in added : : #depName"
        assert receivedProcessPaths.containsAll(expectedFilePaths)

        where:
        "Data inputs are :"

        expectedFilePaths | depName

        ["/usr/sbin/nginx"] as Set | NGINXDEPLOYMENT

        ["/docker-java-home/jre/bin/java",
         "/usr/bin/tty", "/bin/uname",
         "/usr/local/tomcat/bin/catalina.sh",
         "/usr/bin/dirname"] as Set | STRUTSDEPLOYMENT

        ["/bin/sh", "/bin/sleep"] as Set | CENTOSDEPLOYMENT

        ["/bin/sh", "/bin/sleep"] as Set | FEDORADEPLOYMENT

        ["/usr/bin/tr", "/bin/chown", "/bin/egrep", "/bin/grep",
         "/usr/local/bin/gosu", "/bin/hostname",
         "/usr/share/elasticsearch/bin/elasticsearch", "/sbin/ldconfig",
         "/docker-entrypoint.sh", "/usr/bin/cut", "/usr/bin/id",
         "/docker-java-home/jre/bin/java", "/usr/bin/dirname"] as Set | ELASTICDEPLOYMENT

        ["/usr/bin/id", "/usr/bin/find", "/usr/local/bin/docker-entrypoint.sh",
         "/usr/local/bin/gosu", "/usr/local/bin/redis-server"] as Set | REDISDEPLOYMENT

        ["/bin/chown", "/usr/local/bin/docker-entrypoint.sh",
         "/bin/rm", "/usr/bin/id", "/usr/bin/find",
         "/usr/local/bin/gosu", "/usr/bin/mongod", "/usr/bin/numactl"] as Set | MONGODEPLOYMENT
   }
}
