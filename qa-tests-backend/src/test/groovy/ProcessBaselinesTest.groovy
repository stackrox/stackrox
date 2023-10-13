import static Services.waitForSuspiciousProcessInRiskIndicators
import static Services.waitForViolation
import static util.Helpers.evaluateWithRetry

import java.util.concurrent.TimeUnit

import org.apache.commons.lang3.StringUtils
import org.junit.Rule
import org.junit.rules.Timeout

import io.stackrox.proto.api.v1.AlertServiceOuterClass
import io.stackrox.proto.storage.AlertOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import io.stackrox.proto.storage.ProcessBaselineOuterClass
import io.stackrox.proto.storage.RiskOuterClass

import objects.Deployment
import services.AlertService
import services.ClusterService
import services.PolicyService
import services.ProcessBaselineService

import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Unroll

@Tag("Parallel")
@Tag("PZ")
class ProcessBaselinesTest extends BaseSpecification {
    @Shared
    private String clusterId

    static final private String TEST_NAMESPACE = "qa-process-baselines"

    static final private String DEPLOYMENTNGINX = "pb-deploymentnginx"
    static final private String DEPLOYMENTNGINX_RESOLVE_VIOLATION = "pb-deploymentnginx-violation-resolve"
    static final private String DEPLOYMENTNGINX_RESOLVE_AND_BASELINE_VIOLATION =
            "pb-deploymentnginx-violation-resolve-baseline"
    static final private String DEPLOYMENTNGINX_LOCK = "pb-deploymentnginx-lock"
    static final private String DEPLOYMENTNGINX_DELETE = "pb-deploymentnginx-delete"
    static final private String DEPLOYMENTNGINX_DELETE_API = "pb-deploymentnginx-delete-api"
    static final private String DEPLOYMENTNGINX_POST_DELETE_API = "pb-deploymentnginx-post-delete-api"
    static final private String DEPLOYMENTNGINX_REMOVEPROCESS = "pb-deploymentnginx-removeprocess"

    static final private List<Deployment> DEPLOYMENTS =
        [
             new Deployment()
                     .setName(DEPLOYMENTNGINX)
                     .setNamespace(TEST_NAMESPACE)
                     .setImage(TEST_IMAGE)
                     .addPort(22, "TCP")
                     .addAnnotation("test", "annotation")
                     .setEnv(["CLUSTER_NAME": "main"])
                     .addLabel("app", "test"),
             new Deployment()
                     .setName(DEPLOYMENTNGINX_RESOLVE_VIOLATION)
                     .setNamespace(TEST_NAMESPACE)
                     .setImage(TEST_IMAGE)
                     .addPort(22, "TCP")
                     .addAnnotation("test", "annotation")
                     .setEnv(["CLUSTER_NAME": "main"])
                     .addLabel("app", "test"),
             new Deployment()
                     .setName(DEPLOYMENTNGINX_RESOLVE_AND_BASELINE_VIOLATION)
                     .setNamespace(TEST_NAMESPACE)
                     .setImage(TEST_IMAGE)
                     .addPort(22, "TCP")
                     .addAnnotation("test", "annotation")
                     .setEnv(["CLUSTER_NAME": "main"])
                     .addLabel("app", "test"),
             new Deployment()
                     .setName(DEPLOYMENTNGINX_LOCK)
                     .setNamespace(TEST_NAMESPACE)
                     .setImage(TEST_IMAGE)
                     .addPort(22, "TCP")
                     .addAnnotation("test", "annotation")
                     .setEnv(["CLUSTER_NAME": "main"])
                     .addLabel("app", "test"),
             new Deployment()
                     .setName(DEPLOYMENTNGINX_DELETE)
                     .setNamespace(TEST_NAMESPACE)
                     .setImage(TEST_IMAGE)
                     .addPort(22, "TCP")
                     .addAnnotation("test", "annotation")
                     .setEnv(["CLUSTER_NAME": "main"])
                     .addLabel("app", "test"),
             new Deployment()
                     .setName(DEPLOYMENTNGINX_DELETE_API)
                     .setNamespace(TEST_NAMESPACE)
                     .setImage(TEST_IMAGE)
                     .addPort(22, "TCP")
                     .addAnnotation("test", "annotation")
                     .setEnv(["CLUSTER_NAME": "main"])
                     .addLabel("app", "test"),
             new Deployment()
                     .setName(DEPLOYMENTNGINX_POST_DELETE_API)
                     .setNamespace(TEST_NAMESPACE)
                     .setImage(TEST_IMAGE)
                     .addPort(22, "TCP")
                     .addAnnotation("test", "annotation")
                     .setEnv(["CLUSTER_NAME": "main"])
                     .addLabel("app", "test"),
             new Deployment()
                     .setName(DEPLOYMENTNGINX_REMOVEPROCESS)
                     .setNamespace(TEST_NAMESPACE)
                     .setImage(TEST_IMAGE)
                     .addPort(22, "TCP")
                     .addAnnotation("test", "annotation")
                     .setEnv(["CLUSTER_NAME": "main"])
                     .addLabel("app", "test"),
            ]

    static final private Integer BASELINE_WAIT_TIME = 100
    static final private Integer RISK_WAIT_TIME = 240

    // Override the global JUnit test timeout to cover a test instance taking
    // LONGEST_TEST over three test tries and the appprox. 6
    // minutes it can take to gather debug when the first test run fails plus
    // some padding.
    @Rule
    @SuppressWarnings(["JUnitPublicProperty"])
    Timeout globalTimeout = new Timeout(3*(BASELINE_WAIT_TIME + RISK_WAIT_TIME) + 300 + 120, TimeUnit.SECONDS)

    @Shared
    private Policy unauthorizedProcessExecution

    def setupSpec() {
        clusterId = ClusterService.getClusterId()

        unauthorizedProcessExecution = PolicyService.clonePolicyAndScopeByNamespace(
            "Unauthorized Process Execution",
            TEST_NAMESPACE
        )
        assert unauthorizedProcessExecution
    }

    def cleanupSpec() {
        PolicyService.deletePolicy(unauthorizedProcessExecution.getId())
        orchestrator.deleteNamespace(TEST_NAMESPACE)
    }

    @Unroll
    @Tag("BAT")
    def "Verify processes risk indicators for the given key after lock on #deploymentName"() {
        when:
        "exec into the container and run a process and wait for lock to kick in"
        def deployment = DEPLOYMENTS.find { it.name == deploymentName }
        assert deployment != null
        orchestrator.createDeployment(deployment)
        assert Services.waitForDeployment(deployment)
        String deploymentId = deployment.getDeploymentUid()
        assert deploymentId != null
        orchestrator.execInContainer(deployment, "ls")

        String containerName = deployment.getName()

        // wait for baseline to come out of observation
        ProcessBaselineOuterClass.ProcessBaseline baseline = evaluateWithRetry(10, 10) {
            def tmpBaseline = ProcessBaselineService.getProcessBaseline(clusterId, deployment, containerName)
            def now = System.currentTimeSeconds()
            if (tmpBaseline.getStackRoxLockedTimestamp().getSeconds() > now) {
                throw new RuntimeException(
                    "Baseline ${deployment} is still in observation. Baseline is ${tmpBaseline}."
                )
            }
            return tmpBaseline
        }
        assert baseline
        assert ((baseline.key.deploymentId.equalsIgnoreCase(deploymentId)) &&
                    (baseline.key.containerName.equalsIgnoreCase(containerName)))
        assert baseline.elementsList.find { it.element.processName == processName } != null

        log.info "Baseline Before after observation: ${baseline}"

        // sleep 10 seconds to allow for propagation to sensor
        sleep 10000
        orchestrator.execInContainer(deployment, "pwd")

        then:
        "verify for suspicious process in risk indicator"
        RiskOuterClass.Risk.Result result = waitForSuspiciousProcessInRiskIndicators(deploymentId, 120)
        assert (result != null)
        // Check that ls doesn't exist as a risky process
        RiskOuterClass.Risk.Result.Factor lsFactor =  result.factorsList.find { it.message.contains("ls") }
        assert lsFactor == null
        // Check that pwd is a risky process
        RiskOuterClass.Risk.Result.Factor pwdFactor =  result.factorsList.find { it.message.contains("pwd") }
        assert pwdFactor != null

        cleanup:
        "Remove deployment"
        log.info "Cleaning up deployment: ${deployment}"
        orchestrator.deleteAndWaitForDeploymentDeletion(deployment)

        where:
        "Data inputs are :"
        deploymentName                      |   processName
        DEPLOYMENTNGINX_LOCK                |   "/usr/sbin/nginx"
    }

    /* TODO(ROX-3108)
    @Unroll
    @Tag("BAT")
    def "Verify baseline processes for the given key before and after locking "() {
        when:
        def deployment = DEPLOYMENTS.find { it.name == deploymentName }
        assert deployment != null
        String deploymentId = deployment.getDeploymentUid()
        // Currently, we always create a deployment where the container name is the same
        // as the deployment name
        String containerName = deployment.getName()
        "get process baselines is called for a key"
        ProcessBaselineOuterClass.ProcessBaseline baseline = ProcessBaselineService.
                getProcessBaseline(clusterId, deployment, containerName)

        assert (baseline != null)

        then:
        "Verify  baseline processes for a given key before and after calling lock baselines"
        assert ((baseline.key.deploymentId.equalsIgnoreCase(deploymentId)) &&
                    (baseline.key.containerName.equalsIgnoreCase(containerName)))
        assert  baseline.getElements(0).element.processName.contains(processName)

        // lock the baseline with the key of the container just deployed
        List<ProcessBaselineOuterClass.ProcessBaseline> lockProcessBaselines = ProcessBaselineService.
                lockProcessBaselines(clusterId, deployment, containerName, true)
        assert  lockProcessBaselines.size() == 1
        assert  lockProcessBaselines.get(0).getElementsList().
            find { it.element.processName.equalsIgnoreCase(processName) } != null

        where:
        "Data inputs are :"
        deploymentName     | processName

        DEPLOYMENTNGINX    | "/usr/sbin/nginx"
    }
    */

    @Unroll
    @Tag("BAT")
    @Tag("COMPATIBILITY")
    def "Verify baseline process violation after resolve baseline on #deploymentName"() {
               /*
                    a)Lock the processes in the baseline for the key
                    b)exec into the container and run a process
                    c)verify violation alert for Unauthorized Process Execution
                    d)
                        test case :choose to only resolve violation
                            exec into the container and run the  process again and verify violation alert
                        test case : choose to both resolve and add to the baseline
                            exec into the container and run the  process again and verify no violation alert
               */
        when:
        "exec into the container after locking baseline and create a baseline violation"
        def deployment = DEPLOYMENTS.find { it.name == deploymentName }
        assert deployment != null
        orchestrator.createDeployment(deployment)
        String deploymentId = deployment.getDeploymentUid()
        assert deploymentId != null

        String containerName = deployment.getName()
        // Need to make sure the processes show up before we lock.
        def baseline = evaluateWithRetry(10, 10) {
            def tmpBaseline = ProcessBaselineService.
                 getProcessBaseline(clusterId, deployment, containerName)
            if (tmpBaseline.elementsList.size() == 0) {
                throw new RuntimeException(
                    "No processes in baseline for deployment ${deploymentId} yet. Baseline is ${tmpBaseline}"
                )
            }
            return tmpBaseline
        }

        assert (baseline != null)
        log.info "Baseline Before locking: ${baseline}"
        assert ((baseline.key.deploymentId.equalsIgnoreCase(deploymentId)) &&
                 (baseline.key.containerName.equalsIgnoreCase(containerName)))
        assert baseline.elementsList.find { it.element.processName == processName } != null

        List<ProcessBaselineOuterClass.ProcessBaseline> lockProcessBaselines = ProcessBaselineService.
                 lockProcessBaselines(clusterId, deployment, containerName, true)
        assert (!StringUtils.isEmpty(lockProcessBaselines.get(0).getElements(0).getElement().processName))

        // sleep 5 seconds to allow for propagation to sensor
        sleep 5000
        orchestrator.execInContainer(deployment, "pwd")

        log.info "Locked Process Baseline after pwd: ${lockProcessBaselines}"

        // check for process baseline violation
        assert waitForViolation(containerName, unauthorizedProcessExecution.getName(), RISK_WAIT_TIME)
        List<AlertOuterClass.ListAlert> alertList = AlertService.getViolations(AlertServiceOuterClass.ListAlertsRequest
                 .newBuilder().build())
        String alertId
        for (AlertOuterClass.ListAlert alert : alertList) {
            if (alert.getPolicy().name.equalsIgnoreCase(unauthorizedProcessExecution.getName()) &&
                     alert.deployment.id.equalsIgnoreCase(deploymentId)) {
                alertId = alert.id
                AlertService.resolveAlert(alertId, addToBaseline)
                // again, allow the new baseline that contains pwd to propagate
                sleep 5000
            }
         }
        orchestrator.execInContainer(deployment, "pwd")
        if (addToBaseline) {
            waitForViolation(containerName, unauthorizedProcessExecution.getName(), 15)
        }
        else {
            assert waitForViolation(containerName, unauthorizedProcessExecution.getName(), 15)
        }
        then:
        "Verify for violation or no violation after resolve/resolve and baseline"
        List<AlertOuterClass.ListAlert> alertListAnother = AlertService
                 .getViolations(AlertServiceOuterClass.ListAlertsRequest
                 .newBuilder().build())

        int numAlertsAfterResolve = 0
        for (AlertOuterClass.ListAlert alert : alertListAnother) {
            if (alert.getPolicy().name.equalsIgnoreCase(unauthorizedProcessExecution.getName())
                     && alert.deployment.id.equalsIgnoreCase(deploymentId)) {
                numAlertsAfterResolve++
                AlertService.resolveAlert(alert.id, false)
                break
             }
         }
        assert (numAlertsAfterResolve  == expectedViolationsCount)

        cleanup:
        "Remove deployment"
        orchestrator.deleteAndWaitForDeploymentDeletion(deployment)

        where:
        "Data inputs are :"
        deploymentName                                 | processName       | addToBaseline | expectedViolationsCount

        DEPLOYMENTNGINX_RESOLVE_VIOLATION              | "/usr/sbin/nginx" | false         | 1

        DEPLOYMENTNGINX_RESOLVE_AND_BASELINE_VIOLATION | "/usr/sbin/nginx" | true          | 0
    }

    @Tag("BAT")
    def "Verify baselines are deleted when their deployment is deleted"() {
        /*
                a)get all baselines
                b)verify baselines exist for a deployment
                c)delete the deployment
                d)get all baselines
                e)verify all baselines for the deployment have been deleted
        */
        when:
        "a deployment is deleted"
        // Get all baselines for our deployment and assert they exist
        def deployment = DEPLOYMENTS.find { it.name == DEPLOYMENTNGINX_DELETE }
        assert deployment != null
        orchestrator.createDeployment(deployment)
        String containerName = deployment.getName()
        def baselinesCreated = ProcessBaselineService.
                waitForDeploymentBaselinesCreated(clusterId, deployment, containerName)
        assert(baselinesCreated)

        // Delete the deployment
        orchestrator.deleteDeployment(deployment)
        Services.waitForSRDeletion(deployment)

        then:
        "Verify that all baselines with that deployment ID have been deleted"
        def baselinesDeleted = ProcessBaselineService.
                waitForDeploymentBaselinesDeleted(clusterId, deployment, containerName)
        assert(baselinesDeleted)

        cleanup:
        "Remove deployment"
        orchestrator.deleteAndWaitForDeploymentDeletion(deployment)
    }

    @Unroll
    @Tag("BAT")
    def "Verify removed baseline process not getting added back to baseline after rerun on #deploymentName"() {
        /*
                1.run a process and verify if it exists in the baseline
                2.remove the process
                3.rerun the process to verify it it does not get added to the baseline
         */
        when:
        "an added process is removed and baseline is locked and the process is run"
        def deployment = DEPLOYMENTS.find { it.name == deploymentName }
        assert deployment != null
        orchestrator.createDeployment(deployment)
        def deploymentId = deployment.deploymentUid
        assert deploymentId != null

        def containerName = deploymentName
        def namespace = deployment.getNamespace()

        // Wait for baseline to be created
        def initialBaseline = ProcessBaselineService.
                getProcessBaseline(clusterId, deployment, containerName)
        assert (initialBaseline != null)

        // Add the process to the baseline
        ProcessBaselineOuterClass.ProcessBaselineKey [] keys = [
                ProcessBaselineOuterClass.ProcessBaselineKey
                .newBuilder().setContainerName(containerName)
                .setDeploymentId(deploymentId).setClusterId(clusterId).setNamespace(namespace).build(),
        ]
        String [] toBeAddedProcesses = ["/bin/pwd"]
        String [] toBeRemovedProcesses = []
        List<ProcessBaselineOuterClass.ProcessBaseline> updatedList = ProcessBaselineService
                .updateProcessBaselines(keys, toBeAddedProcesses, toBeRemovedProcesses)
        assert ( updatedList!= null)
        ProcessBaselineOuterClass.ProcessBaseline baseline = ProcessBaselineService.
                getProcessBaseline(clusterId, deployment, containerName)
        List<ProcessBaselineOuterClass.BaselineElement> elements = baseline.elementsList
        ProcessBaselineOuterClass.BaselineElement element = elements.find {
            it.element.processName.contains("/bin/pwd") }
        assert ( element != null)

        // Remove the process from the baseline
        toBeAddedProcesses = []
        toBeRemovedProcesses = ["/bin/pwd"]
        List<ProcessBaselineOuterClass.ProcessBaseline> updatedListAfterRemoveProcess = ProcessBaselineService
                .updateProcessBaselines(keys, toBeAddedProcesses, toBeRemovedProcesses)
        assert ( updatedListAfterRemoveProcess!= null)

        orchestrator.execInContainer(deployment, "pwd")
        then:
        "verify process is not added to the baseline"
        ProcessBaselineOuterClass.ProcessBaseline baselineAfterReRun = ProcessBaselineService.
                getProcessBaseline(clusterId, deployment, containerName)
        assert  ( baselineAfterReRun.elementsList.find { it.element.processName.contains("pwd") } == null)

        cleanup:
        "Remove deployment"
        orchestrator.deleteAndWaitForDeploymentDeletion(deployment)

        where:
        deploymentName                                   | processName
        DEPLOYMENTNGINX_REMOVEPROCESS           |   "nginx"
    }

    @Tag("BAT")
    def "Delete process baselines via API"() {
        given:
        "a baseline is deleted"
        // Get all baselines for our deployment and assert they exist
        def deployment = DEPLOYMENTS.find { it.name == DEPLOYMENTNGINX_DELETE_API }
        assert deployment != null
        orchestrator.createDeployment(deployment)
        String containerName = deployment.getName()
        def baselinesCreated = ProcessBaselineService.
                waitForDeploymentBaselinesCreated(clusterId, deployment, containerName)
        assert(baselinesCreated)

        when:
        "delete the baselines"
        log.info "ID: ${deployment.getDeploymentUid()}"
        ProcessBaselineService.deleteProcessBaselines("Deployment Id:${deployment.getDeploymentUid()}")

        then:
        "Verify that all baselines with that deployment ID have been deleted (i.e. the baseline contents cleared)"
        ProcessBaselineOuterClass.ProcessBaseline baselineAfterDelete = ProcessBaselineService.
                            getProcessBaseline(clusterId, deployment, containerName)
        // Baseline should still exist but have no elements associated.  Essentially cleared out.
        assert  ( baselineAfterDelete.elementsList == [] )

        cleanup:
        "Remove deployment"
        orchestrator.deleteAndWaitForDeploymentDeletion(deployment)
    }

    @Unroll
    @Tag("BAT")
    def "Processes come in after baseline deleted by API for #deploymentName"() {
        when:
        def deployment = DEPLOYMENTS.find { it.name == deploymentName }
        assert deployment != null
        orchestrator.createDeployment(deployment)
        String deploymentId = deployment.getDeploymentUid()
        assert deploymentId != null
        orchestrator.execInContainer(deployment, "ls")

        String containerName = deployment.getName()

        // Wait on the process to be baseline to come out of observation
        ProcessBaselineOuterClass.ProcessBaseline baseline = evaluateWithRetry(10, 10) {
            def tmpBaseline = ProcessBaselineService.getProcessBaseline(clusterId, deployment, containerName)
            def now = System.currentTimeSeconds()
            if (tmpBaseline.getStackRoxLockedTimestamp().getSeconds() > now) {
                throw new RuntimeException(
                    "Baseline ${deployment} is still in observation. Baseline is ${tmpBaseline}."
                )
            }
            return tmpBaseline
        }

        assert (baseline != null)
        assert ((baseline.key.deploymentId.equalsIgnoreCase(deploymentId)) &&
                    (baseline.key.containerName.equalsIgnoreCase(containerName)))
        assert baseline.elementsList.find { it.element.processName == processName } != null

        // Delete the baseline
        ProcessBaselineService.deleteProcessBaselines("Deployment Id:${deploymentId}")

        // Retrieve the cleared out baseline
        ProcessBaselineOuterClass.ProcessBaseline baselineAfterDelete = ProcessBaselineService.
                            getProcessBaseline(clusterId, deployment, containerName)
        // Baseline should still exist but have no elements associated.  Essentially cleared out.
        assert  ( baselineAfterDelete.elementsList == [] )

        // Give the baseline time to come back out of observation
        baselineAfterDelete = evaluateWithRetry(10, 10) {
            def tmpBaseline = ProcessBaselineService.getProcessBaseline(clusterId, deployment, containerName)
            def now = System.currentTimeSeconds()
            if (tmpBaseline.getStackRoxLockedTimestamp().getSeconds() > now) {
                throw new RuntimeException(
                    "Baseline ${deployment} is still in observation. Baseline is ${tmpBaseline}."
                )
            }
            return tmpBaseline
        }
        assert baselineAfterDelete

        log.info "Process Baseline before pwd: ${baselineAfterDelete}"

        // sleep 10 seconds to allow for propagation to sensor
        sleep 10000
        orchestrator.execInContainer(deployment, "pwd")

        then:
        "verify for suspicious process in risk indicator"
        RiskOuterClass.Risk.Result result = waitForSuspiciousProcessInRiskIndicators(deploymentId, RISK_WAIT_TIME)
        assert (result != null)
        // Check that pwd is a risky process
        RiskOuterClass.Risk.Result.Factor pwdFactor =  result.factorsList.find { it.message.contains("pwd") }
        assert pwdFactor != null

        cleanup:
        "Remove deployment"
        log.info "Cleaning up deployment: ${deployment}"
        orchestrator.deleteAndWaitForDeploymentDeletion(deployment)

        where:
        "Data inputs are :"
        deploymentName                          |   processName
        DEPLOYMENTNGINX_POST_DELETE_API         |   "/usr/sbin/nginx"
    }
}
