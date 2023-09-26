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
                .setImage ("quay.io/rhacs-eng/qa-multi-arch:nginx-1-14-alpine")
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
                .setName (REDISDEPLOYMENT)
                .setImage ("icr.io/ppc64le-oss/redis-ppc64le:v6.2.6-bv")
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

        ["/usr/bin/id", "/usr/bin/find", "/usr/local/bin/docker-entrypoint.sh",
         "/usr/local/bin/gosu"] as Set | REDISDEPLOYMENT

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
