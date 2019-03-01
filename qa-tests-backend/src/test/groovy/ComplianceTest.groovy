import com.opencsv.bean.CsvToBean
import com.opencsv.bean.CsvToBeanBuilder
import com.opencsv.bean.HeaderColumnNameTranslateMappingStrategy
import common.Constants
import groups.BAT
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass
import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass.ComplianceRunScheduleInfo
import io.stackrox.proto.storage.Compliance
import io.stackrox.proto.storage.Compliance.ComplianceResultValue
import io.stackrox.proto.storage.Compliance.ComplianceRunResults
import io.stackrox.proto.storage.Compliance.ComplianceState
import io.stackrox.proto.storage.ImageOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import objects.Control
import objects.CsvRow
import objects.Deployment
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import objects.Service
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ClusterService
import services.ComplianceManagementService
import services.ComplianceService
import services.CreatePolicyService
import services.NetworkPolicyService
import services.ImageService
import services.ProcessService
import spock.lang.Shared
import v1.ComplianceServiceOuterClass.ComplianceControl
import v1.ComplianceServiceOuterClass.ComplianceStandard
import v1.ComplianceServiceOuterClass.ComplianceAggregation.Result
import v1.ComplianceServiceOuterClass.ComplianceAggregation.Scope
import v1.ComplianceServiceOuterClass.ComplianceStandardMetadata

import java.nio.charset.StandardCharsets
import java.nio.file.Files
import java.nio.file.Paths

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
    @Shared
    private clairifyId = ""

    def setupSpec() {
        // Get cluster ID
        clusterId = ClusterService.getClusterId()

        // Clear image cache and add clairify/remove dtr scanners
        Services.deleteImageIntegration(dtrId)
        ImageService.clearImageCaches()
        dtrId = Services.addDockerTrustedRegistry(false)
        orchestrator.createClairifyDeployment()
        clairifyId = Services.addClairifyScanner(orchestrator.getClairifyEndpoint())

        // Generate baseline compliance runs
        def complianceRuns = ComplianceManagementService.triggerComplianceRunsAndWait()
        for (String standard : complianceRuns.keySet()) {
            def runId = complianceRuns.get(standard)
            ComplianceRunResults results = ComplianceService.getComplianceRunResult(standard, clusterId).results
            assert runId == results.runMetadata.runId
            BASE_RESULTS.put(standard, results)
        }
    }

    def cleanupSpec() {
        Services.deleteImageIntegration(clairifyId)
        Services.deleteImageIntegration(dtrId)
        orchestrator.deleteDeployment(new Deployment(name: "clairify", namespace: "stackrox"))
        orchestrator.waitForDeploymentDeletion(new Deployment(name: "clairify", namespace: "stackrox"))
        orchestrator.deleteService("clairify", "stackrox")
        orchestrator.waitForServiceDeletion(new Service("clairify", "stackrox"))
        ImageService.clearImageCaches()
        dtrId = Services.addDockerTrustedRegistry()
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
                        ["Runtime support is enabled (or collector service is running) for cluster remote. Network " +
                                 "visualization for active network connections is possible."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS).setType(Control.ControlType.CLUSTER),
                new Control(
                        "HIPAA_164:310_d",
                        ["Runtime support is enabled (or collector service is running) for cluster remote. Network " +
                                 "visualization for active network connections is possible."],
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
                if (it.value.overallState == ComplianceState.COMPLIANCE_STATE_ERROR) {
                    errorChecks.put(it.key, it.value.evidenceList)
                }
            }
            results.nodeResultsMap.each {
                it.value.controlResultsMap.each {
                    if (it.value.overallState == ComplianceState.COMPLIANCE_STATE_ERROR) {
                        errorChecks.put(it.key, it.value.evidenceList)
                    }
                }
            }
            results.deploymentResultsMap.each {
                it.value.controlResultsMap.each {
                    if (it.value.overallState == ComplianceState.COMPLIANCE_STATE_ERROR) {
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

    private convertStringState(String state) {
        switch (state) {
            case "Fail":
                return ComplianceState.COMPLIANCE_STATE_FAILURE
            case "Pass":
                return ComplianceState.COMPLIANCE_STATE_SUCCESS
            case "Error":
                return ComplianceState.COMPLIANCE_STATE_ERROR
            case "N/A":
                return ComplianceState.COMPLIANCE_STATE_SKIP
        }
    }

    private convertStandardToId(String standard) {
        return standard
                .replace('.', '_')
                .replace(' ', '_')
                .replace('-', '_')
    }

    @Category([BAT])
    def "Verify compliance csv export"() {
        when:
        "a compliance CSV export file"
        def exportFile = ComplianceService.exportComplianceCsv()

        then:
        "parse and verify export file"
        try {
            HeaderColumnNameTranslateMappingStrategy<CsvRow> strategy =
                    new HeaderColumnNameTranslateMappingStrategy<CsvRow>()
            strategy.setType(CsvRow)
            strategy.setColumnMapping(Constants.CSV_COLUMN_MAPPING)

            Reader reader = Files.newBufferedReader(Paths.get(exportFile), StandardCharsets.UTF_8)
            reader.mark(1)
            def index = reader.readLine().indexOf("Standard")
            reader.reset()
            for (int i = 0; i < index; i++) {
                reader.read()
            }
            CsvToBean<CsvRow> csvToBean = new CsvToBeanBuilder(reader)
                    .withType(CsvRow)
                    .withIgnoreLeadingWhiteSpace(true)
                    .withMappingStrategy(strategy)
                    .build()

            Iterator<CsvRow> csvUserIterator = csvToBean.iterator()
            int rowNumber = 0
            int verifiedRows = 0

            Map<String, ComplianceStandard> sDetails = ComplianceService.getComplianceStandards().collectEntries {
                [(it.id) : ComplianceService.getComplianceStandardDetails(it.id)]
            }

            while (csvUserIterator.hasNext()) {
                CsvRow row = csvUserIterator.next()
                def controlId = row.standard ?
                        convertStandardToId(row.control.replaceAll("\"*=*\\(*\\)*", "")) :
                        null
                def standardId = row.standard ?
                        convertStandardToId(row.standard) :
                        null
                rowNumber++
                ComplianceRunResults result = BASE_RESULTS.get(standardId)
                assert result
                ComplianceControl control = sDetails.get(standardId).controlsList.find {
                    it.id == "${standardId}:${controlId}"
                }
                ComplianceResultValue value
                switch (row.objectType.toLowerCase()) {
                    case "cluster":
                        value = result.clusterResults.controlResultsMap.find {
                            it.key == "${standardId}:${controlId}"
                        }?.value
                        break
                    case "node":
                        value = result.nodeResultsMap.get(
                            result.domain.nodesMap.find { it.value.name == row.objectName }?.key
                        )?.controlResultsMap?.find {
                            it.key == "${standardId}:${controlId}"
                        }?.value
                        break
                    default:
                        value = result.deploymentResultsMap.get(
                            result.domain.deploymentsMap.find {
                            it.value.name == row.objectName && it.value.namespace == row.namespace
                            }?.key
                        )?.controlResultsMap?.find {
                            it.key == "${standardId}:${controlId}"
                        }?.value
                        break
                }
                assert value
                assert control
                if (value.evidenceCount == 1) {
                    assert convertStringState(row.state) ?
                            convertStringState(row.state) == value.overallState :
                            row.state == "Unknown"
                    verifiedRows++
                }
                //assert row.controlDescription == control.description
                assert control.description.startsWith(row.controlDescription[0..row.controlDescription.length() - 5])
            }
            println "Verified ${verifiedRows} out of ${rowNumber} total rows"
        } catch (Exception e) {
            println e.printStackTrace()
        }
    }

    @Category(BAT)
    def "Verify compliance scheduling"() {
        // Schedules are not yet supported, so skipping this test for now.
        // Once we fully support Compliance Run scheduling, we can reneable.
        // Running this test now will expose ROX-1255
        Assume.assumeTrue(Constants.SCHEDULES_SUPPORTED)

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
                new Control(
                        "HIPAA_164:314_a_2_i_c",
                        ["At least one notifier is enabled."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
        ]

        given:
        "remove image integrations"
        def clairRemoved = Services.deleteImageIntegration(clairifyId)

        and:
        "add notifier integration"
        def slackNotiferId = Services.addSlackNotifier("Slack Notifier").id

        when:
        "trigger compliance runs"
        def pciRunId = ComplianceManagementService.triggerComplianceRunAndWait(PCI_ID, clusterId)
        ComplianceRunResults pciResults = ComplianceService.getComplianceRunResult(PCI_ID, clusterId).results
        assert pciResults.getRunMetadata().runId == pciRunId

        def nistRunId = ComplianceManagementService.triggerComplianceRunAndWait(NIST_ID, clusterId)
        ComplianceRunResults nistResults = ComplianceService.getComplianceRunResult(NIST_ID, clusterId).results
        assert nistResults.getRunMetadata().runId == nistRunId

        def hipaaRunid = ComplianceManagementService.triggerComplianceRunAndWait(HIPAA_ID, clusterId)
        ComplianceRunResults hipaaResults = ComplianceService.getComplianceRunResult(HIPAA_ID, clusterId).results
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
        if (clairRemoved) {
            clairifyId = Services.addClairifyScanner(orchestrator.getClairifyEndpoint())
        }
        if (slackNotiferId) {
            Services.deleteNotifier(slackNotiferId)
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
        "Skip test for now, until we can stabilize the test"
        Assume.assumeTrue(Constants.RUN_FLAKEY_TESTS)

        and:
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
        ComplianceRunResults pciResults = ComplianceService.getComplianceRunResult(PCI_ID, clusterId).results
        assert pciResults.getRunMetadata().runId == pciRunId

        def nistRunId = ComplianceManagementService.triggerComplianceRunAndWait(NIST_ID, clusterId)
        ComplianceRunResults nistResults = ComplianceService.getComplianceRunResult(NIST_ID, clusterId).results
        assert nistResults.getRunMetadata().runId == nistRunId

        def hipaaRunid = ComplianceManagementService.triggerComplianceRunAndWait(HIPAA_ID, clusterId)
        ComplianceRunResults hipaaResults = ComplianceService.getComplianceRunResult(HIPAA_ID, clusterId).results
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
        if (deployment?.deploymentUid) {
            orchestrator.deleteDeployment(deployment)
            orchestrator.deleteService(service.name, service.namespace)
        }
        if (policyId) {
            orchestrator.deleteNetworkPolicy(policy)
        }
    }

    @Category([BAT])
    def "Verify checks based on Policies"() {
        def controls = [
                new Control(
                        "NIST_800_190:4_1_1",
                        ["Build time policies that disallows images with a critical CVSS score is enabled and enforced",
                         "At least one build time policy is enabled and enforced",
                         "Cluster has an image scanner in use"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "NIST_800_190:4_1_2",
                        ["Policies are in place to detect and enforce \"Privileges\" category issues.",
                         "At least one build time policy is enabled and enforced",
                         "Cluster has an image scanner in use"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "NIST_800_190:4_1_4",
                        ["Policy that detects secrets in env is enabled and enforced"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "NIST_800_190:4_2_2",
                        ["Policy that disallows old images to be deployed is enabled and enforced",
                         "Policy that disallows images with tag 'latest' to be deployed is enabled and enforced"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
        ]
        def enforcementPolicies = [
                "CVSS >= 7",
                "Privileged Container",
                "90-Day Image Age",
                "Latest tag",
        ]

        given:
        "update policies"
        Services.updatePolicyLifecycleStage(
                "CVSS >= 7",
                [PolicyOuterClass.LifecycleStage.BUILD, PolicyOuterClass.LifecycleStage.DEPLOY])
        for (String policyName : enforcementPolicies) {
            def enforcements = []
            PolicyOuterClass.Policy policy = Services.getPolicyByName(policyName)
            if (policy.lifecycleStagesList.contains(PolicyOuterClass.LifecycleStage.BUILD)) {
                enforcements.add(PolicyOuterClass.EnforcementAction.FAIL_BUILD_ENFORCEMENT)
            }
            if (policy.lifecycleStagesList.contains(PolicyOuterClass.LifecycleStage.DEPLOY)) {
                enforcements.add(PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT)
            }
            if (policy.lifecycleStagesList.contains(PolicyOuterClass.LifecycleStage.RUNTIME)) {
                enforcements.add(PolicyOuterClass.EnforcementAction.KILL_POD_ENFORCEMENT)
            }
            Services.updatePolicyEnforcement(policyName, enforcements)
        }
        def policyId = CreatePolicyService.createNewPolicy(PolicyOuterClass.Policy.newBuilder()
                .setName("XYZ Compliance Secrets")
                .setDescription("Test Secrets in Compliance")
                .setRationale("Test Secrets in Compliance")
                .addLifecycleStages(PolicyOuterClass.LifecycleStage.DEPLOY)
                .addEnforcementActions(PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT)
                .addCategories("Image Assurance")
                .setDisabled(false)
                .setSeverityValue(2)
                .setFields(PolicyOuterClass.PolicyFields.newBuilder()
                        .setEnv(PolicyOuterClass.KeyValuePolicy.newBuilder()
                                .setKey(".*SECRET.*")
                                .setValue(".*"))
                        .build())
                .build())

        when:
        "trigger compliance runs"
        def nistRunId = ComplianceManagementService.triggerComplianceRunAndWait(NIST_ID, clusterId)
        ComplianceRunResults nistResults = ComplianceService.getComplianceRunResult(NIST_ID, clusterId).results
        assert nistResults.getRunMetadata().runId == nistRunId

        then:
        "confirm state and evidence of expected controls"
        Map<String, ComplianceResultValue> clusterResults = [:]
        clusterResults << nistResults.clusterResults.controlResultsMap
        assert clusterResults
        def missingControls = []
        for (Control control : controls) {
            if (clusterResults.keySet().contains(control.id)) {
                println "Validating deployment control ${control.id}"
                ComplianceResultValue value = clusterResults.get(control.id)
                assert value.overallState == control.state
                assert value.evidenceList*.message.containsAll(control.evidenceMessages)
            } else {
                missingControls.add(control)
            }
        }
        assert missingControls*.id.size() == 0

        cleanup:
        "undo policy changes"
        for (String policyName : enforcementPolicies) {
            Services.updatePolicyEnforcement(policyName, [PolicyOuterClass.EnforcementAction.UNSET_ENFORCEMENT])
        }
        Services.updatePolicyLifecycleStage(
                "CVSS >= 7",
                [PolicyOuterClass.LifecycleStage.DEPLOY])
        if (policyId) {
            CreatePolicyService.deletePolicy(policyId)
        }
    }

    @Category([BAT])
    def "Verify controls that rely on CIS Benchmarks"() {
        def controls = [
                new Control(
                        "PCI_DSS_3_2:2_2",
                        ["CIS Benchmarks have been run."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "NIST_800_190:4_3_5",
                        ["CIS Benchmarks have been run."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "NIST_800_190:4_4_3",
                        ["CIS Benchmarks have been run."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "NIST_800_190:4_5_1",
                        ["CIS Benchmarks have been run."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
        ]

        given:
        "re-run PCI and HIPAA to make sure they see the run CIS standards"
        def pciRunId = ComplianceManagementService.triggerComplianceRunAndWait(PCI_ID, clusterId)
        ComplianceRunResults pciResults = ComplianceService.getComplianceRunResult(PCI_ID, clusterId).results
        assert pciResults.getRunMetadata().runId == pciRunId

        def nistRunId = ComplianceManagementService.triggerComplianceRunAndWait(NIST_ID, clusterId)
        ComplianceRunResults nistResults = ComplianceService.getComplianceRunResult(NIST_ID, clusterId).results
        assert nistResults.getRunMetadata().runId == nistRunId

        expect:
        "check the CIS based controls for state"
        Map<String, ComplianceResultValue> clusterResults = [:]
        clusterResults << pciResults.clusterResults.controlResultsMap
        clusterResults << nistResults.clusterResults.controlResultsMap
        assert clusterResults
        def missingControls = []
        for (Control control : controls) {
            if (clusterResults.keySet().contains(control.id)) {
                println "Validating cluster control ${control.id}"
                ComplianceResultValue value = clusterResults.get(control.id)
                assert value.overallState == control.state
                assert value.evidenceList*.message.containsAll(control.evidenceMessages)
            } else {
                missingControls.add(control)
            }
        }
        assert missingControls*.id.size() == 0
    }

    @Category([BAT])
    def "Verify controls that checks for fixable CVEs"() {
        def controls = [
                new Control(
                        "PCI_DSS_3_2:6_2",
                        ["Image apollo-dtr.rox.systems/legacy-apps/ssl-terminator:latest has 78 fixed CVEs. " +
                                 "An image upgrade is required."],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "HIPAA_164:306_e",
                        ["Image apollo-dtr.rox.systems/legacy-apps/ssl-terminator:latest has 78 fixed CVEs. " +
                                 "An image upgrade is required."],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
        ]

        given:
        "skip test due to ROX-1336"
        Assume.assumeTrue(Constants.CHECK_CVES_IN_COMPLIANCE)

        and:
        "deploy image with fixable CVEs"
        Deployment cveDeployment = new Deployment()
                .setName("cve-compliance-deployment")
                .setImage("apollo-dtr.rox.systems/legacy-apps/ssl-terminator:latest")
                .addLabel("app", "cve-compliance-deployment")
        orchestrator.createDeployment(cveDeployment)

        and:
        "wait for image to be scanned"
        def start = System.currentTimeMillis()
        ImageOuterClass.ListImage image = ImageService.getImages().find { it.name == cveDeployment.image }
        while (image?.getFixableCves() == 0 && System.currentTimeMillis() - start < 30000) {
            sleep 2000
            image = ImageService.getImages().find { it.name == cveDeployment.image }
        }

        when:
        "trigger compliance runs"
        def pciRunId = ComplianceManagementService.triggerComplianceRunAndWait(PCI_ID, clusterId)
        ComplianceRunResults pciResults = ComplianceService.getComplianceRunResult(PCI_ID, clusterId).results
        assert pciResults.getRunMetadata().runId == pciRunId

        def hipaaRunId = ComplianceManagementService.triggerComplianceRunAndWait(HIPAA_ID, clusterId)
        ComplianceRunResults hipaaResults = ComplianceService.getComplianceRunResult(HIPAA_ID, clusterId).results
        assert hipaaResults.getRunMetadata().runId == hipaaRunId

        then:
        "confirm state and evidence of expected controls"
        Map<String, ComplianceResultValue> clusterResults = [:]
        clusterResults << pciResults.getClusterResults().controlResultsMap
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
        if (cveDeployment?.deploymentUid) {
            orchestrator.deleteDeployment(cveDeployment)
        }
    }

    @Category([BAT])
    def "Verify failed run result"() {
        given:
        "Get Sensor pod name"
        def sensorPod = orchestrator.getSensorContainerName()

        when:
        "trigger compliance run"
        ComplianceManagementServiceOuterClass.ComplianceRun complianceRun =
                ComplianceManagementService.triggerComplianceRun(NIST_ID, clusterId)
        Long startTime = System.currentTimeMillis()

        and:
        "kill sensor"
        orchestrator.deleteContainer(sensorPod, "stackrox")
        while (complianceRun.state != ComplianceManagementServiceOuterClass.ComplianceRun.State.FINISHED &&
                (System.currentTimeMillis() - startTime) < 30000) {
            sleep 1000
            complianceRun = ComplianceManagementService.getRecentRuns(NIST_ID).find { it.id == complianceRun.id }
        }

        then:
        "validate result contains errors"
        ComplianceRunResults results =
                ComplianceService.getComplianceRunResult(NIST_ID, clusterId).results
        assert results != null
        Compliance.ComplianceRunMetadata metadata = results.runMetadata
        assert metadata.clusterId == clusterId
        assert metadata.runId == complianceRun.id
        assert metadata.standardId == NIST_ID

        def numErrors = 0
        for (def ctrlResults : results.clusterResults.controlResultsMap.values()) {
            if (ctrlResults.overallState == Compliance.ComplianceState.COMPLIANCE_STATE_ERROR) {
                numErrors++
            }
        }
        for (def deploymentResults : results.deploymentResultsMap.values()) {
            for (def ctrlResults : deploymentResults.controlResultsMap.values()) {
                if (ctrlResults.overallState == Compliance.ComplianceState.COMPLIANCE_STATE_ERROR) {
                    numErrors++
                }
            }
        }
        for (def nodeResults : results.nodeResultsMap.values()) {
            for (def ctrlResults : nodeResults.controlResultsMap.values()) {
                if (ctrlResults.overallState == Compliance.ComplianceState.COMPLIANCE_STATE_ERROR) {
                    numErrors++
                }
            }
        }
        assert numErrors > 0

        cleanup:
        "wait for sensor to come back up"
        def start = System.currentTimeMillis()
        orchestrator.waitForSensor()
        println "waited ${System.currentTimeMillis() - start}ms for sensor to come back online"
    }
}
