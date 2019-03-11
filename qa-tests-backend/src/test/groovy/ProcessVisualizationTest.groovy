import services.ProcessService

import groups.BAT
import groups.SMOKE
import spock.lang.Unroll
import objects.Deployment
import org.junit.experimental.categories.Category

class ProcessVisualizationTest extends BaseSpecification {
    // Deployment names
    static final private String NGINXDEPLOYMENT = "qanginx"
    static final private String STRUTSDEPLOYMENT = "qastruts"
    static final private String SSL_TERMINATOR = "qasslterm"
    static final private String APACHEDEPLOYMENT = "apacheserverdeployment"
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
                .setName (SSL_TERMINATOR)
                .setImage ("apollo-dtr.rox.systems/legacy-apps/ssl-terminator:latest")
                .addLabel ("app", "test" ) ,
            new Deployment()
                .setName (APACHEDEPLOYMENT)
                .setImage ("apollo-dtr.rox.systems/legacy-apps/apache-server")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (CENTOSDEPLOYMENT)
                .setImage ("centos@sha256:6f6d986d425aeabdc3a02cb61c02abb2e78e57357e92417d6d58332856024faf")
                .setCommand(["/bin/sh", "-c", "/bin/sleep 600"])
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (FEDORADEPLOYMENT)
                .setImage ("fedora@sha256:b41cd083421dd7aa46d619e958b75a026a5d5733f08f14ba6d53943d6106ea6d")
                .setCommand(["/bin/sh", "-c", "/bin/sleep 600"])
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (ELASTICDEPLOYMENT)
                .setImage ("elasticsearch@sha256:a8081d995ef3443dc6d077093172a5931e02cdb8ffddbf05c67e01d348a9770e")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (REDISDEPLOYMENT)
                .setImage ("redis@sha256:911f976312f503692709ad9534f15e2564a0967f2aa6dd08a74c684fb1e53e1a")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (MONGODEPLOYMENT)
                .setImage ("mongo@sha256:e9bab21970befb113734c6ec549a4cf90377961dbe0ec94fe65be2a0abbdcc30")
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

    @Category([BAT, SMOKE])
    @Unroll
    def "Verify process visualization on default: #depName"()  {
        when:
        "Get Process IDs running on deployment: #depName"
        String uid = DEPLOYMENTS.find { it.name == depName }.deploymentUid
        assert uid != null

        Set<String> receivedProcessPaths = ProcessService.getUniqueProcessPaths(uid)
        def sleepTime = 0L
        while (!receivedProcessPaths.equals(expectedFilePaths) && sleepTime < MAX_SLEEP_TIME) {
            println "Didn't find all the expected processes, retrying..."
            sleep(SLEEP_INCREMENT)
            sleepTime += SLEEP_INCREMENT
            receivedProcessPaths = ProcessService.getUniqueProcessPaths(uid)
        }
        println "ProcessVisualizationTest: Dep: " + depName + " Processes: " + receivedProcessPaths

        then:
        "Verify process in added : : #depName"
        assert receivedProcessPaths.equals(expectedFilePaths)

        where:
        "Data inputs are :"

        expectedFilePaths | depName

        ["/usr/sbin/nginx"] as Set | NGINXDEPLOYMENT

        ["/docker-java-home/jre/bin/java",
         "/usr/bin/tty", "/bin/uname",
         "/usr/local/tomcat/bin/catalina.sh",
         "/usr/bin/dirname"] as Set | STRUTSDEPLOYMENT

        // Need to skip because of DTR dumspterfire
        // ["/bin/mv", "/bin/cat", "/main.sh", "/usr/bin/stat", "/usr/sbin/apache2",
        // "/usr/sbin/apache2ctl", "/bin/mkdir", "/bin/chown",
        // "/bin/chmod", "/bin/mktemp"] as Set | SSL_TERMINATOR

        // Missing apache server image after DTR dumpsterfire - skipping for now
        // ["/bin/mktemp", "/bin/mv", "/main.sh", "/usr/sbin/apache2", "/usr/sbin/apache2ctl",
        //  "/bin/chown", "/usr/bin/stat", "/bin/chmod", "/bin/mkdir"] as Set | APACHEDEPLOYMENT

        ["/bin/sh", "/bin/sleep"] as Set | CENTOSDEPLOYMENT

        ["/bin/sh", "/bin/sleep"] as Set | FEDORADEPLOYMENT

        ["/usr/bin/tr", "/bin/chown", "/bin/egrep", "/bin/grep",
         "/usr/local/bin/gosu", "/bin/hostname",
         "/usr/share/elasticsearch/bin/elasticsearch", "/sbin/ldconfig",
         "/docker-entrypoint.sh", "/usr/bin/cut", "/usr/bin/id",
         "/docker-java-home/jre/bin/java", "/usr/bin/dirname"] as Set | ELASTICDEPLOYMENT

        ["/usr/bin/id", "/usr/bin/find", "/usr/local/bin/docker-entrypoint.sh",
         "/usr/local/bin/gosu", "/usr/local/bin/redis-server"] as Set | REDISDEPLOYMENT

        ["/bin/true", "/bin/chown", "/usr/local/bin/docker-entrypoint.sh",
         "/bin/rm", "/usr/bin/id", "/usr/bin/find",
         "/usr/local/bin/gosu", "/usr/bin/mongod", "/usr/bin/numactl"] as Set | MONGODEPLOYMENT
   }
}
