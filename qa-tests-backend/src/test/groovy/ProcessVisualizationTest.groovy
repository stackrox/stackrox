import io.stackrox.proto.api.v1.SearchServiceOuterClass

import objects.Deployment
import services.DeploymentService
import services.ProcessService
import util.Timer

import org.junit.Assume
import spock.lang.Tag
import spock.lang.Unroll

@Tag("PZ")
class ProcessVisualizationTest extends BaseSpecification {
    // Deployment names
    static final private String NGINXDEPLOYMENT = "qanginx"
    static final private String STRUTSDEPLOYMENT = "qastruts"
    static final private String CENTOSDEPLOYMENT = "centosdeployment"
    static final private String FEDORADEPLOYMENT = "fedoradeployment"
    static final private String ELASTICDEPLOYMENT = "elasticdeployment"
    static final private String MONGODEPLOYMENT = "mongodeployment"
    static final private String ROX4751DEPLOYMENT = "rox4751deployment"
    static final private String ROX4979DEPLOYMENT = "rox4979deployment"
    // ldconfig process
    static final private String LDCONFIG = "/sbin/ldconfig"

    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                .setName (NGINXDEPLOYMENT)
                .setImage (TEST_IMAGE)
                .addLabel ( "app", "test" ),
            new Deployment()
                .setName (STRUTSDEPLOYMENT)
                .setImage("quay.io/rhacs-eng/qa-multi-arch:struts-app")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (CENTOSDEPLOYMENT)
                .setImage ("quay.io/centos/centos:stream9")
                .setCommand(["/bin/sh", "-c", "/bin/sleep 600"])
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (FEDORADEPLOYMENT)
                .setImage ("quay.io/rhacs-eng/qa-multi-arch:fedora-"+
                           "6fb84ba634fe68572a2ac99741062695db24b921d0aa72e61ee669902f88c187")
                .setCommand(["/bin/sh", "-c", "/bin/sleep 600"])
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (ELASTICDEPLOYMENT)
                .setImage ("quay.io/rhacs-eng/qa-multi-arch:elasticsearch-"+
                           "cdeb134689bb0318a773e03741f4414b3d1d0ee443b827d5954f957775db57eb")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (MONGODEPLOYMENT)
                .setImage ("quay.io/rhacs-eng/qa-multi-arch:mongodb")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (ROX4751DEPLOYMENT)
                .setImage ("quay.io/rhacs-eng/qa-multi-arch:ROX4751")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (ROX4979DEPLOYMENT)
                .setImage ("quay.io/rhacs-eng/qa-multi-arch:ROX4979")
                .addLabel ("app", "test" ),
     ]

    static final private MAX_SLEEP_TIME = 240000
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

    @Tag("BAT")
    @Tag("RUNTIME")
    def "Verify process visualization on kube-proxy"() {
        when:
        "Check if kube-proxy is running"
        def kubeProxyPods = orchestrator.getPodsByLabel("kube-system", ["component": "kube-proxy"])
        def podOwnerIsTracked = false
        for (pod in kubeProxyPods) {
            if (orchestrator.ownerIsTracked(pod.getMetadata())) {
                podOwnerIsTracked = true
                break
            }
        }
        // We only want to run this test if kube-proxy is running
        Assume.assumeFalse(kubeProxyPods == null || kubeProxyPods.size() == 0)

        then:
        "Ensure it has processes"
        def query = "Namespace:kube-system+Deployment:static-kube-proxy-pods"
        // if the kube-proxy pod owner is tracked (e.g. DaemonSet), then sensor does not track the individual pods as
        // `static-kube-proxy-pods` so we must assert against the DaemonSet `kube-proxy`
        if (podOwnerIsTracked) {
            query = "Namespace:kube-system+Deployment:kube-proxy"
        }
        def kubeProxyDeploymentsInRox = DeploymentService.listDeploymentsSearch(
                SearchServiceOuterClass.RawQuery.newBuilder().
                        setQuery(query).
                        build()
        )
        assert kubeProxyDeploymentsInRox.getDeploymentsList().size() == 1
        def kubeProxyDeploymentID = kubeProxyDeploymentsInRox.getDeployments(0).getId()
        def receivedProcessPaths = ProcessService.getUniqueProcessPaths(kubeProxyDeploymentID)
        log.info "Received processes: ${receivedProcessPaths}"
        // Avoid asserting on the specific process names since that might change across versions/distributions.
        // The goal is to make sure we pick up processes running in pods.
        assert receivedProcessPaths.size() > 0
    }

    @Tag("BAT")
    @Tag("RUNTIME")
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
            log.info "Didn't find all the expected processes, retrying..."
        }
        log.info "ProcessVisualizationTest: Dep: " + depName + " Processes: " + receivedProcessPaths

        then:
        "Verify process in added : : #depName"
        // ldconfig sometimes takes up to 10 minutes to be reported.
        // If it is the only process missing we ignore it in order to avoid waiting for 10 minutes.
        // It should be enough to assert on the other processes to validate this feature.
        // See: https://github.com/stackrox/stackrox/pull/12254
        if (!receivedProcessPaths.containsAll(expectedFilePaths) &&
                !receivedProcessPaths.contains(LDCONFIG)) {
            log.info("ldconfig took too long to be reported. Skipping it...")
            expectedFilePaths.remove(LDCONFIG)
        }
        assert receivedProcessPaths.containsAll(expectedFilePaths)

        where:
        "Data inputs are :"

        expectedFilePaths | depName

        ["/usr/sbin/nginx"] as Set | NGINXDEPLOYMENT

        ["/usr/bin/bash", "/usr/bin/uname",
         "/usr/local/tomcat/bin/catalina.sh",
         "/usr/bin/dirname"] as Set | STRUTSDEPLOYMENT

        ["/bin/sh", "/bin/sleep"] as Set | CENTOSDEPLOYMENT

        ["/bin/sh", "/bin/sleep"] as Set | FEDORADEPLOYMENT

        ["/usr/bin/tr", "/usr/bin/egrep", "/usr/bin/grep",
         "/usr/bin/hostname",
         "/usr/share/elasticsearch/bin/elasticsearch", "/sbin/ldconfig",
         "/usr/bin/cut",
         "/usr/bin/dirname"] as Set | ELASTICDEPLOYMENT

        ["/usr/local/bin/docker-entrypoint.sh",
         "/usr/bin/id",
         "/usr/bin/mongod", "/usr/bin/numactl"] as Set | MONGODEPLOYMENT

        ["/test/bin/exec.sh", "/usr/bin/date", "/usr/bin/sleep"] as Set | ROX4751DEPLOYMENT

        ["/qa/exec.sh", "/bin/sleep"] as Set | ROX4979DEPLOYMENT
    }

    @Tag("BAT")
    @Tag("RUNTIME")
    @Unroll
    def "Verify process paths, UIDs, and GIDs on #depName"()  {
        when:
        "Get Processes running on deployment: #depName"
        String uid = DEPLOYMENTS.find { it.name == depName }.deploymentUid
        assert uid != null

        Map<String,Set<Tuple2<Integer,Integer>>> processToUserAndGroupIds
        int retries = MAX_SLEEP_TIME / SLEEP_INCREMENT
        int delaySeconds = SLEEP_INCREMENT / 1000
        Timer t = new Timer(retries, delaySeconds)
        while (t.IsValid()) {
            processToUserAndGroupIds = ProcessService.getProcessUserAndGroupIds(uid)
            if (containsAllProcessInfo(processToUserAndGroupIds, expectedFilePathAndUIDs)) {
                break
            }
            log.info "Didn't find all the expected processes in " + depName +
                    ", retrying... " + processToUserAndGroupIds
        }
        log.info "ProcessVisualizationTest: Dep: " + depName +
                " Processes and UIDs: " + processToUserAndGroupIds

        then:
        "Verify process in added : : #depName"
        // ldconfig sometimes takes up to 10 minutes to be reported.
        // If it is the only process missing we ignore it in order to avoid waiting for 10 minutes.
        // It should be enough to assert on the other processes to validate this feature.
        // See: https://github.com/stackrox/stackrox/pull/12254
        if (!containsAllProcessInfo(processToUserAndGroupIds, expectedFilePathAndUIDs) &&
                !processToUserAndGroupIds.containsKey(LDCONFIG)) {
            log.info("ldconfig took too long to be reported. Skipping it...")
            expectedFilePathAndUIDs.remove(LDCONFIG)
        }
        assert containsAllProcessInfo(processToUserAndGroupIds, expectedFilePathAndUIDs)

        where:
        "Data inputs are :"

        expectedFilePathAndUIDs | depName

        [ "/usr/sbin/nginx":[[0, 0]],
        ] | NGINXDEPLOYMENT

        [ "/opt/java/openjdk/bin/java": [[0, 0]],
          "/usr/bin/uname":[[0, 0]],
          "/usr/local/tomcat/bin/catalina.sh":[[0, 0]],
          "/usr/bin/dirname":[[0, 0]],
        ] | STRUTSDEPLOYMENT

        [ "/bin/sh":[[0, 0]],
          "/bin/sleep":[[0, 0]],
        ] | CENTOSDEPLOYMENT

        [ "/bin/sh":[[0, 0]],
          "/bin/sleep":[[0, 0]],
        ] | FEDORADEPLOYMENT

        [ "/usr/bin/tr":[[1000, 1000]],
          "/usr/bin/egrep":[[1000, 1000]],
          "/usr/bin/grep":[[1000, 1000]],
          "/usr/share/elasticsearch/bin/elasticsearch":[[1000, 1000]],
          "/sbin/ldconfig":[[1000, 1000]],
          "/usr/bin/cut":[[1000, 1000]],
          "/bin/java":[[1000, 1000]],
          "/usr/bin/dirname":[[1000, 1000]],
        ] | ELASTICDEPLOYMENT

        [ "/test/bin/exec.sh":[[0, 0]],
          "/usr/bin/date":[[0, 0]],
          "/usr/bin/sleep":[[0, 0]],
        ] | ROX4751DEPLOYMENT

        [ "/qa/exec.sh":[[9001, 9000]],
          "/bin/sleep":[[9001, 9000]],
        ] | ROX4979DEPLOYMENT
    }

    @Tag("BAT")
    @Tag("RUNTIME")
    @Unroll
    def "Verify process arguments on #depName"() {
        when:
        "Get Process args running on deployment: #depName"
        String depId = DEPLOYMENTS.find { it.name == depName }.deploymentUid
        assert depId != null

        List<Tuple2<String, String>> processToArgs
        int retries = MAX_SLEEP_TIME / SLEEP_INCREMENT
        int delaySeconds = SLEEP_INCREMENT / 1000

        Timer t = new Timer(retries, delaySeconds)
        while (t.IsValid()) {
            processToArgs = ProcessService.getProcessesWithArgs(depId)
            if (processToArgs.containsAll(expectedProcessArgs)) {
                break
            }
            log.info "Didn't find all the expected processes, retrying..."
        }
        log.info "ProcessVisualizationTest: Dep: " + depName + " Processes: " + processToArgs

        then:
        "Verify process args for #depName"
        // ldconfig sometimes takes up to 10 minutes to be reported.
        // If it is the only process missing we ignore it in order to avoid waiting for 10 minutes.
        // It should be enough to assert on the other processes to validate this feature.
        // See: https://github.com/stackrox/stackrox/pull/12254
        if (!processToArgs.containsAll(expectedProcessArgs) &&
                processToArgs.find { it[0] == LDCONFIG } == null) {
            log.info("ldconfig took too long to be reported. Skipping it...")
            expectedProcessArgs.removeAll { it[0] == LDCONFIG }
        }
        assert processToArgs.containsAll(expectedProcessArgs)

        where:
        "Data inputs are:"

        expectedProcessArgs | depName

        [["/usr/sbin/nginx", "-g daemon off;"]] | NGINXDEPLOYMENT

        [
            ["/bin/sh", "-c /bin/sleep 600"],
            ["/bin/sleep", "--coreutils-prog-shebang=sleep /bin/sleep 600"],
        ] | CENTOSDEPLOYMENT

        [
            ["/bin/sleep", "600"],
            ["/bin/sh", "-c /bin/sleep 600"],
        ] | FEDORADEPLOYMENT

        // this is not a full selection of processes expected in the ELASTICDEPLOYMENT
        // but constitutes a decent range, with a variety of args, including no args,
        // or unusual characters.
        [
            ["/usr/bin/dirname", "/usr/share/elasticsearch/bin/elasticsearch"],
            ["/usr/bin/tr", "\\n  "],
            ["/usr/bin/grep", "project.name"],
            ["/usr/bin/cut", "-d. -f1"],
            ["/usr/bin/egrep", "/usr/bin/egrep -- (^-d |-d\$| -d |--daemonize\$|--daemonize )"],
            ["/usr/bin/hostname", ""],
            ["/usr/bin/grep", "-E -- (^-d |-d\$| -d |--daemonize\$|--daemonize )"],
            ["/usr/bin/grep", "^- /etc/elasticsearch/jvm.options"],
            ["/sbin/ldconfig", "-p"],
        ] | ELASTICDEPLOYMENT
    }

    // Returns true if received contains all the (path,UIDGIDSet) pairs found in expected
    private static Boolean containsAllProcessInfo(Map<String,Set<Tuple2<Integer,Integer>>> received,
                                                  Map<String,Set<Tuple2<Integer,Integer>>> expected) {
        if (received.size() < expected.size()) {
            return false
        }
        Boolean allFound = true
        expected.keySet().each {  String path ->
            if (!received.containsKey(path)) {
                allFound = false
                return
            }
            if (expected[path].size() != received[path].size()) {
                allFound = false
                return
            }
            if (expected[path].any { !received[path].contains(it) }) {
                allFound = false
                return
            }
        }
        return allFound
    }
}
