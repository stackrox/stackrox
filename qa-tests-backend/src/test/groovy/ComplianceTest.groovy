import groups.BAT
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass.ComplianceRunScheduleInfo
import io.stackrox.proto.storage.Compliance.ComplianceResultValue
import io.stackrox.proto.storage.Compliance.ComplianceRunResults
import io.stackrox.proto.storage.Compliance.ComplianceState
import objects.Control
import objects.Deployment
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import objects.Service
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ClusterService
import services.ComplianceManagementService
import services.ComplianceService
import services.NetworkPolicyService
import services.ProcessService
import spock.lang.Shared
import spock.lang.Unroll
import v1.ComplianceServiceOuterClass.ComplianceAggregation.Result
import v1.ComplianceServiceOuterClass.ComplianceAggregation.Scope
import v1.ComplianceServiceOuterClass.ComplianceStandardMetadata

class ComplianceTest extends BaseSpecification {
    @Shared
    private static final PCI_ID = "PCI_DSS_3_2"
    @Shared
    private static final NIST_ID = "NIST_800_190"
    @Shared
    private static final HIPAA_ID = "HIPAA_164"
    @Shared
    private static final Map<String, ComplianceRunResults> BASE_RESULTS = [:]
    @Shared
    private String clusterId
    private static final SCHEDULES_SUPPORTED = false

    def setupSpec() {
        clusterId = ClusterService.getClusterId()
        def complianceRuns = ComplianceManagementService.triggerComplianceRunsAndWait()
        for (String standard : complianceRuns.keySet()) {
            def runId = complianceRuns.get(standard)
            ComplianceRunResults results = ComplianceService.getComplianceRunResult(standard, clusterId)
            assert runId == results.runMetadata.runId
            BASE_RESULTS.put(standard, results)
        }
    }

    @Category(BAT)
    def "Verify static compliance checks"() {
        given:
        "given a known list of static checks"
        List<Control> staticControls = [
                new Control(
                        "PCI_DSS_3_2:1_1_2",
                        ["StackRox shows all connections between deployments as well as connections from deployments" +
                                 " to outside of the cluster."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS).setType(Control.ControlType.CLUSTER),
                new Control(
                        "PCI_DSS_3_2:2_1",
                        ["StackRox either randomly generates a strong admin password, or the user supplies one, for " +
                                 "every deployment."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS).setType(Control.ControlType.CLUSTER),
                new Control(
                        "PCI_DSS_3_2:2_3",
                        ["StackRox only uses TLS 1.2 or higher for all API and UI communication."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS).setType(Control.ControlType.CLUSTER),
                new Control(
                        "PCI_DSS_3_2:2_4",
                        ["StackRox gives you visibility into all system components in kubernetes."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS).setType(Control.ControlType.CLUSTER),
                new Control(
                        "HIPAA_164:308_a_3_ii_a",
                        ["StackRox collects runtime process information and network flow data. This data is used to " +
                          "render a visual representation of network and service topology."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS).setType(Control.ControlType.CLUSTER),
                new Control(
                        "HIPAA_164:310_d",
                        ["StackRox collects runtime process information and network flow data. This data is used to " +
                          "render a visual representation of network and service topology."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS).setType(Control.ControlType.CLUSTER),
        ]

        expect:
        "confirm details of static checks"
        for (Control control : staticControls) {
            ComplianceRunResults result = BASE_RESULTS.get(control.standard)
            switch (control.type) {
                case Control.ControlType.CLUSTER:
                    assert result.clusterResults.controlResultsMap.get(control.id)?.overallState == control.state
                    assert result.clusterResults.controlResultsMap.get(control.id)?.evidenceList*.message
                            .containsAll(control.evidenceMessages)
                    break
                case Control.ControlType.DEPLOYMENT:
                    result.deploymentResultsMap.each {
                        k, v ->
                        assert v.controlResultsMap.get(control.id)?.overallState == control.state
                        assert v.controlResultsMap.get(control.id)?.evidenceList*.message
                                .containsAll(control.evidenceMessages)
                    }
                    break
                case Control.ControlType.NODE:
                    result.nodeResultsMap.each {
                        k, v ->
                        assert v.controlResultsMap.get(control.id)?.overallState == control.state
                        assert v.controlResultsMap.get(control.id)?.evidenceList*.message
                                .containsAll(control.evidenceMessages)
                    }
                    break
            }
        }
    }

    @Category(BAT)
    def "Verify compliance aggregation results"() {
        given:
        "get compliance aggregation results"
        List<Result> aggResults = ComplianceService.getAggregatedResults(Scope.CONTROL, [Scope.CLUSTER, Scope.STANDARD])

        expect:
        "compare"
        for (Result result : aggResults) {
            Map<ComplianceState, Set<String>> counts = [:]
            def standardId = result.aggregationKeysList.find { it.scope == Scope.STANDARD }?.id

            ComplianceRunResults run = BASE_RESULTS.get(standardId)
            println "Verifying aggregate counts for ${standardId}"
            run.clusterResults.controlResultsMap.each {
                counts.get(it.value.overallState) ?
                        counts.get(it.value.overallState).add(it.key) :
                        counts.put(it.value.overallState, [it.key] as Set)
            }
            run.nodeResultsMap.each {
                it.value.controlResultsMap.each {
                    counts.get(it.value.overallState) ?
                            counts.get(it.value.overallState).add(it.key) :
                            counts.put(it.value.overallState, [it.key] as Set)
                }
            }
            run.deploymentResultsMap.each {
                it.value.controlResultsMap.each {
                    counts.get(it.value.overallState) ?
                            counts.get(it.value.overallState).add(it.key) :
                            counts.put(it.value.overallState, [it.key] as Set)
                }
            }
            counts.get(ComplianceState.COMPLIANCE_STATE_SUCCESS)
                    .removeAll(counts.get(ComplianceState.COMPLIANCE_STATE_FAILURE) ?: [])
            counts.get(ComplianceState.COMPLIANCE_STATE_SUCCESS)
                    .removeAll(counts.get(ComplianceState.COMPLIANCE_STATE_ERROR) ?: [])
            assert result.numPassing == counts.get(ComplianceState.COMPLIANCE_STATE_SUCCESS)?.size() ?: 0
            assert result.numFailing == counts.get(ComplianceState.COMPLIANCE_STATE_FAILURE)?.size() ?: 0 +
                    counts.get(ComplianceState.COMPLIANCE_STATE_ERROR)?.size() ?: 0
        }
    }

    @Category(BAT)
    def "Verify compliance checks contain no ERROR states"() {
        expect:
        "check that each check does not have ERROR state"
        def errorChecks = [:]
        for (String standardId : BASE_RESULTS.keySet()) {
            ComplianceRunResults results = BASE_RESULTS.get(standardId)
            results.clusterResults.controlResultsMap.each {
                if (it.value.overallState == ComplianceState.COMPLIANCE_STATE_ERROR
                        // Skip NIST 4.1.4 for now, since we know it will fail
                        && it.key != "NIST_800_190:4_1_4") {
                    errorChecks.put(it.key, it.value.evidenceList)
                }
            }
            results.nodeResultsMap.each {
                it.value.controlResultsMap.each {
                    if (it.value.overallState == ComplianceState.COMPLIANCE_STATE_ERROR
                            // Skip NIST 4.1.4 for now, since we know it will fail
                            && it.key != "NIST_800_190:4_1_4") {
                        errorChecks.put(it.key, it.value.evidenceList)
                    }
                }
            }
            results.deploymentResultsMap.each {
                it.value.controlResultsMap.each {
                    if (it.value.overallState == ComplianceState.COMPLIANCE_STATE_ERROR
                            // Skip NIST 4.1.4 for now, since we know it will fail
                            && it.key != "NIST_800_190:4_1_4") {
                        errorChecks.put(it.key, it.value.evidenceList)
                    }
                }
            }
        }
        assert errorChecks.size() == 0
    }

    @Category(BAT)
    def "Verify all compliance checks contain evidence"() {
        expect:
        "check that each check contains evidence"
        def missingEvidenceChecks = [] as Set
        for (String standardId : BASE_RESULTS.keySet()) {
            ComplianceRunResults results = BASE_RESULTS.get(standardId)
            results.clusterResults.controlResultsMap.each {
                if (it.value.evidenceList.size() == 0) {
                    missingEvidenceChecks.add(it.key)
                }
            }
            results.nodeResultsMap.each {
                it.value.controlResultsMap.each {
                    if (it.value.evidenceList.size() == 0) {
                        missingEvidenceChecks.add(it.key)
                    }
                }
            }
            results.deploymentResultsMap.each {
                it.value.controlResultsMap.each {
                    if (it.value.evidenceList.size() == 0) {
                        missingEvidenceChecks.add(it.key)
                    }
                }
            }
        }
        assert missingEvidenceChecks.size() == 0
    }

    private determineOverallState(List<ComplianceState> evidenceStates) {
        def returnState = ComplianceState.COMPLIANCE_STATE_UNKNOWN
        for (ComplianceState state : evidenceStates) {
            if (state > returnState) {
                returnState = state
            }
        }
        return returnState
    }

    @Category(BAT)
    def "Verify overall state of each check is correct based on each piece of evidence"() {
        expect:
        "check that the state of each check is correct based on each piece of evidence"
        def invalidOverallState = [:]
        for (String standardId : BASE_RESULTS.keySet()) {
            ComplianceRunResults results = BASE_RESULTS.get(standardId)
            results.clusterResults.controlResultsMap.each {
                if (it.value.overallState != determineOverallState(it.value.evidenceList*.state)) {
                    invalidOverallState.put(it.key, it.value.evidenceList)
                }
            }
            results.nodeResultsMap.each {
                it.value.controlResultsMap.each {
                    if (it.value.overallState != determineOverallState(it.value.evidenceList*.state)) {
                        invalidOverallState.put(it.key, it.value.evidenceList)
                    }
                }
            }
            results.deploymentResultsMap.each {
                it.value.controlResultsMap.each {
                    if (it.value.overallState != determineOverallState(it.value.evidenceList*.state)) {
                        invalidOverallState.put(it.key, it.value.evidenceList)
                    }
                }
            }
        }
        assert invalidOverallState.size() == 0
    }

    @Category(BAT)
    def "Verify all kube-system namespace checks are SKIPPED"() {
        expect:
        "check that each check does not have ERROR state"
        def kubeSystemNotSkipped = [] as Set
        for (String standardId : BASE_RESULTS.keySet()) {
            ComplianceRunResults results = BASE_RESULTS.get(standardId)
            results.deploymentResultsMap.each {
                it.value.controlResultsMap.each {
                    if (results.domain.deploymentsMap.get(it.key)?.namespace == "kube-system" &&
                            it.value.overallState != ComplianceState.COMPLIANCE_STATE_SKIP) {
                        kubeSystemNotSkipped.add(it.key)
                    }
                }
            }
        }
        assert kubeSystemNotSkipped.size() == 0
    }

    @Category(BAT)
    def "Verify compliance scheduling"() {
        // Schedules are not yet supported, so skipping this test for now.
        // Once we fully support Compliance Run scheduling, we can reneable.
        // Running this test now will expose ROX-1255
        Assume.assumeTrue(SCHEDULES_SUPPORTED)

        given:
        "List of Standards"
        List<ComplianceStandardMetadata> standards = ComplianceService.getComplianceStandards()

        when:
        "create a schedule"
        ComplianceRunScheduleInfo info = ComplianceManagementService.addSchedule(
                        standards.get(0).id,
                        clusterId,
                        "* 4 * * *"
                )
        assert info
        assert ComplianceManagementService.getSchedules().find { it.schedule.id == info.schedule.id }

        and:
        "verify schedule details"
        Calendar nextRun = Calendar.getInstance(TimeZone.getTimeZone("GMT"))
        nextRun.setTime(new Date(info.nextRunTime.seconds * 1000))
        Calendar now = Calendar.getInstance(TimeZone.getTimeZone("GMT"))
        now.get(Calendar.HOUR_OF_DAY) < 4 ?: now.add(Calendar.DAY_OF_YEAR, 1)
        assert nextRun.get(Calendar.HOUR_OF_DAY) == 4
        assert nextRun.get(Calendar.DAY_OF_YEAR) == now.get(Calendar.DAY_OF_YEAR)

        and:
        "update schedule"
        int minute = now.get(Calendar.MINUTE)
        int hour = now.get(Calendar.HOUR_OF_DAY)
        if (minute < 59) {
            minute++
        } else {
            minute = 0
            hour ++
        }
        String cron = "${minute} ${hour} * * *"
        ComplianceRunScheduleInfo update = ComplianceManagementService.updateSchedule(
                        info.schedule.id,
                        standards.get(0).id,
                        clusterId,
                        cron
                )
        assert update

        and:
        "verify update"
        assert ComplianceManagementService.getSchedules().find {
            it.schedule.id == info.schedule.id && it.schedule.crontabSpec == cron
        }

        and:
        "verify standard started on schedule"
        println "Waiting for schedule to start..."
        while (now.get(Calendar.MINUTE) < minute) {
            sleep 1000
            now = Calendar.getInstance(TimeZone.getTimeZone("GMT"))
        }
        long mostRecent = 0
        ComplianceManagementService.getRecentRuns(standards.get(0).id).each {
            if (it.startTime.seconds > mostRecent) {
                mostRecent = it.startTime.seconds
            }
        }
        assert mostRecent >= update.nextRunTime.seconds

        then:
        "delete schedule"
        ComplianceManagementService.deleteSchedule(info.schedule.id)
    }

    @Category([BAT])
    def "Verify checks based on Integrations"() {
        def failureEvidence = ["No image scanners are being used in the cluster"]
        def controls = [
                new Control("PCI_DSS_3_2:6_1", failureEvidence, ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control("PCI_DSS_3_2:6_5_6", failureEvidence, ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control("PCI_DSS_3_2:11_2_1", failureEvidence, ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control("NIST_800_190:4_1_1", failureEvidence, ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control("NIST_800_190:4_1_2", failureEvidence, ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control("HIPAA_164:306_e", failureEvidence, ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control("HIPAA_164:308_a_1_ii_b", failureEvidence, ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control("HIPAA_164:308_a_7_ii_e", failureEvidence, ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control("HIPAA_164:310_a_1", failureEvidence, ComplianceState.COMPLIANCE_STATE_FAILURE),
        ]

        given:
        "remove image integrations"
        def removed = Services.deleteDockerTrustedRegistry(dtrId)

        when:
        "trigger compliance runs"
        def pciRunId = ComplianceManagementService.triggerComplianceRunAndWait(PCI_ID, clusterId)
        ComplianceRunResults pciResults = ComplianceService.getComplianceRunResult(PCI_ID, clusterId)
        assert pciResults.getRunMetadata().runId == pciRunId

        def nistRunId = ComplianceManagementService.triggerComplianceRunAndWait(NIST_ID, clusterId)
        ComplianceRunResults nistResults = ComplianceService.getComplianceRunResult(NIST_ID, clusterId)
        assert nistResults.getRunMetadata().runId == nistRunId

        def hipaaRunid = ComplianceManagementService.triggerComplianceRunAndWait(HIPAA_ID, clusterId)
        ComplianceRunResults hipaaResults = ComplianceService.getComplianceRunResult(HIPAA_ID, clusterId)
        assert hipaaResults.getRunMetadata().runId == hipaaRunid

        then:
        "confirm state and evidence of expected controls"
        Map<String, ComplianceResultValue> clusterResults = [:]
        clusterResults << pciResults.getClusterResults().controlResultsMap
        clusterResults << nistResults.getClusterResults().controlResultsMap
        clusterResults << hipaaResults.getClusterResults().controlResultsMap
        assert clusterResults
        def missingControls = []
        for (Control control : controls) {
            if (clusterResults.keySet().contains(control.id)) {
                println "Validating ${control.id}"
                ComplianceResultValue value = clusterResults.get(control.id)
                assert value.overallState == control.state
                assert value.evidenceList*.message.containsAll(control.evidenceMessages)
            } else {
                missingControls.add(control)
            }
        }
        assert missingControls*.id.size() == 0

        cleanup:
        "re-add image integrations"
        if (removed) {
            dtrId = Services.addDockerTrustedRegistry()
        }
    }

    @Category([BAT])
    def "Verify checks based on Deployments"() {
        def controls = [
                new Control(
                        "PCI_DSS_3_2:1_3_5",
                        ["Deployment uses UDP, which allows data exchange without an established connection"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "PCI_DSS_3_2:1_2_1",
                        ["No egress network policies apply to this deployment, hence all egress connections are " +
                                 "allowed",
                         "Deployment uses host network, which allows it to subvert network policies"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "PCI_DSS_3_2:1_3_2",
                        ["Deployment uses host network, which allows it to subvert network policies"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "PCI_DSS_3_2:2_2_5",
                        ["Deployment has exposed ports that are not receiving traffic: [80]"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "NIST_800_190:4_3_3",
                        ["No egress network policies apply to this deployment, hence all egress connections are " +
                                 "allowed",
                         "Deployment uses host network, which allows it to subvert network policies"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "NIST_800_190:4_4_2",
                        ["No egress network policies apply to this deployment, hence all egress connections are " +
                                 "allowed",
                         "Deployment uses host network, which allows it to subvert network policies"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "HIPAA_164:308_a_4_ii_b",
                        ["No egress network policies apply to this deployment, hence all egress connections are " +
                                 "allowed",
                         "Deployment uses host network, which allows it to subvert network policies"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
        ]

        given:
        "create Deployment that forces checks to fail"
        Deployment deployment = new Deployment()
                .setName("compliance-deployment")
                .setImage("nginx:1.15.4-alpine")
                .addPort(80, "UDP")
                .setCommand(["/bin/sh", "-c",])
                .setArgs(["dd if=/dev/zero of=/dev/null & yes"])
                .setHostNetwork(true)
        Service service = new Service(deployment)
        orchestrator.createService(service)
        orchestrator.createDeployment(deployment)
        assert Services.waitForDeployment(deployment)

        and:
        "apply network policy to the deployment"
        NetworkPolicy policy = new NetworkPolicy("deny-all-namespace-ingress-compliance")
                .setNamespace("qa")
                .addPodSelector()
                .addPolicyType(NetworkPolicyTypes.INGRESS)
        def policyId = orchestrator.applyNetworkPolicy(policy)
        assert NetworkPolicyService.waitForNetworkPolicy(policyId)

        and:
        "verify deployment fully detected"
        Set<String> receivedProcessPaths = ProcessService.getUniqueProcessPaths(deployment.deploymentUid)
        def sleepTime = 0L
        while (receivedProcessPaths.size() <= 1 && sleepTime < 60000) {
            println "Didn't find all the expected processes, retrying..."
            sleep(2000)
            sleepTime += 2000
            receivedProcessPaths = ProcessService.getUniqueProcessPaths(deployment.deploymentUid)
        }
        assert receivedProcessPaths.size() > 1

        when:
        "trigger compliance runs"
        def pciRunId = ComplianceManagementService.triggerComplianceRunAndWait(PCI_ID, clusterId)
        ComplianceRunResults pciResults = ComplianceService.getComplianceRunResult(PCI_ID, clusterId)
        assert pciResults.getRunMetadata().runId == pciRunId

        def nistRunId = ComplianceManagementService.triggerComplianceRunAndWait(NIST_ID, clusterId)
        ComplianceRunResults nistResults = ComplianceService.getComplianceRunResult(NIST_ID, clusterId)
        assert nistResults.getRunMetadata().runId == nistRunId

        def hipaaRunid = ComplianceManagementService.triggerComplianceRunAndWait(HIPAA_ID, clusterId)
        ComplianceRunResults hipaaResults = ComplianceService.getComplianceRunResult(HIPAA_ID, clusterId)
        assert hipaaResults.getRunMetadata().runId == hipaaRunid

        then:
        "confirm state and evidence of expected controls"
        Map<String, ComplianceResultValue> deploymentResults = [:]
        deploymentResults << pciResults.getDeploymentResultsMap().get(deployment.deploymentUid).controlResultsMap
        deploymentResults << nistResults.getDeploymentResultsMap().get(deployment.deploymentUid).controlResultsMap
        deploymentResults << hipaaResults.getDeploymentResultsMap().get(deployment.deploymentUid).controlResultsMap
        assert deploymentResults
        def missingControls = []
        for (Control control : controls) {
            if (deploymentResults.keySet().contains(control.id)) {
                println "Validating deployment control ${control.id}"
                ComplianceResultValue value = deploymentResults.get(control.id)
                assert value.overallState == control.state
                assert value.evidenceList*.message.containsAll(control.evidenceMessages)
            } else {
                missingControls.add(control)
            }
        }
        assert missingControls*.id.size() == 0

        cleanup:
        "remove deployment"
        if (deployment) {
            orchestrator.deleteDeployment(deployment)
        }
        if (service) {
            orchestrator.deleteService(service.name, service.namespace)
        }
        if (policyId) {
            orchestrator.deleteNetworkPolicy(policy)
        }
    }

    // Legacy Benchmark Test - to remove once Compliance is done
    @Unroll
    @Category(BAT)
    def "Verify that we can run a benchmark: "(String benchmarkName) {
        when:
        "Trigger a compliance benchmark"
        String benchmarkID = ComplianceService.getBenchmark(benchmarkName)
        println ("Found benchmark ID ${benchmarkID} for ${benchmarkName}")
        String clusterID = ClusterService.getClusterId()
        ComplianceService.runBenchmark(benchmarkID, clusterID)

        then:
        "Verify Scan is run"
        assert ComplianceService.checkBenchmarkRan(benchmarkID, clusterID)

        cleanup:
        "Make sure the daemonset benchmark is gone"
        orchestrator.waitForDaemonSetDeletion("benchmark", "stackrox")

        where:
        "Data inputs are :"

        benchmarkName | _
        "CIS Kubernetes v1.2.0 Benchmark" | _
        "CIS Docker v1.1.0 Benchmark" | _
    }
}
