import static io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery.newBuilder

import orchestratormanager.OrchestratorTypes

import io.stackrox.proto.api.v1.DeploymentServiceOuterClass.ListDeploymentsWithProcessInfoResponse.DeploymentWithProcessInfo
import io.stackrox.proto.storage.DeploymentOuterClass.ListDeployment
import io.stackrox.proto.storage.ProcessBaselineOuterClass

import objects.Deployment
import services.ClusterService
import services.DeploymentService
import services.ImageService
import services.ProcessBaselineService
import services.ProcessService
import util.Env
import util.Timer

import spock.lang.IgnoreIf
import spock.lang.Shared
import spock.lang.Stepwise

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

    static final private int RETRIES = isRaceBuild() ? 120 : (
            (Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT) ? 70 : 35)
    static final private int RETRY_DELAY = 5
    static final private List<Deployment> DEPLOYMENTS = []
    static final private String TEST_NAMESPACE = "qa-risk-${UUID.randomUUID()}"

    def setupSpec() {
        clusterId = ClusterService.getClusterId()

        // ROX-6260: pre scan the image to avoid different risk scores
        ImageService.scanImage(TEST_IMAGE, false)

        for (int i = 0; i < 2; i++) {
            DEPLOYMENTS.push(
                    new Deployment()
                            .setName("risk-deployment-${i}")
                            .setNamespace(TEST_NAMESPACE)
                            .setImage(TEST_IMAGE)
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
            orchestrator.deleteAndWaitForDeploymentDeletion(deployment)
        }
        orchestrator.deleteNamespace(TEST_NAMESPACE)
    }

    def "Deployment count == 2"() {
        expect:
        listDeployments().size() == DEPLOYMENTS.size()
    }

    def "Risk is the same for equivalent deployments"() {
        when:
        "waiting for SR to get to an initial priority and process baseline for each deployment"
        def t = new Timer(RETRIES, RETRY_DELAY)
        while (t.IsValid()) {
            def response = listDeployments()
            if (!response || response.size() < DEPLOYMENTS.size()) {
                log.info "not yet ready to test - no deployments found"
                continue
            }
            if (response.any { it.baselineStatusesList.size() == 0 }) {
                log.info "not yet ready to test - container summary status are not set"
                continue
            }

            def processesFound = true
            for (int i = 0; i < DEPLOYMENTS.size(); i++) {
                def processes = ProcessBaselineService.getProcessBaseline(clusterId, DEPLOYMENTS[i], null, 0)
                if (!processes || processes.elementsList.size() == 0) {
                    log.info "not yet ready to test - processes not found for ${DEPLOYMENTS[i].name}"
                    processesFound = false
                }

                if (processes) {
                    processes.elementsList.forEach { element ->
                        log.info "SR found ${element.element.processName} for ${DEPLOYMENTS[i].name}"
                    }
                }
            }

            if (!processesFound) {
                log.info "not yet ready to test - processes not found"
                continue
            }

            List<String> deploymentNamesWithoutRisk = listDeployments()
                    .collect {
                        DeploymentService.getDeploymentWithRisk(it.getDeployment().getId()) }
                    .findAll { !it.hasRisk() }
                    .collect { it.deployment.name }

            if (!deploymentNamesWithoutRisk.isEmpty()) {
                log.info "not yet ready to test - risks not found ${deploymentNamesWithoutRisk}"
                continue
            }

            log.info "ready to test"
            whenEquivalent = response
            break
        }

        assert whenEquivalent, "SR found the deployments, containers and processes required"

        def one = whenEquivalent.get(0)
        def two = whenEquivalent.get(1)
        log.info debugPriorityAndState(one)
        log.info debugPriorityAndState(two)

        then:
        "should have the same risk"
        risk(one.deployment) == risk(two.deployment)

        and:
        "should be at equivalent priority"
        one.deployment.priority == two.deployment.priority

        and:
        "not anomalous"
        !one.baselineStatusesList.get(0).anomalousProcessesExecuted
        !two.baselineStatusesList.get(0).anomalousProcessesExecuted
    }

    // Skip for OpenShift, it does not reliably find all processes
    // https://stack-rox.atlassian.net/browse/ROX-5813
    @IgnoreIf({ Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT })
    def "Processes grouped by deployment (GetGroupedProcessByDeploymentAndContainer)"() {
        when:
        "waiting for SR to get to an initial process list for each deployment"
        def allFound = false
        def t = new Timer(RETRIES, RETRY_DELAY)
        while (t.IsValid() && !allFound) {
            allFound = true
            for (int i = 0; i < DEPLOYMENTS.size(); i++) {
                def processes = ProcessService.getGroupedProcessByDeploymentAndContainer(DEPLOYMENTS[i].deploymentUid)
                if (!processes || processes.size() < 2) {
                    log.info "not yet ready to test - all processes not found for ${DEPLOYMENTS[i].name}"
                    allFound = false
                }

                if (processes) {
                    processes.forEach { group ->
                        log.info "SR found process ${group.name} for ${DEPLOYMENTS[i].name}"
                    }
                }
            }
        }

        then:
        allFound
    }

    def "Risk priority changes when a process is executed after the discovery phase"() {
        when:
        "no longer in the process discovery phase"
        // Note: This test (and ProcessWLTest.groovy) rely heavily on the deployed SR using an
        // artificially reduced process discovery phase. i.e. "1m" instead of the default 1 hour.
        // See ROX_BASELINE_GENERATION_DURATION.
        log.info "sleeping for 60 seconds to ensure the discovery phase is over"
        sleep(60000)

        def before = whenEquivalent
        def withRiskIndex = before.get(0).deployment.name == deploymentWithRisk.name ? 0:1
        def withoutRiskIndex = ( withRiskIndex + 1 ) % 2
        def riskBefore = risk(before.get(withRiskIndex).deployment)

        and:
        "a new process is exec'd"
        orchestrator.execInContainer(deploymentWithRisk, "ls")

        and:
        "the changes are discovered"
        // Now the risk score of one deployment diverges from the risk score of the
        // other. This must cause the change in priority since one
        // deployment is strictly riskier than the other one.
        def after = null
        def t = new Timer(RETRIES, RETRY_DELAY)
        while (t.IsValid()) {
            after = listDeployments()
            if (before.get(0).deployment.id != after.get(0).deployment.id) {
                after = after.reverse()
            }
            debugBeforeAndAfter(before, after)
            if (after.get(withRiskIndex).deployment.priority == after.get(withoutRiskIndex).deployment.priority) {
                log.info "not yet ready to test - there is no change yet to priorities"
                after = null
                continue
            }
            if (!after.get(withRiskIndex).baselineStatusesList.get(0).anomalousProcessesExecuted) {
                log.info "not yet ready to test - there is no anomalous process spotted yet"
                after = null
                continue
            }
            log.info "ready to test"
            break
        }
        assert after
        whenOneHasRisk = after

        then:
        "the deployment with risk has now higher risk score then before"
        risk(after.get(withRiskIndex).deployment) > riskBefore

        and:
        "and the deployment with risk is now at a higher priority (lower value) then the one without"
        after.get(withRiskIndex).deployment.priority < after.get(withoutRiskIndex).deployment.priority

        and:
        after.get(withRiskIndex).baselineStatusesList.get(0).anomalousProcessesExecuted
    }

    def "Risk changes when an anomalous process is added to the baseline"() {
        when:
        "the baseline is updated"
        def before = whenOneHasRisk
        def withRiskIndex = before.get(0).deployment.name == deploymentWithRisk.name ? 0 : 1
        def riskBefore = risk(before.get(withRiskIndex).deployment)
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
                log.info "not yet ready to test - could not update the baseline"
                continue
            }
            log.info "the process baseline is updated"
            break
        }
        assert response && response.size() > 0

        and:
        "SR discovers the change"
        def after = null
        t = new Timer(RETRIES, RETRY_DELAY)
        while (t.IsValid()) {
            after = listDeployments()
            if (before.get(0).deployment.id != after.get(0).deployment.id) {
                after = after.reverse()
            }
            debugBeforeAndAfter(before, after)
            if (risk(after.get(withRiskIndex).deployment) == riskBefore) {
                log.info "not yet ready to test - there is no change yet to risk score"
                after = null
                continue
            }
            if (after.get(withRiskIndex).baselineStatusesList.get(0).anomalousProcessesExecuted) {
                log.info "not yet ready to test - the process anomaly is not cleared"
                after = null
                continue
            }
            log.info "ready to test"
            break
        }
        assert after

        then:
        "the updated deployment has a lower risk score then before"
        assert risk(after.get(withRiskIndex).deployment) < riskBefore

        assert !after.get(withRiskIndex).baselineStatusesList.get(0).anomalousProcessesExecuted
    }

    def debugBeforeAndAfter(
            List<DeploymentWithProcessInfo> before, List<DeploymentWithProcessInfo> after
    ) {
        def withRiskIndex = before.get(0).deployment.name == deploymentWithRisk.name ? 0 : 1
        def withoutRiskIndex = (withRiskIndex + 1) % 2

        log.info "Before:"
        log.info "\tDeployment with risk:    ${debugPriorityAndState(before.get(withRiskIndex))}"
        log.info "\tDeployment without risk: ${debugPriorityAndState(before.get(withoutRiskIndex))}"
        log.info "After:"
        log.info "\tDeployment with risk:    ${debugPriorityAndState(after.get(withRiskIndex))}"
        log.info "\tDeployment without risk: ${debugPriorityAndState(after.get(withoutRiskIndex))}"
        log.info "Process List:"
        log.info "\tDeployment with risk:    ${debugProcesses(deploymentWithRisk.deploymentUid)}"
        log.info "\tDeployment without risk: ${debugProcesses(deploymentWithoutRisk.deploymentUid)}"
    }

    def debugPriorityAndState(DeploymentWithProcessInfo dpl) {
        return "${dpl.deployment.name} "+
            "priority ${dpl.deployment.priority}, "+
            "anomalous ${dpl.baselineStatusesList?.get(0)?.anomalousProcessesExecuted}"
    }

    def debugProcesses(String uid) {
        def processes = ProcessService.getGroupedProcessByDeploymentAndContainer(uid)
        if (!processes || processes.size() == 0) {
            return "no processes"
        }
        return processes*.name.join(", ")
    }

    private static float risk(ListDeployment deployment) {
        DeploymentService.getDeploymentWithRisk(deployment.id).deployment.riskScore
    }

    private static List<DeploymentWithProcessInfo> listDeployments() {
        DeploymentService.listDeploymentsWithProcessInfo(
                newBuilder().setQuery("Namespace:" + TEST_NAMESPACE).build()
        )?.deploymentsList
    }
}
