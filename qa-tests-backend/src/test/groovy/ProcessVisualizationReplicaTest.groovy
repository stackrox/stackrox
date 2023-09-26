import objects.Deployment
import services.ProcessService
import util.Timer

import spock.lang.Tag
import spock.lang.Unroll

class ProcessVisualizationReplicaTest extends BaseSpecification {
    static final private Integer REPLICACOUNT = 4

    // Deployment names
    static final private String APACHEDEPLOYMENT = "apacheserverdeployment"
    static final private String MONGODEPLOYMENT = "mongodeployment"

    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                .setName (APACHEDEPLOYMENT)
                .setReplicas(REPLICACOUNT)
                .setImage ("quay.io/rhacs-eng/qa-multi-arch:apache-server")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (MONGODEPLOYMENT)
                .setReplicas(REPLICACOUNT)
                .setImage ("quay.io/rhacs-eng/qa-multi-arch:mongodb")
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
        for (Deployment deployment : DEPLOYMENTS) {
            Services.waitForSRDeletion(deployment)
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

    @Tag("BAT")
    @Tag("RUNTIME")
    @Unroll
    def "Verify process visualization with replicas on #depName"()  {
        when:
        "Get Process IDs running on deployment: #depName"
        String uid = DEPLOYMENTS.find { it.name == depName }.deploymentUid
        assert uid != null

        // processContainerMap contains a map of process path to a container id for each time that path was executed
        Map<String, List<String>> processContainerMap
        Set<String> receivedProcessPaths
        int retries = MAX_SLEEP_TIME / SLEEP_INCREMENT
        int delaySeconds = SLEEP_INCREMENT / 1000
        Timer t = new Timer(retries, delaySeconds)
        while (t.IsValid()) {
            receivedProcessPaths = ProcessService.getUniqueProcessPaths(uid)
            processContainerMap = ProcessService.getProcessContainerMap(uid, expectedFilePaths)

            // check that every container list has k*REPLICACOUNT containerId's
            def observedPathOnEachContainer = processContainerMap.every {
                k, v -> REPLICACOUNT == new HashSet<String>(v).size()
            }
            if (receivedProcessPaths.containsAll(expectedFilePaths) && observedPathOnEachContainer) {
                break
            }
            log.info "Didn't find all the expected processes, retrying..."
        }
        log.info "ProcessVisualizationTest: Dep: " + depName + " Processes: " + receivedProcessPaths

        processContainerMap = ProcessService.getProcessContainerMap(uid, expectedFilePaths)

        log.info processContainerMap.toString()

        processContainerMap.each { k, v ->
            // check that every path has k*REPLICACOUNT containerId's
            assert REPLICACOUNT == new HashSet<String>(v).size()
            // check that every container executed this path an equal number of times
            assert new HashSet<Integer>(getStringListCounts(v).values()).size() == 1
        }

        then:
        "Verify process in added : : #depName"
        assert receivedProcessPaths.containsAll(expectedFilePaths)

        where:
        "Data inputs are :"

        expectedFilePaths | depName

        ["/run.sh", "/usr/sbin/apache2",
          "/bin/chown", "/usr/bin/tail"] as Set | APACHEDEPLOYMENT

        ["/usr/local/bin/docker-entrypoint.sh",
         "/usr/bin/id",
         "/usr/bin/mongod", "/usr/bin/numactl"] as Set | MONGODEPLOYMENT
   }
}
