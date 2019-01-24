import groups.BAT
import io.stackrox.proto.storage.Compliance.ComplianceResultValue
import io.stackrox.proto.storage.Compliance.ComplianceRunResults
import io.stackrox.proto.storage.Compliance.ComplianceState
import objects.Control
import objects.Deployment
import objects.Service
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ClusterService
import services.ComplianceManagementService
import services.ComplianceService
import spock.lang.Shared
import v1.ComplianceServiceOuterClass.ComplianceStandardMetadata

class PciComplianceTest extends BaseSpecification {
    @Shared
    private static final PCI_ID = "PCI_DSS_3_2"
    @Shared
    private static final NIST_ID = "NIST_800_190"
    @Shared
    private static final HIPAA_ID = "HIPAA_164"
    @Shared
    private static final Map<String, ComplianceRunResults> BASE_RESULTS = [:]
    @Shared
    private clusterId

    def setupSpec() {
        clusterId = ClusterService.getClusterId()
        /*
        for (ComplianceStandardMetadata standard : ComplianceService.getComplianceStandards()) {
            def runId = ComplianceManagementService.triggerComplianceRunAndWait(standard.id, clusterId)
            ComplianceRunResults results = ComplianceService.getComplianceRunResult(standard.id, clusterId)
            assert runId == results.runMetadata.runId
            BASE_RESULTS.put(standard.id, results)
        }
        */
    }

    @Category([BAT])
    def "Verify checks based on Integrations"() {
        Assume.assumeTrue(false) //always skip for now
        def baseControls = [
                new Control(
                        "PCI_DSS_3_2:6_1",
                        ["An image vulnerability scanner (dtr) is configured"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
                new Control(
                        "NIST_800_190:4_1_1",
                        ["An image vulnerability scanner (dtr) is configured",
                         "Policy that disallows images, with a CVSS score above a threshold, to be deployed not found",
                         "Unable to find a build time policy that is enabled and enforced"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "NIST_800_190:4_1_2",
                        ["'Secure Shell (ssh) Port Exposed' policy is in use",
                         "'Secure Shell Server (sshd) Execution' policy is in use",
                         "'CAP_SYS_ADMIN capability added' policy is not being enforced",
                         "'Privileged Container' policy is not being enforced",
                         "'Linux Group Add Execution' policy is not being enforced",
                         "'CVSS >= 6 and Privileged' policy is not being enforced",
                         "'Linux User Add Execution' policy is not being enforced",
                         "'CVSS >= 6 and Privileged' policy is not being enforced",
                         "'Shellshock: CVE-2014-6271' policy is not being enforced",
                         "'Heartbleed: CVE-2014-0160' policy is not being enforced",
                         "'Apache Struts: CVE-2017-5638' policy is not being enforced",
                         "'CVSS >= 7' policy is not being enforced",
                         "An image vulnerability scanner (dtr) is configured",
                         "Unable to find a build time policy that is enabled and enforced"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "HIPAA_164:306_e",
                        ["An image vulnerability scanner (dtr) is configured"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
        ]
        def controls = [
                new Control(
                        "PCI_DSS_3_2:6_1",
                        ["No image vulnerability scanners are configured"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "NIST_800_190:4_1_1",
                        ["No image vulnerability scanners are configured",
                         "Policy that disallows images, with a CVSS score above a threshold, to be deployed not found",
                         "Unable to find a build time policy that is enabled and enforced"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "NIST_800_190:4_1_2",
                        ["'Secure Shell (ssh) Port Exposed' policy is in use",
                         "'Secure Shell Server (sshd) Execution' policy is in use",
                         "'CAP_SYS_ADMIN capability added' policy is not being enforced",
                         "'Privileged Container' policy is not being enforced",
                         "'Linux Group Add Execution' policy is not being enforced",
                         "'CVSS >= 6 and Privileged' policy is not being enforced",
                         "'Linux User Add Execution' policy is not being enforced",
                         "'CVSS >= 6 and Privileged' policy is not being enforced",
                         "'Shellshock: CVE-2014-6271' policy is not being enforced",
                         "'Heartbleed: CVE-2014-0160' policy is not being enforced",
                         "'Apache Struts: CVE-2017-5638' policy is not being enforced",
                         "'CVSS >= 7' policy is not being enforced",
                         "No image vulnerability scanners are configured",
                         "Unable to find a build time policy that is enabled and enforced"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "HIPAA_164:306_e",
                        ["No image vulnerability scanners are configured"],
                        ComplianceState.COMPLIANCE_STATE_SUCCESS),
        ]

        given:
        "existing compliance run passes"
        Map<String, ComplianceResultValue> clusterResults = [:]
        clusterResults << BASE_RESULTS.get(PCI_ID).getClusterResults().controlResultsMap
        clusterResults << BASE_RESULTS.get(NIST_ID).getClusterResults().controlResultsMap
        assert clusterResults
        for (String control : clusterResults.keySet()) {
            Control c = baseControls.find { it.id == control }
            if (c) {
                ComplianceResultValue value = clusterResults.get(control)
                assert value.overallState == c.success
                for (String evidence : c.evidenceMessages) {
                    assert value.evidenceList.find { it.message == evidence }
                }
            }
        }

        and:
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
        Map<String, ComplianceResultValue> clusterResultsPost = [:]
        clusterResultsPost << pciResults.getClusterResults().controlResultsMap
        clusterResultsPost << nistResults.getClusterResults().controlResultsMap
        clusterResultsPost << hipaaResults.getClusterResults().controlResultsMap
        assert clusterResultsPost
        for (String control : clusterResultsPost.keySet()) {
            Control c = controls.find { it.id == control }
            if (c) {
                ComplianceResultValue value = clusterResultsPost.get(control)
                assert value.overallState == c.success
                for (String evidence : c.evidenceMessages) {
                    assert value.evidenceList.find { it.message == evidence }
                }
            }
        }

        cleanup:
        "re-add image integrations"
        if (removed) {
            dtrId = Services.addDockerTrustedRegistry()
        }
    }

    @Category([BAT])
    def "Verify checks based on Deployments"() {
        Assume.assumeTrue(false) //always skip for now
        def controls = [
                new Control(
                        "PCI_DSS_3_2:1_3_5",
                        ["Deployment uses UDP, which allows data exchange without an established connection"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "PCI_DSS_3_2:1_2_1",
                        ["No ingress network policies apply to this deployment, hence all ingress connections are " +
                                 "allowed",
                         "No egress network policies apply to this deployment, hence all egress connections are " +
                                 "allowed",
                         "Deployment uses host network, which allows it to subvert network policies"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "PCI_DSS_3_2:2_2_1",
                        ["Container compliance-deployment in Deployment is running processes from multiple binaries, " +
                                 "indicating the container is performing multiple tasks"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "PCI_DSS_3_2:1_3_2",
                        ["No ingress network policies apply to the deployment, hence all ingress connections are " +
                                 "allowed",
                         "Deployment uses host network, which allows it to subvert network policies"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "PCI_DSS_3_2:2_2_5",
                        ["Deployment has exposed ports that are not receiving traffic: [80]"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "NIST_800_190:4_5_5",
                        ["Deployment compliance-deployment is using host mounts."],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "NIST_800_190:4_3_3",
                        ["No ingress network policies apply to this deployment, hence all ingress connections are " +
                                 "allowed",
                         "No egress network policies apply to this deployment, hence all egress connections are " +
                                 "allowed",
                         "Deployment uses host network, which allows it to subvert network policies"],
                        ComplianceState.COMPLIANCE_STATE_FAILURE),
                new Control(
                        "NIST_800_190:4_4_2",
                        ["No ingress network policies apply to this deployment, hence all ingress connections are " +
                                 "allowed",
                         "No egress network policies apply to this deployment, hence all egress connections are " +
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
                .addVolName("test", "/tmp")
                .setCommand(["/bin/sh", "-c",])
                .setArgs(["dd if=/dev/zero of=/dev/null & yes"])
        Service service = new Service(deployment)
                .setType(Service.Type.NODEPORT)
        orchestrator.createService(service)
        orchestrator.createDeployment(deployment)
        assert Services.waitForDeployment(deployment)

        when:
        "trigger compliance runs"
        def pciRunId = ComplianceManagementService.triggerComplianceRunAndWait(PCI_ID, clusterId)
        ComplianceRunResults pciResults = ComplianceService.getComplianceRunResult(PCI_ID, clusterId)
        assert pciResults.getRunMetadata().runId == pciRunId

        def nistRunId = ComplianceManagementService.triggerComplianceRunAndWait(NIST_ID, clusterId)
        ComplianceRunResults nistResults = ComplianceService.getComplianceRunResult(NIST_ID, clusterId)
        assert nistResults.getRunMetadata().runId == nistRunId

        then:
        "confirm state and evidence of expected controls"
        Map<String, ComplianceResultValue> deploymentResults = [:]
        deploymentResults << pciResults.getDeploymentResultsMap().get(deployment.deploymentUid).controlResultsMap
        deploymentResults << nistResults.getDeploymentResultsMap().get(deployment.deploymentUid).controlResultsMap
        assert deploymentResults
        for (String control : deploymentResults.keySet()) {
            Control c = controls.find { it.id == control }
            if (c) {
                ComplianceResultValue value = deploymentResults.get(control)
                assert value.overallState == c.success
                for (String evidence : c.evidenceMessages) {
                    assert value.evidenceList.find { it.message == evidence }
                }
            }
        }

        cleanup:
        "remove deployment"
        if (deployment) {
            orchestrator.deleteDeployment(deployment)
        }
        if (service) {
            orchestrator.deleteService(service.name, service.namespace)
        }
    }
}
