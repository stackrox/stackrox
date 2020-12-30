import io.stackrox.proto.api.v1.DeploymentServiceOuterClass.ListDeploymentsWithProcessInfoResponse.DeploymentWithProcessInfo
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.storage.ProcessBaselineOuterClass
import objects.Deployment
import orchestratormanager.OrchestratorTypes
import org.junit.Assume
import services.ClusterService
import services.DeploymentService
import services.ProcessService
import services.ProcessBaselineService
import spock.lang.Shared
import spock.lang.Stepwise
import util.Env
import util.Timer

// RiskTest - Test coverage for functionality used on the Risk page and not covered elsewhere.
// i.e.
// - ListDeploymentsWithProcessInfo
// - CountDeployments
// - GetGroupedProcessByDeploymentAndContainer

@Stepwise // tests are ordered and dependent
class RiskTest extends BaseSpecification {
    @Shared
    private String clusterId

    // This test relies on two initially equivalent deployments. One of which executes a process after
    //  soft lock has taken affect.
    @Shared
    private Deployment deploymentWithRisk
    @Shared
    private Deployment deploymentWithoutRisk

    // DeploymentWithProcessInfo for both deployments to pass between tests for comparison.
    @Shared
    private List<DeploymentWithProcessInfo> whenEquivalent
    @Shared
    private List<DeploymentWithProcessInfo> whenOneHasRisk

    static final private int RETRIES = 24
    static final private int RETRY_DELAY = 5
    static final private List<Deployment> DEPLOYMENTS = []
    static final private String TEST_NAMESPACE = "qa-risk"

    def setupSpec() {
        clusterId = ClusterService.getClusterId()

        for (int i = 0; i < 2; i++) {
            DEPLOYMENTS.push(
                    new Deployment()
                            .setName("risk-deployment-${i}")
                            .setNamespace(TEST_NAMESPACE)
                            .setImage("busybox:1.31")
                            .setCommand(["/bin/sh", "-c",])
                            .setArgs(["sleep 36000",])
            )
        }
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment d : DEPLOYMENTS) {
            assert Services.waitForDeployment(d)
        }
        deploymentWithRisk = DEPLOYMENTS[0]
        deploymentWithoutRisk = DEPLOYMENTS[1]
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
    }

    def "Risk is the same for equivalent deployments"() {
        when:
        "waiting for SR to get to an initial priority and process baseline for each deployment"
        def t = new Timer(RETRIES, RETRY_DELAY)
        while (t.IsValid()) {
            def response = DeploymentService.listDeploymentsWithProcessInfo(
                    RawQuery.newBuilder().setQuery("Namespace:" + TEST_NAMESPACE).build()
            )
            if (!response || response.deploymentsList.size() < DEPLOYMENTS.size()) {
                println "not yet ready to test - no deployments found"
                continue
            }
            if (response.deploymentsList.get(0).whitelistStatusesList.size() == 0 ||
                    response.deploymentsList.get(1).whitelistStatusesList.size() == 0) {
                println "not yet ready to test - container summary status are not set"
                continue
            }

            def processesFound = true
            for (int i = 0; i < DEPLOYMENTS.size(); i++) {
                def processes = ProcessBaselineService.getProcessBaseline(clusterId, DEPLOYMENTS[i], null, 0)
                if (!processes || processes.elementsList.size() == 0) {
                    println "not yet ready to test - processes not found for ${DEPLOYMENTS[i].name}"
                    processesFound = false
                }

                if (processes) {
                    processes.elementsList.forEach { element ->
                        println "SR found ${element.element.processName} for ${DEPLOYMENTS[i].name}"
                    }
                }
            }

            if (!processesFound) {
                println "not yet ready to test - processes not found"
                continue
            }

            println "ready to test"
            whenEquivalent = response.deploymentsList
            break
        }

        assert whenEquivalent, "SR found the deployments, containers and processes required"

        def one = whenEquivalent.get(0)
        def two = whenEquivalent.get(1)
        println debugPriorityAndState(one)
        println debugPriorityAndState(two)

        then:
        "should be at equivalent priority"
        one.deployment.priority == two.deployment.priority

        and:
        "not anomalous"
        !one.whitelistStatusesList.get(0).anomalousProcessesExecuted
        !two.whitelistStatusesList.get(0).anomalousProcessesExecuted
    }

    def "Deployment count == 2"() {
        expect:
        DeploymentService.getDeploymentCount(
                RawQuery.newBuilder().setQuery("Namespace:" + TEST_NAMESPACE).build()
        ) == DEPLOYMENTS.size()
    }

    def "Processes grouped by deployment (GetGroupedProcessByDeploymentAndContainer)"() {
        given:
        // Skip for OpenShift, it does not reliably find all processes
        // https://stack-rox.atlassian.net/browse/ROX-5813
        Assume.assumeTrue(Env.mustGetOrchestratorType() != OrchestratorTypes.OPENSHIFT)

        when:
        "waiting for SR to get to an initial process list for each deployment"
        def allFound = false
        def t = new Timer(RETRIES, RETRY_DELAY)
        while (t.IsValid() && !allFound) {
            allFound = true
            for (int i = 0; i < DEPLOYMENTS.size(); i++) {
                def processes = ProcessService.getGroupedProcessByDeploymentAndContainer(DEPLOYMENTS[i].deploymentUid)
                if (!processes || processes.size() < 2) {
                    println "not yet ready to test - all processes not found for ${DEPLOYMENTS[i].name}"
                    allFound = false
                }

                if (processes) {
                    processes.forEach { group ->
                        println "SR found process ${group.name} for ${DEPLOYMENTS[i].name}"
                    }
                }
            }
        }

        then:
        allFound
    }

    def "Risk changes when a process is executed after the discovery phase"() {
        when:
        "no longer in the process discovery phase"
        // Note: This test (and ProcessWLTest.groovy) rely heavily on the deployed SR using an
        // artificially reduced process discovery phase. i.e. "1m" instead of the default 1 hour.
        // See ROX_WHITELIST_GENERATION_DURATION.
        println "sleeping for 60 seconds to ensure the discovery phase is over"
        sleep(60000)

        def before = whenEquivalent
        def withRiskIndex = before.get(0).deployment.name == deploymentWithRisk.name ? 0 : 1
        def withoutRiskIndex = (withRiskIndex + 1) % 2

        and:
        "a new process is exec'd"
        orchestrator.execInContainer(deploymentWithRisk, "ls")

        and:
        "the changes are discovered"
        def after = null
        def t = new Timer(RETRIES, RETRY_DELAY)
        while (t.IsValid()) {
            after = DeploymentService.listDeploymentsWithProcessInfo(
                    RawQuery.newBuilder().setQuery("Namespace:" + TEST_NAMESPACE).build()
            ).deploymentsList
            if (before.get(0).deployment.id != after.get(0).deployment.id) {
                after = after.reverse()
            }
            debugBeforeAndAfter(before, after)
            if (after.get(withRiskIndex).deployment.priority == before.get(withRiskIndex).deployment.priority) {
                println "not yet ready to test - there is no change yet to priorities"
                after = null
                continue
            }
            if (!after.get(withRiskIndex).whitelistStatusesList.get(0).anomalousProcessesExecuted) {
                println "not yet ready to test - there is no anomalous process spotted yet"
                after = null
                continue
            }
            println "ready to test"
            break
        }
        assert after
        whenOneHasRisk = after

        then:
        "the deployment with risk is now at a higher (lower value) priority then before"
        after.get(withRiskIndex).deployment.priority < before.get(withRiskIndex).deployment.priority

        and:
        "and thw deployment with risk is now at a higher priority (lower value) then the one without"
        after.get(withRiskIndex).deployment.priority < after.get(withoutRiskIndex).deployment.priority

        and:
        after.get(withRiskIndex).whitelistStatusesList.get(0).anomalousProcessesExecuted
    }

    def "Risk changes when an anomalous process is added to the baseline"() {
        when:
        "the baseline is updated"
        def before = whenOneHasRisk
        def withRiskIndex = before.get(0).deployment.name == deploymentWithRisk.name ? 0 : 1
        def response = null
        def t = new Timer(RETRIES, RETRY_DELAY)
        while (t.IsValid()) {
            response = ProcessBaselineService.updateProcessBaselines(
                    [ProcessBaselineOuterClass.ProcessBaselineKey
                        .newBuilder()
                            .setClusterId(clusterId)
                            .setNamespace(deploymentWithRisk.namespace)
                            .setDeploymentId(deploymentWithRisk.deploymentUid)
                            .setContainerName(deploymentWithRisk.name)
                        .build(),] as ProcessBaselineOuterClass.ProcessBaselineKey[],
                    ["/bin/ls",] as String[],
                    [] as String
            )
            if (!response || response.size() == 0) {
                println "not yet ready to test - could not update the baseline"
                continue
            }
            println "the process baseline is updated"
            break
        }
        assert response && response.size() > 0

        and:
        "SR discovers the change"
        def after = null
        t = new Timer(RETRIES, RETRY_DELAY)
        while (t.IsValid()) {
            after = DeploymentService.listDeploymentsWithProcessInfo(
                    RawQuery.newBuilder().setQuery("Namespace:" + TEST_NAMESPACE).build()
            ).deploymentsList
            if (before.get(0).deployment.id != after.get(0).deployment.id) {
                after = after.reverse()
            }
            debugBeforeAndAfter(before, after)
            if (after.get(withRiskIndex).deployment.priority == before.get(withRiskIndex).deployment.priority) {
                println "not yet ready to test - there is no change yet to priorities"
                after = null
                continue
            }
            if (after.get(withRiskIndex).whitelistStatusesList.get(0).anomalousProcessesExecuted) {
                println "not yet ready to test - the process anomaly is not cleared"
                after = null
                continue
            }
            println "ready to test"
            break
        }
        assert after

        then:
        "the updated deployment is at a lower priority (higher value) then before"
        assert after.get(withRiskIndex).deployment.priority > before.get(withRiskIndex).deployment.priority

        assert !after.get(withRiskIndex).whitelistStatusesList.get(0).anomalousProcessesExecuted
    }

    def debugBeforeAndAfter(
            List<DeploymentWithProcessInfo> before, List<DeploymentWithProcessInfo> after
    ) {
        def withRiskIndex = before.get(0).deployment.name == deploymentWithRisk.name ? 0 : 1
        def withoutRiskIndex = (withRiskIndex + 1) % 2

        println "Before:"
        println "\tDeployment with risk:    ${debugPriorityAndState(before.get(withRiskIndex))}"
        println "\tDeployment without risk: ${debugPriorityAndState(before.get(withoutRiskIndex))}"
        println "After:"
        println "\tDeployment with risk:    ${debugPriorityAndState(after.get(withRiskIndex))}"
        println "\tDeployment without risk: ${debugPriorityAndState(after.get(withoutRiskIndex))}"
        println "Process List:"
        println "\tDeployment with risk:    ${debugProcesses(deploymentWithRisk.deploymentUid)}"
        println "\tDeployment without risk: ${debugProcesses(deploymentWithoutRisk.deploymentUid)}"
    }

    def debugPriorityAndState(DeploymentWithProcessInfo dpl) {
        return "${dpl.deployment.name} "+
            "priority ${dpl.deployment.priority}, "+
            "anomalous ${dpl.whitelistStatusesList?.get(0)?.anomalousProcessesExecuted}"
    }

    def debugProcesses(String uid) {
        def processes = ProcessService.getGroupedProcessByDeploymentAndContainer(uid)
        if (!processes || processes.size() == 0) {
            return "no processes"
        }
        return processes*.name.join(", ")
    }
}
