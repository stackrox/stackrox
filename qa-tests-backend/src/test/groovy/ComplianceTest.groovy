import static io.stackrox.proto.api.v1.ComplianceServiceOuterClass.ComplianceControl
import static io.stackrox.proto.api.v1.ComplianceServiceOuterClass.ComplianceStandard
import static io.stackrox.proto.storage.RoleOuterClass.Access.READ_WRITE_ACCESS
import static io.stackrox.proto.storage.RoleOuterClass.SimpleAccessScope.newBuilder
import static services.ClusterService.DEFAULT_CLUSTER_NAME
import static util.Helpers.withRetry

import java.nio.charset.StandardCharsets
import java.nio.file.Files
import java.nio.file.Paths
import java.time.Instant
import java.time.ZoneId
import java.time.format.DateTimeFormatter

import com.google.protobuf.util.Timestamps
import com.opencsv.bean.CsvToBean
import com.opencsv.bean.CsvToBeanBuilder
import com.opencsv.bean.HeaderColumnNameTranslateMappingStrategy

import io.stackrox.proto.api.v1.ComplianceManagementServiceOuterClass
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.Compliance
import io.stackrox.proto.storage.Compliance.ComplianceAggregation.Result
import io.stackrox.proto.storage.Compliance.ComplianceAggregation.Scope
import io.stackrox.proto.storage.Compliance.ComplianceResultValue
import io.stackrox.proto.storage.Compliance.ComplianceRunResults
import io.stackrox.proto.storage.Compliance.ComplianceState
import io.stackrox.proto.storage.ImageOuterClass
import io.stackrox.proto.storage.NodeOuterClass.Node
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.PolicyGroup
import io.stackrox.proto.storage.PolicyOuterClass.PolicyValue
import io.stackrox.proto.storage.RoleOuterClass

import common.Constants
import objects.Control
import objects.CsvRow
import objects.Deployment
import objects.GCRImageIntegration
import objects.NetworkPolicy
import objects.NetworkPolicyTypes
import objects.Service
import objects.SlackNotifier
import services.ApiTokenService
import services.BaseService
import services.ClusterService
import services.ComplianceManagementService
import services.ComplianceService
import services.ImageIntegrationService
import services.ImageService
import services.NetworkPolicyService
import services.NodeService
import services.PolicyService
import services.ProcessService
import services.RoleService
import util.Timer

import org.junit.Assume
import spock.lang.IgnoreIf
import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Unroll
import util.Env

class ComplianceTest extends BaseSpecification {
    @Shared
    private static final PCI_ID = "PCI_DSS_3_2"
    @Shared
    private static final NIST_800_190_ID = "NIST_800_190"
    @Shared
    private static final NIST_800_53_ID = "NIST_SP_800_53_Rev_4"
    @Shared
    private static final HIPAA_ID = "HIPAA_164"
    @Shared
    private static final DOCKER_1_2_0_ID = "CIS_Docker_v1_2_0"
    @Shared
    private static final Map<String, ComplianceRunResults> BASE_RESULTS = [:]
    @Shared
    private String clusterId
    @Shared
    private gcrId = ""
    @Shared
    private Map<String, String> standardsByName = [:]
    static final private String COMPLIANCETOKEN = "stackrox-compliance"

    def setupSpec() {
        BaseService.useBasicAuth()

        // Get cluster ID
        clusterId = ClusterService.getClusterId()
        assert clusterId

        // Clear image cache and add gcr
        ImageService.clearImageCaches()
        gcrId = GCRImageIntegration.createDefaultIntegration()

        // Get compliance metadata
        standardsByName = ComplianceService.getComplianceStandards().collectEntries {
            [(it.getName()): it.getId()]
        }

        // Generate baseline compliance runs
        sleep 30000
        def complianceRuns = ComplianceManagementService.triggerComplianceRunsAndWait()
        for (String standard : complianceRuns.keySet()) {
            def runId = complianceRuns.get(standard)
            ComplianceRunResults results = ComplianceService.getComplianceRunResult(standard, clusterId, runId).results
            assert runId == results.runMetadata.runId
            BASE_RESULTS.put(standard, results)
        }
    }

    def cleanupSpec() {
        BaseService.useBasicAuth()
        ImageIntegrationService.deleteImageIntegration(gcrId)
        ImageService.clearImageCaches()

        // Wait for compliance daemonset to be deleted
        Map<String, String> complianceLabels = new HashMap<>()
        complianceLabels.put("com.stackrox.io/service", "compliance")
        assert orchestrator.waitForAllPodsToBeRemoved("stackrox", complianceLabels, 30, 5)
    }

    @Tag("BAT")
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
                        ["Runtime support is enabled (or collector service is running) for cluster "
                                 + DEFAULT_CLUSTER_NAME
                                 + ". Network visualization for active network connections is possible."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS).setType(Control.ControlType.CLUSTER),
                new Control(
                        "HIPAA_164:310_d",
                        ["Runtime support is enabled (or collector service is running) for cluster "
                                + DEFAULT_CLUSTER_NAME
                                + ". Network visualization for active network connections is possible."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS).setType(Control.ControlType.CLUSTER),
                new Control(
                        "HIPAA_164:310_d",
                        ["Runtime support is enabled (or collector service is running) for cluster "
                                 + DEFAULT_CLUSTER_NAME
                                 + ". Network visualization for active network connections is possible."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS).setType(Control.ControlType.CLUSTER),
                new Control(
                        "NIST_SP_800_53_Rev_4:RA_3",
                        ['StackRox is installed in cluster "' + DEFAULT_CLUSTER_NAME +
                                '", and provides continuous risk assessment.'],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS).setType(Control.ControlType.CLUSTER),
        ]
        if (!ClusterService.isAKS()) { // ROX-6993
            List<Node> nodes = NodeService.getNodes()
            if (nodes.size() > 0 && nodes.get(0).containerRuntimeVersion.contains("docker")) {
                staticControls.add(new Control(
                        "CIS_Docker_v1_2_0:2_6",
                        ["Docker daemon is not exposed over TCP"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS).setType(Control.ControlType.NODE))
            } else {
                staticControls.add(new Control(
                        "CIS_Docker_v1_2_0:2_6",
                        ["Node does not use Docker container runtime"],
                        ComplianceState.COMPLIANCE_STATE_SKIP).setType(Control.ControlType.NODE))
            }
        }

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

    @Tag("BAT")
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
            log.info "Verifying aggregate counts for ${standardId}"
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
            run.machineConfigResultsMap.each {
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

            def countPassing = (counts.get(ComplianceState.COMPLIANCE_STATE_SUCCESS) ?: []).size()
            assert result.numPassing == countPassing

            def countFailing = (counts.get(ComplianceState.COMPLIANCE_STATE_FAILURE) ?: []).size() +
                    (counts.get(ComplianceState.COMPLIANCE_STATE_ERROR) ?: []).size()
            assert result.numFailing == countFailing
        }
    }

    @Tag("BAT")
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

    @Tag("BAT")
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

    @Tag("BAT")
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

    @Tag("BAT")
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
            case "Info":
                return ComplianceState.COMPLIANCE_STATE_NOTE
        }
    }

    @Tag("BAT")
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
                [(it.id): ComplianceService.getComplianceStandardDetails(it.id)]
            }

            while (csvUserIterator.hasNext()) {
                CsvRow row = csvUserIterator.next()
                rowNumber++

                def standardId = standardsByName.get(row.standard)
                ComplianceRunResults result = BASE_RESULTS.get(standardId)
                assert result

                // The control name is formatted with `fmt.Sprintf(`=("%s")`, controlName)` in Go.
                // Just undo this to get the name.
                assert row.control.length() > 5
                def normalizedControlName = row.control[3..(row.control.length() - 3)]

                ComplianceControl control = sDetails.get(standardId).controlsList.find {
                    it.name == normalizedControlName
                }
                if (!control) {
                    log.info "Couldn't find ${normalizedControlName} (row " +
                            "was ${row.cluster} ${row.standard} ${row.control}"
                }
                assert control

                ComplianceResultValue value
                switch (row.objectType.toLowerCase()) {
                    case "cluster":
                        value = result.clusterResults.controlResultsMap.find {
                            it.key == control.id
                        }?.value
                        break
                    case "node":
                        value = result.nodeResultsMap.get(
                                result.domain.nodesMap.find { it.value.name == row.objectName }?.key
                        )?.controlResultsMap?.find {
                            it.key == control.id
                        }?.value
                        break
                    case "machineconfig":
                        value = result.machineConfigResultsMap.get(row.objectName).controlResultsMap?.find {
                            it.key == control.id
                        }?.value
                        break
                    default:
                        value = result.deploymentResultsMap.get(
                                result.domain.deploymentsMap.find {
                            it.value.name == row.objectName && it.value.namespace == row.namespace
                                }?.key
                        )?.controlResultsMap?.find {
                            it.key == control.id
                        }?.value
                        break
                }
                if (!value) {
                    log.info "Control: ${control} StandardId: ${standardId}" +
                            "Row: ${row.cluster}, ${row.standard}, ${row.objectType}, ${row.control}, ${row.evidence}"
                    log.info result.clusterResults.controlResultsMap.keySet()
                }
                assert value
                assert convertStringState(row.state) ?
                            convertStringState(row.state) == value.overallState :
                            row.state == "Unknown"
                verifiedRows++
                assert row.controlDescription == control.description
                assert row.cluster == result.domain.cluster.name
                Instant i = Instant.parse(Timestamps.toString(result.runMetadata.finishTimestamp))
                DateTimeFormatter formatter = DateTimeFormatter.ofPattern("EEE, dd MMM yyyy HH:mm:ss 'UTC'")
                        .withZone(ZoneId.of("UTC"))
                assert row.timestamp == formatter.format(i)
            }
            log.info "Verified ${verifiedRows} out of ${rowNumber} total rows"
        } catch (Exception e) {
            log.error("Exception", e)
        }
    }

    @Tag("BAT")
    def "Verify a subset of the checks in nodes were run in each node"() {
        expect:
        "check a subset of the checks run in the compliance pods are present in the results"
        def dockerResults = BASE_RESULTS.get("CIS_Docker_v1_2_0")
        for (ComplianceRunResults.EntityResults nodeResults : dockerResults.getNodeResultsMap().values()) {
            def controlResults = nodeResults.getControlResultsMap()
            assert controlResults.containsKey("CIS_Docker_v1_2_0:1_1_1")
            assert controlResults.containsKey("CIS_Docker_v1_2_0:2_1")
            assert controlResults.containsKey("CIS_Docker_v1_2_0:3_1")
            assert controlResults.containsKey("CIS_Docker_v1_2_0:4_2")
            assert controlResults.containsKey("CIS_Docker_v1_2_0:5_1")
            assert controlResults.containsKey("CIS_Docker_v1_2_0:6_1")
        }

        def kubernetesResults = BASE_RESULTS.get("CIS_Kubernetes_v1_5")
        for (ComplianceRunResults.EntityResults nodeResults : kubernetesResults.getNodeResultsMap().values()) {
            def controlResults = nodeResults.getControlResultsMap()
            assert controlResults.containsKey("CIS_Kubernetes_v1_5:3_1_1")
            assert controlResults.containsKey("CIS_Kubernetes_v1_5:2_1")
            assert controlResults.containsKey("CIS_Kubernetes_v1_5:4_2_1")
            assert controlResults.containsKey("CIS_Kubernetes_v1_5:1_2_5")
            assert controlResults.containsKey("CIS_Kubernetes_v1_5:1_1_1")
            assert controlResults.containsKey("CIS_Kubernetes_v1_5:5_5_1")
            assert controlResults.containsKey("CIS_Kubernetes_v1_5:5_6_1")
            assert controlResults.containsKey("CIS_Kubernetes_v1_5:5_3_1")
            assert controlResults.containsKey("CIS_Kubernetes_v1_5:5_2_1")
            assert controlResults.containsKey("CIS_Kubernetes_v1_5:5_1_1")
            assert controlResults.containsKey("CIS_Kubernetes_v1_5:5_4_1")
            assert controlResults.containsKey("CIS_Kubernetes_v1_5:4_1_1")
        }

        def nistResults = BASE_RESULTS.get("NIST_800_190")
        for (ComplianceRunResults.EntityResults nodeResults : nistResults.getNodeResultsMap().values()) {
            def controlResults = nodeResults.getControlResultsMap()
            assert controlResults.containsKey("NIST_800_190:4_2_1")
        }
    }

    @Tag("BAT")
    def "Verify per-node cluster checks generate correct results when there is a master node"() {
        given:
        "a control result which should only be returned from a master node"
        def kubernetesResults = BASE_RESULTS.get("CIS_Kubernetes_v1_5")
        def clusterResults = kubernetesResults.getClusterResults().getControlResultsMap()
        // pick any check which should have a result when a master node exists and a note when no master node exists,
        // for example Kubernetes 1.2.32
        def controlResult = clusterResults["CIS_Kubernetes_v1_5:1_2_32"]

        expect:
        "the control result has a pass/fail result when run in an environment with a master node"
        List<objects.Node> orchNodes = orchestrator.getNodeDetails()
        def hasMaster = false
        for (objects.Node node : orchNodes) {
            for (String label : node.getLabels().keySet()) {
                if (label == "node-role.kubernetes.io/master" || label == "node-role.kubernetes.io/control-plane") {
                    hasMaster = true
                    break
                }
            }
            if (hasMaster) {
                break
            }
        }

        def overallState = controlResult.getOverallState()
        assert controlResult.evidenceList.size() >= 1
        def evidence = controlResult.evidenceList[0]
        if (hasMaster) {
            if (overallState == ComplianceState.COMPLIANCE_STATE_NOTE) {
                // openshift-crio has a master node but does not run the master API process so it should note that the
                // master API process does not exist.
                assert evidence.message.contains("not found on host")
            } else {
                // kops has a master node and DOES run the master API process so it should succeed or fail
                assert controlResult.getOverallState() == ComplianceState.COMPLIANCE_STATE_SUCCESS ||
                        controlResult.getOverallState() == ComplianceState.COMPLIANCE_STATE_FAILURE
            }
        } else {
            // When there is no master node we should make sure the note is the default generated in Central
            assert overallState == ComplianceState.COMPLIANCE_STATE_NOTE
            assert evidence.message.contains("No evidence was received for this check")
        }
    }

    @Tag("BAT")
    def "Verify Compliance aggregations with caching"() {
        given:
        "get compliance aggregation results"
        List<Result> aggResults = ComplianceService.getAggregatedResults(Scope.CONTROL, [Scope.CLUSTER, Scope.STANDARD])

        when:
        "getting the same results again"
        List<Result> sameAggResults = ComplianceService.getAggregatedResults(
                Scope.CONTROL,
                [Scope.CLUSTER, Scope.STANDARD]
        )

        then:
        "both result sets should be the same"
        aggResults.size() == sameAggResults.size()
        for (int i = 0; i < aggResults.size(); ++i) {
            def aggResult = aggResults[i]
            def sameResult = sameAggResults[i]
            assert aggResult == sameResult
        }
    }

    /*
    **  Remaining tests in the spec trigger new compliance runs. If you are adding tests that do not require a fresh
    **  compliance run, add them above this comment and use the compliance data in BASE_RESULTS.
    */

    @Tag("BAT")
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
                        ["At least one enabled policy has a notifier configured."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "HIPAA_164:314_a_2_i_c",
                        ["At least one enabled policy has a notifier configured."],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "NIST_SP_800_53_Rev_4:IR_6_(1)",
                        ["Policy \"Ubuntu Package Manager Execution\" is a runtime policy, set to send notifications"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
        ]

        given:
        "remove image integrations"
        def gcrRemoved = ImageIntegrationService.deleteImageIntegration(gcrId)
        ImageIntegrationService.deleteStackRoxScannerIntegrationIfExists()

        and:
        "add notifier integration"
        SlackNotifier notifier = new SlackNotifier()
        notifier.createNotifier()
        def originalUbuntuPackageManagementPolicy = Services.getPolicyByName("Ubuntu Package Manager Execution")
        assert originalUbuntuPackageManagementPolicy
        def updatedPolicy = PolicyOuterClass.Policy.newBuilder(originalUbuntuPackageManagementPolicy).
                addNotifiers(notifier.id).build()
        Services.updatePolicy(updatedPolicy)

        when:
        "trigger compliance runs"
        def pciResults = ComplianceService.triggerComplianceRunAndWaitForResult(PCI_ID, clusterId)
        def nist800190Results = ComplianceService.triggerComplianceRunAndWaitForResult(NIST_800_190_ID, clusterId)
        def hipaaResults = ComplianceService.triggerComplianceRunAndWaitForResult(HIPAA_ID, clusterId)
        def nist80053Results = ComplianceService.triggerComplianceRunAndWaitForResult(NIST_800_53_ID, clusterId)

        then:
        "confirm state and evidence of expected controls"
        Map<String, ComplianceResultValue> clusterResults = [:]
        clusterResults << pciResults.getClusterResults().controlResultsMap
        clusterResults << nist800190Results.getClusterResults().controlResultsMap
        clusterResults << hipaaResults.getClusterResults().controlResultsMap
        clusterResults << nist80053Results.getClusterResults().controlResultsMap
        assert clusterResults
        def missingControls = []
        for (Control control : controls) {
            if (clusterResults.keySet().contains(control.id)) {
                log.info "Validating ${control.id}"
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
        if (gcrRemoved) {
            gcrId = GCRImageIntegration.createDefaultIntegration()
        }
        notifier.deleteNotifier()
        Services.updatePolicy(originalUbuntuPackageManagementPolicy)
        ImageIntegrationService.addStackroxScannerIntegration()
    }

    @Tag("BAT")
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
                .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-15-4-alpine")
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
        Set<String> receivedProcessPaths = []
        Timer t = new Timer(30, 2)
        while (t.IsValid()) {
            receivedProcessPaths = ProcessService.getUniqueProcessPaths(deployment.deploymentUid)
            if (receivedProcessPaths.size() > 1) {
                break
            }
            log.info "Didn't find all the expected processes, retrying..."
        }
        assert receivedProcessPaths.size() > 1

        when:
        "trigger compliance runs"
        def pciResults = ComplianceService.triggerComplianceRunAndWaitForResult(PCI_ID, clusterId)
        def nistResults = ComplianceService.triggerComplianceRunAndWaitForResult(NIST_800_190_ID, clusterId)
        def hipaaResults = ComplianceService.triggerComplianceRunAndWaitForResult(HIPAA_ID, clusterId)

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
                log.info "Validating deployment control ${control.id}"
                ComplianceResultValue value = deploymentResults.get(control.id)
                assert value.overallState == control.state
                assert value.evidenceList*.message.containsAll(control.evidenceMessages)
            } else {
                missingControls.add(control)
            }
        }
        assert missingControls.size() == 0

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

    @Tag("BAT")
    def "Verify checks based on Policies"() {
        def controls = [
                new Control(
                        "NIST_800_190:4_1_1",
                        ["At least one build-stage policy is enabled and enforced that " +
                                 "disallows images with a critical vulnerability",
                         "At least one policy in lifecycle stage \"BUILD\" is enabled and enforced",
                         "Cluster has an image scanner in use"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "NIST_800_190:4_1_2",
                        ["Policies are in place to detect and enforce \"Privileges\" category issues.",
                         "At least one policy in lifecycle stage \"BUILD\" is enabled and enforced",
                         "Cluster has an image scanner in use"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "NIST_800_190:4_1_4",
                        ["At least one policy is enabled and enforced that detects secrets in environment variables"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "NIST_800_190:4_2_2",
                        ["Policy that disallows old images to be deployed is enabled and enforced",
                         "Policy that disallows images with tag 'latest' to be deployed is enabled and enforced"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "NIST_SP_800_53_Rev_4:CM_2",
                        ["At least one policy in lifecycle stage \"DEPLOY\" is enabled"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "NIST_SP_800_53_Rev_4:CM_3",
                        ["At least one policy in lifecycle stage \"DEPLOY\" is enabled and enforced"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "NIST_SP_800_53_Rev_4:IR_4_(5)",
                        ["At least one policy in lifecycle stage \"RUNTIME\" is enabled and enforced"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),

        ]
        def enforcementPolicies = [
                "Fixable Severity at least Important",
                "Privileged Container",
                "90-Day Image Age",
                "Latest tag",
                "Ubuntu Package Manager Execution",
                "Environment Variable Contains Secret",
        ]
        Map<String, List<PolicyOuterClass.EnforcementAction>> priorEnforcement = [:]

        given:
        "update policies"
        Services.updatePolicyLifecycleStage(
                "Fixable Severity at least Important",
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
            def prior = Services.updatePolicyEnforcement(policyName, enforcements)
            priorEnforcement.put(policyName, prior)
        }
        def policyGroup = PolicyGroup.newBuilder()
                .setFieldName("Environment Variable")
                .setBooleanOperator(PolicyOuterClass.BooleanOperator.AND)
        policyGroup.addAllValues([PolicyValue.newBuilder().setValue(".*SECRET.*=.*").build()])

        def policyId = PolicyService.createNewPolicy(PolicyOuterClass.Policy.newBuilder()
                .setName("XYZ Compliance Secrets")
                .setDescription("Test Secrets in Compliance")
                .setRationale("Test Secrets in Compliance")
                .addLifecycleStages(PolicyOuterClass.LifecycleStage.DEPLOY)
                .addEnforcementActions(PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT)
                .addCategories("Image Assurance")
                .setDisabled(false)
                .setSeverityValue(2)
                .addPolicySections(
                        PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(policyGroup.build()).build())
                .build())

        when:
        "trigger compliance runs"
        def nist800190Results = ComplianceService.triggerComplianceRunAndWaitForResult(NIST_800_190_ID, clusterId)
        def nist80053Results = ComplianceService.triggerComplianceRunAndWaitForResult(NIST_800_53_ID, clusterId)

        then:
        "confirm state and evidence of expected controls"
        Map<String, ComplianceResultValue> clusterResults = [:]
        clusterResults << nist800190Results.clusterResults.controlResultsMap
        clusterResults << nist80053Results.clusterResults.controlResultsMap
        assert clusterResults
        def missingControls = []
        for (Control control : controls) {
            if (clusterResults.keySet().contains(control.id)) {
                log.info "Validating deployment control ${control.id}"
                ComplianceResultValue value = clusterResults.get(control.id)
                assert value.overallState == control.state
                assert value.evidenceList*.message.containsAll(control.evidenceMessages)
            } else {
                missingControls.add(control)
            }
        }
        assert missingControls.size() == 0

        cleanup:
        "undo policy changes"
        for (String policyName : enforcementPolicies) {
            Services.updatePolicyEnforcement(policyName, priorEnforcement.get(policyName))
        }
        Services.updatePolicyLifecycleStage(
                "Fixable CVSS >= 7",
                [PolicyOuterClass.LifecycleStage.DEPLOY])
        if (policyId) {
            PolicyService.deletePolicy(policyId)
        }
    }

    @Tag("BAT")
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
        def pciResults = ComplianceService.triggerComplianceRunAndWaitForResult(PCI_ID, clusterId)
        def nistResults = ComplianceService.triggerComplianceRunAndWaitForResult(NIST_800_190_ID, clusterId)

        expect:
        "check the CIS based controls for state"
        Map<String, ComplianceResultValue> clusterResults = [:]
        clusterResults << pciResults.clusterResults.controlResultsMap
        clusterResults << nistResults.clusterResults.controlResultsMap
        assert clusterResults
        def missingControls = []
        for (Control control : controls) {
            if (clusterResults.keySet().contains(control.id)) {
                log.info "Validating cluster control ${control.id}"
                ComplianceResultValue value = clusterResults.get(control.id)
                assert value.overallState == control.state
                assert value.evidenceList*.message.containsAll(control.evidenceMessages)
            } else {
                missingControls.add(control)
            }
        }
        assert missingControls*.id.size() == 0
    }

    @Unroll
    @Tag("BAT")
    @IgnoreIf({ true }) // ROX-12461 The compliance operator tests are not working as expected
    def "Verify Compliance Operator aggregation results on OpenShift for machine configs #standard"() {
        Assume.assumeTrue(ClusterService.isOpenShift4())

        given:
        "get compliance aggregation results"
        log.info "Getting compliance results for ${standard}"
        ComplianceRunResults run = BASE_RESULTS.get(standard)

        expect:
        "compare"

        // We shouldn't have more than two machine config maps as we only have the roles master/worker
        def machineConfigsWithResults = 0
        def numErrors = 0
        for (def entry in run.machineConfigResultsMap) {
            log.info "Found machine config ${entry.key} with ${entry.value.controlResultsMap.size()} results"
            if (entry.value.controlResultsMap.size()  > 0) {
                machineConfigsWithResults++
            }
            for (def ctrlResults : entry.value.controlResultsMap.values()) {
                if (ctrlResults.overallState == Compliance.ComplianceState.COMPLIANCE_STATE_ERROR) {
                    numErrors++
                }
            }
        }
        assert numErrors == 0
        assert machineConfigsWithResults == 2

        where:
        "Data inputs are: "
        standard                     | _
        "ocp4-cis-node"              | _
        "rhcos4-moderate"            | _
        "rhcos4-moderate-modified"   | _
    }

    @Tag("BAT")
    @IgnoreIf({ true }) // ROX-12461 The compliance operator tests are not working as expected
    def "Verify Tailored Profile does not have evidence for disabled rule"() {
        Assume.assumeTrue(ClusterService.isOpenShift4())

        given:
        "get compliance aggregation results"
        log.info "Getting compliance results for rhcos4-moderate-modified"
        ComplianceRunResults run = BASE_RESULTS.get("rhcos4-moderate-modified")

        expect:
        "compare"

        // We shouldn't have more than two machine config maps as we only have the roles master/worker
        def machineConfigsWithResults = 0
        def numErrors = 0
        for (def entry in run.machineConfigResultsMap) {
            log.info "Found machine config ${entry.key} with ${entry.value.controlResultsMap.size()} results"
            if (entry.value.controlResultsMap.size()  > 0) {
                machineConfigsWithResults++
            }
            assert !entry.value.controlResultsMap.keySet().contains(
                    "rhcos4-moderate-modified:usbguard-allow-hid-and-hub")
        }
        assert numErrors == 0
        assert machineConfigsWithResults == 2
    }

    @Tag("BAT")
    @IgnoreIf({ true }) // ROX-12461 The compliance operator tests are not working as expected
    def "Verify Compliance Operator aggregation results on OpenShift for cluster results"() {
        Assume.assumeTrue(ClusterService.isOpenShift4())

        given:
        "get compliance aggregation results"
        log.info "Getting compliance results for ocp4-cis"
        ComplianceRunResults run = BASE_RESULTS.get("ocp4-cis")

        expect:
        "compare"

        def numErrors = 0
        assert run.clusterResults.controlResultsMap.size() > 0
        for (def ctrlResults : run.clusterResults.controlResultsMap.values()) {
            if (ctrlResults.overallState == Compliance.ComplianceState.COMPLIANCE_STATE_ERROR) {
                numErrors++
            }
        }
        assert numErrors == 0
    }

    @Tag("BAT")
    def "Verify controls that checks for fixable CVEs"() {
        def controls = [
                new Control(
                        "PCI_DSS_3_2:6_2",
                        ["Image quay.io/rhacs-eng/qa-multi-arch:nginx-1.12 has \\d{2}\\d+ fixed CVEs. " +
                                 "An image upgrade is required."],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "HIPAA_164:306_e",
                        ["Image quay.io/rhacs-eng/qa-multi-arch:nginx-1.12 has \\d{2}\\d+ fixed CVEs. " +
                                 "An image upgrade is required."],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
        ]

        given:
        "deploy image with fixable CVEs"
        Deployment cveDeployment = new Deployment()
                .setName("cve-compliance-deployment")
                .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1.12")
                .addLabel("app", "cve-compliance-deployment")
        orchestrator.createDeployment(cveDeployment)

        and:
        "wait for image to be scanned"
        def imageQuery = SearchServiceOuterClass.RawQuery.newBuilder()
                .setQuery("Deployment ID:${cveDeployment.deploymentUid}")
                .build()

        def timer = new Timer(15, 2)
        ImageOuterClass.ListImage image = null

        while (!image?.fixableCves && timer.IsValid()) {
            log.info "Image not found or not scanned: ${image}"
            image = ImageService.getImages(imageQuery).find { it.name == cveDeployment.image }
        }
        assert image?.fixableCves

        log.info "Found scanned image ${image}"

        when:
        "trigger compliance runs"
        def pciResults = ComplianceService.triggerComplianceRunAndWaitForResult(PCI_ID, clusterId)
        def hipaaResults = ComplianceService.triggerComplianceRunAndWaitForResult(HIPAA_ID, clusterId)

        then:
        "confirm state and evidence of expected controls"
        Map<String, ComplianceResultValue> clusterResults = [:]
        clusterResults << pciResults.getClusterResults().controlResultsMap
        clusterResults << hipaaResults.getClusterResults().controlResultsMap
        assert clusterResults
        def missingControls = []
        for (Control control : controls) {
            if (clusterResults.keySet().contains(control.id)) {
                log.info "Validating ${control.id}"
                ComplianceResultValue value = clusterResults.get(control.id)
                assert value.overallState == control.state

                assert value.evidenceList.findAll { it.message.matches(control.evidenceMessages.first()) }.size() > 0
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

    @Tag("SensorBounceNext")
    def "Verify failed run result"() {
        // This seems to be using an auth token for some reason.  Explicitly specify basic auth.
        BaseService.useBasicAuth()
        expect:
        "errors when the sensor is killed during a compliance run"
        def numErrors = 0
        def curIteration = 0
        while (curIteration < 3 && numErrors == 0) {
            if (curIteration > 0) {
                // Make sure the sensor is available.  If this is a retry it could still be down.
                orchestrator.waitForSensor()
            }
            curIteration++
            // Get the sensor pod name
            def sensorPod = orchestrator.getSensorContainerName()
            // Trigger the compliance run
            def complianceRuns = ComplianceManagementService.triggerComplianceRuns(NIST_800_190_ID, clusterId)
            def complianceRun = complianceRuns.get(0)

            // Kill the sensor and wait for the compliance run to complete
            orchestrator.deleteContainer(sensorPod, "stackrox")
            Timer t = new Timer(30, 1)
            while (complianceRun.state != ComplianceManagementServiceOuterClass.ComplianceRun.State.FINISHED &&
                    t.IsValid()) {
                def recentRuns = ComplianceManagementService.getRecentRuns(NIST_800_190_ID)
                complianceRun = recentRuns.find { it.id == complianceRun.id }
            }

            // Check whether there were errors
            ComplianceRunResults results =
                    ComplianceService.getComplianceRunResult(NIST_800_190_ID, clusterId).results
            assert results != null
            Compliance.ComplianceRunMetadata metadata = results.runMetadata
            assert metadata.clusterId == clusterId
            assert metadata.runId == complianceRun.id
            assert metadata.standardId == NIST_800_190_ID

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
        }
        assert numErrors > 0

        cleanup:
        "wait for sensor to come back up"
        def start = System.currentTimeMillis()
        orchestrator.waitForSensor()
        log.info "waited ${System.currentTimeMillis() - start}ms for sensor to come back online"
    }

    @Tag("BAT")
    def "Verify Docker 5_6, no SSH processes"() {
        def deployment = new Deployment()
                .setName("triggerssh")
                .setImage("quay.io/rhacs-eng/qa-multi-arch:fail-compliance-ssh")

        given:
        "create a deployment which forces the ssh check to fail"
        orchestrator.createDeployment(deployment)
        assert Services.waitForDeployment(deployment)

        and:
        "create an expected control result"
        def control = new Control(
                "CIS_Docker_v1_2_0:5_6",
                [],
                ComplianceState.COMPLIANCE_STATE_FAILURE)

        and:
        "verify deployment fully detected"
        Set<String> receivedProcessPaths = []
        def foundSSHProcess = false
        Timer t = new Timer(30, 2)
        while (t.IsValid()) {
            receivedProcessPaths = ProcessService.getUniqueProcessPaths(deployment.deploymentUid)
            for (String path : receivedProcessPaths) {
                if (path.contains("ssh")) {
                    foundSSHProcess = true
                    break
                }
            }
            log.info "Didn't find an SSH processes, retrying..."
        }
        assert foundSSHProcess

        and:
        "trigger compliance runs"
        def dockerResults = ComplianceService.triggerComplianceRunAndWaitForResult(DOCKER_1_2_0_ID, clusterId)

        expect:
        "check the SSH control for a failed for state"

        def results = dockerResults.getDeploymentResultsMap()
        assert results
        assert results.containsKey(deployment.getDeploymentUid())
        def controlResultsMap = results[deployment.getDeploymentUid()].getControlResultsMap()
        assert controlResultsMap
        assert controlResultsMap.containsKey(control.id)
        ComplianceResultValue value = controlResultsMap.get(control.id)
        assert value.overallState == control.state
        assert value.evidenceList*.message.any { msg -> msg =~ /has ssh process running/ }

        cleanup:
        "remove the deployment we created"
        orchestrator.deleteDeployment(deployment)
    }

    @Tag("BAT")
    def "Verify Compliance aggregation cache cleared after each compliance run"() {
        // This seems to be using an auth token for some reason.  Explicitly specify basic auth.
        BaseService.useBasicAuth()
        given:
        "get compliance aggregation results"
        List<Result> aggResults = ComplianceService.getAggregatedResults(Scope.CONTROL, [Scope.CLUSTER, Scope.STANDARD])

        when:
        "starting a new cluster and re-running compliance"
        def otherClusterName = "aNewCluster"
        ClusterService.createCluster(otherClusterName, "stackrox/main:latest", "central.stackrox:443")
        withRetry(10, 2) {
            def clusters = ClusterService.getClusters()
            assert clusters.size() > 1
        }
        ComplianceManagementService.triggerComplianceRunsAndWait()
        List<Result> nextAggResults = ComplianceService.getAggregatedResults(
                Scope.CONTROL,
                [Scope.CLUSTER, Scope.STANDARD]
        )

        then:
        "the result sets should have different lengths"
        aggResults.size() != nextAggResults.size()

        cleanup:
        "delete the extra cluster"
        ClusterService.deleteCluster(ClusterService.getClusterId(otherClusterName))
    }

    @Tag("BAT")
    def "Verify ComplianceRuns with SAC on clusters with wildcard"() {
        def otherClusterName = "disallowedCluster"

        given:
        "Create access scope and test role"
        def remoteStackroxAccessScope = RoleService.createAccessScope(newBuilder()
                .setName(UUID.randomUUID().toString())
                .setRules(RoleOuterClass.SimpleAccessScope.Rules.newBuilder()
                        .addIncludedNamespaces(RoleOuterClass.SimpleAccessScope.Rules.Namespace.newBuilder()
                                .setClusterName(DEFAULT_CLUSTER_NAME)
                                .setNamespaceName("stackrox")))
                .build())
        String testRole = RoleService.createRoleWithScopeAndPermissionSet(
                "Compliance Test Automation Role " + UUID.randomUUID(),
                remoteStackroxAccessScope.id, [
                "Access"                    : READ_WRITE_ACCESS,
                "Administration"            : READ_WRITE_ACCESS,
                "Detection"                 : READ_WRITE_ACCESS,
                "Integration"               : READ_WRITE_ACCESS,
                "WorkflowAdministration"    : READ_WRITE_ACCESS,
                "Cluster"                   : READ_WRITE_ACCESS,
                "Compliance"                : READ_WRITE_ACCESS,
                "Node"                      : READ_WRITE_ACCESS,
        ]).name

        "Enable SAC token and add other cluster"
        ClusterService.createCluster(otherClusterName, "stackrox/main:latest", "central.stackrox:443")
        def token = ApiTokenService.generateToken(COMPLIANCETOKEN, testRole, "None")
        BaseService.useApiToken(token.token)

        when:
        "trigger wildcard compliance run"
        def complianceRuns = ComplianceManagementService.triggerComplianceRunsAndWait("*", "*")

        then:
        "check results under SAC"
        assert complianceRuns.keySet().size() > 0
        for (String standard : complianceRuns.keySet()) {
            def runId = complianceRuns.get(standard)
            ComplianceRunResults results = ComplianceService.getComplianceRunResult(standard, clusterId, runId).results
            assert runId == results.runMetadata.runId
        }

        cleanup:
        "revert to basic auth and delete extra cluster"
        BaseService.useBasicAuth()
        ClusterService.deleteCluster(ClusterService.getClusterId(otherClusterName))
        RoleService.deleteRole(testRole)
    }
}
