import io.stackrox.proto.api.v1.SearchServiceOuterClass

import objects.Deployment
import services.DeploymentService
import services.ProcessService
import util.Timer
import util.Env

import org.junit.Assume
import spock.lang.IgnoreIf
import spock.lang.Tag
import spock.lang.Unroll

class ProcessVisualizationTest extends BaseSpecification {
    // Deployment names
    static final private String NGINXDEPLOYMENT = "qanginx"
    static final private String STRUTSDEPLOYMENT = "qastruts"
    static final private String CENTOSDEPLOYMENT = "centosdeployment"
    static final private String FEDORADEPLOYMENT = "fedoradeployment"
    static final private String ELASTICDEPLOYMENT = "elasticdeployment"
    static final private String REDISDEPLOYMENT = "redisdeployment"
    static final private String MONGODEPLOYMENT = "mongodeployment"
    static final private String ROX4751DEPLOYMENT = "rox4751deployment"
    static final private String ROX4979DEPLOYMENT = "rox4979deployment"

    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                .setName (NGINXDEPLOYMENT)
                .setImage ("quay.io/rhacs-eng/qa:nginx-1.14-alpine")
                .addLabel ( "app", "test" ),
            new Deployment()
                .setName (STRUTSDEPLOYMENT)
                .setImage("quay.io/rhacs-eng/qa:struts-app")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (CENTOSDEPLOYMENT)
                .setImage ("quay.io/rhacs-eng/qa:centos-"+
                           "fc2476ccae2a5186313f2d1dadb4a969d6d2d4c6b23fa98b6c7b0a1faad67685")
                .setCommand(["/bin/sh", "-c", "/bin/sleep 600"])
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (FEDORADEPLOYMENT)
                .setImage ("quay.io/rhacs-eng/qa:fedora-"+
                           "6fb84ba634fe68572a2ac99741062695db24b921d0aa72e61ee669902f88c187")
                .setCommand(["/bin/sh", "-c", "/bin/sleep 600"])
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (ELASTICDEPLOYMENT)
                .setImage ("quay.io/rhacs-eng/qa:elasticsearch-"+
                           "cdeb134689bb0318a773e03741f4414b3d1d0ee443b827d5954f957775db57eb")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (REDISDEPLOYMENT)
                .setImage ("quay.io/rhacs-eng/qa:redis-"+
                           "96be1b5b6e4fe74dfe65b2b52a0fee254c443184b34fe448f3b3498a512db99e")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (MONGODEPLOYMENT)
                .setImage ("quay.io/rhacs-eng/qa:mongo-"+
                           "dec7f10108a87ff660a0d56cb71b0c5ae1f33cba796a33c88b50280fc0707116")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (ROX4751DEPLOYMENT)
                .setImage ("quay.io/rhacs-eng/qa:ROX4751")
                .addLabel ("app", "test" ),
            new Deployment()
                .setName (ROX4979DEPLOYMENT)
                .setImage ("quay.io/rhacs-eng/qa:ROX4979")
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

    @Tag("BAT")
    @Tag("RUNTIME")
    // TODO(ROX-16461): Fails under AKS
    @IgnoreIf({ Env.CI_JOB_NAME.contains("aks-qa-e2e") })
    def "Verify process visualization on kube-proxy"() {
        when:
        "Check if kube-proxy is running"
        def kubeProxyPods = orchestrator.getPodsByLabel("kube-system", ["component": "kube-proxy"])
        // We only want to run this test if kube-proxy is running
        Assume.assumeFalse(kubeProxyPods == null || kubeProxyPods.size() == 0)

        then:
        "Ensure it has processes"
        def kubeProxyDeploymentsInRox = DeploymentService.listDeploymentsSearch(
                SearchServiceOuterClass.RawQuery.newBuilder().
                        setQuery("Namespace:kube-system+Deployment:static-kube-proxy-pods").
                        build()
        )
        assert kubeProxyDeploymentsInRox.getDeploymentsList().size() == 1
        def kubeProxyDeploymentID = kubeProxyDeploymentsInRox.getDeployments(0).getId()
        def receivedProcessPaths = ProcessService.getUniqueProcessPaths(kubeProxyDeploymentID)
        log.info "Received processes: ${receivedProcessPaths}"
        // Avoid asserting on the specific process names since that might change across versions/distributions.
        // The goal is to make sure we pick up processes from static pods.
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
        assert containsAllProcessInfo(processToUserAndGroupIds, expectedFilePathAndUIDs)

            where:
        "Data inputs are :"

        expectedFilePathAndUIDs | depName

        [ "/usr/sbin/nginx":[[0, 0]],
        ] | NGINXDEPLOYMENT

        [ "/docker-java-home/jre/bin/java": [[0, 0]],
          "/usr/bin/tty":[[0, 0]],
          "/bin/uname":[[0, 0]],
          "/usr/local/tomcat/bin/catalina.sh":[[0, 0]],
          "/usr/bin/dirname":[[0, 0]],
        ] | STRUTSDEPLOYMENT

        [ "/bin/sh":[[0, 0]],
          "/bin/sleep":[[0, 0]],
        ] | CENTOSDEPLOYMENT

        [ "/bin/sh":[[0, 0]],
          "/bin/sleep":[[0, 0]],
        ] | FEDORADEPLOYMENT

        [ "/usr/bin/tr":[[101, 101]],
          "/bin/chown":[[0, 0]],
          "/bin/egrep":[[101, 101]],
          "/bin/grep":[[101, 101]],
          "/usr/local/bin/gosu":[[0, 0]],
          "/bin/hostname":[[101, 101]],
          "/usr/share/elasticsearch/bin/elasticsearch":[[101, 101]],
          "/sbin/ldconfig":[[101, 101]],
          "/docker-entrypoint.sh":[[0, 0]],
          "/usr/bin/cut":[[101, 101]],
          "/usr/bin/id":[[0, 0]],
          "/docker-java-home/jre/bin/java":[[101, 101]],
          "/usr/bin/dirname":[[101, 101]],
        ] | ELASTICDEPLOYMENT

        [ "/test/bin/exec.sh":[[0, 0]],
          "/usr/bin/date":[[0, 0]],
          "/usr/bin/sleep":[[0, 0]],
        ] | ROX4751DEPLOYMENT

        [ "/qa/exec.sh":[[9001, 9000]],
          "/bin/sleep":[[9001, 9000]],
        ] | ROX4979DEPLOYMENT

        /*
        // Enable as part of ROX-5417 (process deduplication should include process UIDs)
        [ "/usr/bin/id":[[0,0], [999,999]],
          "/usr/bin/find":[[0,0]],
          "/usr/local/bin/docker-entrypoint.sh":[[0,0], [999,999]],
          "/usr/local/bin/gosu":[[0,0]],
          "/usr/local/bin/redis-server":[[999,999]],
         ] | REDISDEPLOYMENT

        // On machines with NUMA arch, mongo deployment will also execute path `/bin/true`
        [ "/bin/chown":[[0,0]],
          "/usr/local/bin/docker-entrypoint.sh": [[0,0], [999,999]],
          "/bin/rm":[[999,999]],
          "/usr/bin/id":[[0,0], [999,999]],
          "/usr/bin/find":[[0,0]],
          "/usr/local/bin/gosu":[[0,0]],
          "/usr/bin/mongod":[[999,999]],
          "/usr/bin/numactl":[[999,999]],
        ] | MONGODEPLOYMENT
        */
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
        assert processToArgs.containsAll(expectedProcessArgs)

        where:
        "Data inputs are:"

        expectedProcessArgs | depName

        [["/usr/sbin/nginx", "-g daemon off;"]] | NGINXDEPLOYMENT

        [
            ["/bin/sh", "-c /bin/sleep 600"],
            ["/bin/sleep", "600"],
        ] | CENTOSDEPLOYMENT

        [
            ["/bin/sleep", "--coreutils-prog-shebang=sleep /bin/sleep 600"],
            ["/bin/sh", "-c /bin/sleep 600"],
        ] | FEDORADEPLOYMENT

        // this is not a full selection of processes expected in the ELASTICDEPLOYMENT
        // but constitutes a decent range, with a variety of args, including no args,
        // or unusual characters.
        [
            ["/usr/bin/dirname", "/usr/share/elasticsearch/bin/elasticsearch"],
            ["/usr/bin/tr", "\\n  "],
            ["/bin/grep", "project.name"],
            ["/usr/bin/cut", "-d. -f1"],
            ["/usr/local/bin/gosu", "elasticsearch elasticsearch"],
            ["/bin/egrep", "/bin/egrep -- (^-d |-d\$| -d |--daemonize\$|--daemonize )"],
            ["/bin/hostname", ""],
            ["/docker-entrypoint.sh", "/docker-entrypoint.sh elasticsearch"],
            ["/bin/grep", "-E -- (^-d |-d\$| -d |--daemonize\$|--daemonize )"],
            ["/bin/grep", "^- /etc/elasticsearch/jvm.options"],
            ["/bin/chown", "-R elasticsearch:elasticsearch /usr/share/elasticsearch/data"],
            ["/bin/chown", "-R elasticsearch:elasticsearch /usr/share/elasticsearch/logs"],
            ["/sbin/ldconfig", "-p"],
            ["/usr/bin/id", "-u"],
        ] | ELASTICDEPLOYMENT
    }

    // Returns true if received contains all the (path,UIDGIDSet) pairs found in expected
    private static Boolean containsAllProcessInfo(Map<String,Set<Tuple2<Integer,Integer>>> received,
                                                  Map<String,Set<Tuple2<Integer,Integer>>> expected) {
        if (received.size() < expected.size()) {
            return false
        }
        for ( String path : expected.keySet() ) {
            if (!received.containsKey(path)) {
                return false
            }
            if (expected[path].size() != received[path].size()) {
                return false
            }
            for ( Tuple2<Integer, Integer> ids : expected[path]) {
                if (!received[path].contains(ids)) {
                    return false
                }
            }
        }
        return true
    }
}
