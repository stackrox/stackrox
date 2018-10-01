
import static Services.getProcessOnDeployment
import groups.BAT
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
    //static final private String ELASTICDEPLOYMENT = "elasticdeployment"
    //static final private String REDISDEPLOYMENT = "redisdeployment"
    //static final private String MONGODEPLOYMENT = "mongodeployment"

    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                        .setName (NGINXDEPLOYMENT)
                        .setImage ("nginx:1.14-alpine")
                        .addLabel ( "app", "test" ) ,
            new Deployment()
                        .setName (STRUTSDEPLOYMENT)
                        .setImage ( "apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
                        .addLabel ( "app", "test" ),
            new Deployment()
                        .setName (SSL_TERMINATOR)
                        .setImage ( "apollo-dtr.rox.systems/legacy-apps/ssl-terminator:latest")
                        .addLabel ( "app", "test" ) ,
            new Deployment()
                        .setName (APACHEDEPLOYMENT)
                        .setImage ( "apollo-dtr.rox.systems/legacy-apps/apache-server")
                        .addLabel ( "app", "test" ),
            new Deployment()
                        .setName (CENTOSDEPLOYMENT)
                        .setImage ( "centos")
                        .setCommand(["/bin/sh", "-c", "/bin/sleep 600"])
                        .addLabel ( "app", "test" ),
            new Deployment()
                        .setName (FEDORADEPLOYMENT)
                        .setImage ( "fedora")
                        .setCommand(["/bin/sh", "-c", "/bin/sleep 600"])
                        .addLabel ( "app", "test" ),
            //new Deployment()
            //            .setName (ELASTICDEPLOYMENT)
            //            .setImage ( "elasticsearch:latest")
            //            .addLabel ( "app", "test" ),
            //new Deployment()
            //            .setName (REDISDEPLOYMENT)
            //            .setImage ( "redis")
            //            .addLabel ( "app", "test" ),
            //new Deployment()
            //            .setName (MONGODEPLOYMENT)
            //            .setImage ( "mongo")
            //            .addLabel ( "app", "test" ),
     ]

    static final private MAX_SLEEP_TIME = 60000
    static final private SLEEP_INCREMENT = 5000

    def setupSpec() {
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment.getName())
        }
    }

    @Category(BAT)
    @Unroll
    def "Verify process visualization on default: #depName"()  {
        when:
        "Get Process IDs running on deployment: #depName"
        String uid = orchestrator?.getDeploymentId(depName)
        assert uid != null

        List<String> receivedProcessPaths = getProcessOnDeployment(uid)
        def sleepTime = 0L
        while ((!receivedProcessPaths.containsAll(expectedFilePaths)) && sleepTime < MAX_SLEEP_TIME) {
            println "Didn't find all the expected processes, retrying..."
            sleep(SLEEP_INCREMENT)
            sleepTime += SLEEP_INCREMENT
            receivedProcessPaths = getProcessOnDeployment(uid)
        }
        println "ProcessVisualizationTest: Dep: " + depName + " Processes: " + receivedProcessPaths

        then:
        "Verify process in added : : #depName"
        assert receivedProcessPaths.containsAll(expectedFilePaths)

        where:
        "Data inputs are :"

        expectedFilePaths |  depName

        ["/usr/sbin/nginx"] | NGINXDEPLOYMENT

        ["/docker-java-home/jre/bin/java",
         "/usr/bin/tty",
         "/usr/local/tomcat/bin/catalina.sh",
         "/usr/bin/dirname"]  | STRUTSDEPLOYMENT

        ["/bin/mv", "/bin/cat", "/usr/bin/stat", "/main.sh",
          "/usr/sbin/apache2", "/usr/sbin/apache2ctl", "/bin/mkdir", "/bin/chmod"] | SSL_TERMINATOR

        ["/bin/mktemp", "/bin/mv", "/main.sh", "/usr/sbin/apache2",
          "/bin/chown", "/usr/bin/stat", "/bin/chmod", "/bin/mkdir"]  | APACHEDEPLOYMENT

        ["/bin/sh", "/bin/sleep"]  | CENTOSDEPLOYMENT

        ["/bin/sh", "/bin/sleep"]  | FEDORADEPLOYMENT

        //["/usr/bin/tr", "/bin/chown", "/bin/egrep", "/bin/grep",
        // "/usr/local/bin/gosu", "/bin/hostname", "/docker-java-home/jre/bin/java",
        // "/usr/share/elasticsearch/bin/elasticsearch", "/sbin/ldconfig", "/bin/chown",
        // "/docker-entrypoint.sh", "/usr/bin/cut", "/usr/bin/id",
        // "/docker-java-home/jre/bin/java", "/usr/bin/dirname"]  | ELASTICDEPLOYMENT

        //["/usr/bin/id", "/usr/local/bin/docker-entrypoint.sh",
        // "/bin/chown", "/usr/local/bin/gosu", "/usr/local/bin/redis-server"]  | REDISDEPLOYMENT

        //["/bin/true", "/bin/chown", "/usr/local/bin/docker-entrypoint.sh",
        // "/bin/chown", "/usr/local/bin/gosu", "/usr/bin/mongod", "/usr/bin/numactl"] |  MONGODEPLOYMENT
   }
}
