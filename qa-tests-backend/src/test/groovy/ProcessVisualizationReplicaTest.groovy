import static Services.roxDetectedDeployment

import services.ProcessService

import groups.BAT
import spock.lang.Unroll
import objects.Deployment
import org.junit.experimental.categories.Category

class ProcessVisualizationReplicaTest extends BaseSpecification {
    static final private Integer REPLICACOUNT = 10

    // Deployment names
    static final private String APACHEDEPLOYMENT = "apacheserverdeployment"
    static final private String MONGODEPLOYMENT = "mongodeployment"

    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                .setName (APACHEDEPLOYMENT)
                .setReplicas(REPLICACOUNT)
                .setImage ("apollo-dtr.rox.systems/legacy-apps/apache-server")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (MONGODEPLOYMENT)
                .setReplicas(REPLICACOUNT)
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

    def waitForSRDeletion(Deployment deployment) {
        // Wait until the deployment disappears from StackRox.
        long sleepTime = 0
        long sleepInterval = 1000
        boolean disappearedFromStackRox = false
        while (sleepTime < 60000) {
            if (!roxDetectedDeployment(deployment.getDeploymentUid())) {
                disappearedFromStackRox = true
                break
            }
            sleep(sleepInterval)
            sleepTime += sleepInterval
        }
        return disappearedFromStackRox
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
        for (Deployment deployment : DEPLOYMENTS) {
            waitForSRDeletion(deployment)
        }
    }

    // Given a list of strings, return map of string to occurence count
    def getStringListCounts(List<String> containerIds) {
        Map<String, Integer> counts = new HashMap<>()
        for (String k: containerIds) {
            counts.put(k, counts.getOrDefault(k, 0) + 1)
        }
        return counts
    }

    @Category(BAT)
    @Unroll
    def "Verify process visualization with replicas on #depName"()  {
        when:
        "Get Process IDs running on deployment: #depName"
        String uid = DEPLOYMENTS.find { it.name == depName }.deploymentUid
        assert uid != null

        // processContainerMap contains a map of process path to a container id for each time that path was executed
        def processContainerMap = ProcessService.getProcessContainerMap(uid)

        Set<String> receivedProcessPaths = ProcessService.getUniqueProcessPaths(uid)

        def sleepTime = 0L
        def observedPathOnEachContainer = false
        while ((!receivedProcessPaths.equals(expectedFilePaths) || !observedPathOnEachContainer)
               && sleepTime < MAX_SLEEP_TIME) {
            println "Didn't find all the expected processes, retrying..."
            sleep(SLEEP_INCREMENT)
            sleepTime += SLEEP_INCREMENT
            receivedProcessPaths = ProcessService.getUniqueProcessPaths(uid)
            processContainerMap = ProcessService.getProcessContainerMap(uid)

            // check that every container list has k*REPLICACOUNT containerId's
            observedPathOnEachContainer = processContainerMap.every {
                k, v -> REPLICACOUNT == new HashSet<String>(v).size()
            }
        }
        println "ProcessVisualizationTest: Dep: " + depName + " Processes: " + receivedProcessPaths

        processContainerMap = ProcessService.getProcessContainerMap(uid)

        processContainerMap.each { k, v ->
            // check that every path has k*REPLICACOUNT containerId's
            assert REPLICACOUNT == new HashSet<String>(v).size()
            // check that every container executed this path an equal number of times
            assert new HashSet<Integer>(getStringListCounts(v).values()).size() == 1
        }

        then:
        "Verify process in added : : #depName"
        assert receivedProcessPaths.equals(expectedFilePaths)

        where:
        "Data inputs are :"

        expectedFilePaths | depName

        ["/bin/mktemp", "/bin/mv", "/main.sh", "/usr/sbin/apache2", "/usr/sbin/apache2ctl",
          "/bin/chown", "/usr/bin/stat", "/bin/chmod", "/bin/mkdir"] as Set | APACHEDEPLOYMENT

        ["/bin/true", "/bin/chown", "/usr/local/bin/docker-entrypoint.sh",
         "/bin/rm", "/usr/bin/id", "/usr/bin/find",
         "/usr/local/bin/gosu", "/usr/bin/mongod", "/usr/bin/numactl"] as Set | MONGODEPLOYMENT
   }
}
