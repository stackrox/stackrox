import static Services.checkForNoViolations
import static Services.waitForViolation
import static util.Helpers.withRetry

import io.stackrox.proto.api.v1.PolicyServiceOuterClass.DryRunResponse
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.DeploymentOuterClass
import io.stackrox.proto.storage.NodeOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import io.stackrox.proto.storage.Rbac
import io.stackrox.proto.storage.ScopeOuterClass.Scope

import common.Constants
import objects.Deployment
import objects.K8sPolicyRule
import objects.K8sRole
import objects.K8sRoleBinding
import objects.K8sServiceAccount
import objects.K8sSubject
import objects.Service
import objects.Volume
import services.ClusterService
import services.ImageService
import services.NodeService
import services.PolicyService

import org.junit.Assume
import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Unroll
import util.Env

class PolicyConfigurationTest extends BaseSpecification {
    static final private String DEPLOYMENTNGINX = "deploymentnginx"
    static final private String DNS = "qaapachedns"
    static final private String STRUTS = "qadefpolstruts"
    static final private String DEPLOYMENTNGINX_LB = "deploymentnginx-lb"
    static final private String DEPLOYMENTNGINX_NP = "deploymentnginx-np"
    static final private String DEPLOYMENT_RBAC = "deployment-rbac"
    static final private String SERVICE_ACCOUNT_NAME = "policy-config-sa"
    static final private String NGINX_LATEST_WITH_DIGEST_NAME = "nginx-1-12-1-with-tag-and-digest"
    static final private String NGINX_LATEST_NAME = "nginx-latest"
    private static final String CLUSTER_ROLE_NAME = "policy-config-role"

    static final private Integer WAIT_FOR_VIOLATION_TIMEOUT = isRaceBuild() ? 450 : 90

    private static final K8sServiceAccount NEW_SA = new K8sServiceAccount(
            name: SERVICE_ACCOUNT_NAME,
            namespace: Constants.ORCHESTRATOR_NAMESPACE)

    private static final K8sRole NEW_CLUSTER_ROLE =
            new K8sRole(name: CLUSTER_ROLE_NAME, clusterRole: true)

    private static final K8sRoleBinding NEW_CLUSTER_ROLE_BINDING =
            new K8sRoleBinding(NEW_CLUSTER_ROLE, [new K8sSubject(NEW_SA)])

    static final private List<DeploymentOuterClass.PortConfig.ExposureLevel> EXPOSURE_VALUES =
            [DeploymentOuterClass.PortConfig.ExposureLevel.NODE,
             DeploymentOuterClass.PortConfig.ExposureLevel.EXTERNAL]
    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                    .setName(DEPLOYMENTNGINX)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1.12")
                    .addPort(22, "TCP")
                    .setEnv(["CLUSTER_NAME": "main"])
                    .addLabel("app", "test")
                    .setPrivilegedFlag(true)
                    .addLimits("cpu", "0")
                    .addLimits("memory", "0")
                    .addRequest("memory", "0")
                    .addRequest("cpu", "0")
                    .addVolume(new Volume(name: "test-writable-volumemount",
                            hostPath: true,
                            mountPath: "/tmp"))
                    .addVolume(new Volume(name: "test-writable-volume",
                            hostPath: false,
                            mountPath: "/tmp/test")),
            new Deployment()
                    .setName(STRUTS)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:struts-app")
                    .addLabel("app", "test"),
            new Deployment()
                    .setName(DNS)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:apache-dns")
                    .addLabel("app", "test"),
            new Deployment()
                    .setName(DEPLOYMENTNGINX_LB)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1.12")
                    .addPort(22, "TCP")
                    .addAnnotation("test", "annotation")
                    .setEnv(["CLUSTER_NAME": "main"])
                    .addLabel("app", "test")
                    .setCreateLoadBalancer(
                        !(Env.REMOTE_CLUSTER_ARCH == "ppc64le" || Env.REMOTE_CLUSTER_ARCH == "s390x"))
                    .setExposeAsService(true),
            new Deployment()
                    .setName(DEPLOYMENTNGINX_NP)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1.12")
                    .addPort(22, "TCP")
                    .addAnnotation("test", "annotation")
                    .setEnv(["CLUSTER_NAME": "main"])
                    .addLabel("app", "test"),
            new Deployment()
                    .setName(DEPLOYMENT_RBAC)
                    .setNamespace(Constants.ORCHESTRATOR_NAMESPACE)
                    .setServiceAccountName(SERVICE_ACCOUNT_NAME)
                    .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-15-4-alpine")
                    .setSkipReplicaWait(true),
    ]

    static final private Deployment NGINX_WITH_DIGEST = new Deployment()
            .setName(NGINX_LATEST_WITH_DIGEST_NAME)
            .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1.21.1" +
                "@sha256:a05b0cdd4fc1be3b224ba9662ebdf98fe44c09c0c9215b45f84344c12867002e")
            .setCommand(["sleep", "60000"])
            .setSkipReplicaWait(false)

    static final private Deployment NGINX_LATEST = new Deployment()
            .setName(NGINX_LATEST_NAME)
            .setImage("quay.io/rhacs-eng/qa-multi-arch:latest" +
                "@sha256:a05b0cdd4fc1be3b224ba9662ebdf98fe44c09c0c9215b45f84344c12867002e")
            .setCommand(["sleep", "60000"])
            .setSkipReplicaWait(false)

    static final private Service NPSERVICE =
            new Service(DEPLOYMENTS.find { it.name == DEPLOYMENTNGINX_NP })
            .setType(Service.Type.NODEPORT)

    @Shared
    private String containerRuntimeVersion

    def setupSpec() {
        NEW_CLUSTER_ROLE.setRules([new K8sPolicyRule(resources: ["nodes"], apiGroups: [""], verbs: ["list"])])
        orchestrator.createServiceAccount(NEW_SA)
        orchestrator.createClusterRole(NEW_CLUSTER_ROLE)
        orchestrator.createClusterRoleBinding(NEW_CLUSTER_ROLE_BINDING)
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deploymentId : DEPLOYMENTS) {
            assert Services.waitForDeployment(deploymentId)
        }
        orchestrator.createService(NPSERVICE)
        List<NodeOuterClass.Node> nodes = NodeService.getNodes()
        assert nodes.size() > 0, "should be able to getNodes"
        containerRuntimeVersion = nodes.get(0).containerRuntimeVersion
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
        orchestrator.deleteService(NPSERVICE.name, NPSERVICE.namespace)
        orchestrator.deleteClusterRoleBinding(NEW_CLUSTER_ROLE_BINDING)
        orchestrator.deleteClusterRole(NEW_CLUSTER_ROLE)
        orchestrator.deleteServiceAccount(NEW_SA)
    }

    @Tag("BAT")
    @Tag("PZ")
    def "Verify name violations with same ID as existing image are still triggered"() {
        given:
        "Create a busybox deployment has same ID as latest"
        orchestrator.createDeployment(NGINX_WITH_DIGEST)

        when:
        withRetry(30, 2) {
            def image = ImageService.getImage(
                    "sha256:a05b0cdd4fc1be3b224ba9662ebdf98fe44c09c0c9215b45f84344c12867002e")
            assert image != null
        }

        and:
        "Run busybox latest with same digest as previous image"
        orchestrator.createDeployment(NGINX_LATEST)

        then:
        "Ensure that the latest tag violation shows up"
        def hasViolation =
                waitForViolation(NGINX_LATEST_NAME, "Latest Tag", WAIT_FOR_VIOLATION_TIMEOUT)
        log.info "Has violation ${hasViolation}"
        assert hasViolation

        cleanup:
        "Remove the deployments"
        orchestrator.deleteDeployment(NGINX_WITH_DIGEST)
        orchestrator.deleteDeployment(NGINX_LATEST)
    }

    @Tag("BAT")
    @Tag("PZ")
    def "Verify lastUpdated field is updated correctly for policy - ROX-3971 production bug"() {
        given:
        "Create a copy of a Latest Tag"
        PolicyOuterClass.Policy.Builder policy = Services.getPolicyByName("Latest tag").toBuilder()
        def name = policy.name + new Random().nextInt(2000)
        policy.setName(name)
                .setId("") // set ID to empty so that a new policy is created and not overwrite the original latest tag
                .build()
        def policyId = PolicyService.createNewPolicy(policy.build())
        assert policyId != null
        when:
        "Update a policy description"
        def description = "Test image tag " + new Random().nextInt(4000)
        Policy updatedPolicy = Services.getPolicyByName(name).toBuilder()
                .setDescription(description)
                .build()
        long beforeTime = System.currentTimeMillis() / 1000L
        Services.updatePolicy(updatedPolicy)
        sleep(2000)
        long afterTime = System.currentTimeMillis() / 1000L
        Policy policy1 = Services.getPolicy(policyId)
        then:
        "Check the last_updated value is updated correctly"
        assert afterTime > beforeTime
        assert policy1.description == description
        assert policy1.lastUpdated.seconds >= beforeTime && policy1.lastUpdated.seconds <= afterTime
        cleanup:
        "Remove the policy"
        policyId == null ?: PolicyService.deletePolicy(policyId)
    }

    @Unroll
    @Tag("BAT")
    @Tag("SMOKE")
    @Tag("PZ")
    def "Verify policy configuration #policyName can be triggered"() {
        Assume.assumeTrue(canRun == null || canRun())

        when:
        "Image Scan cache is cleared if required"
        if (requireFreshScan) {
            // If a test requires accurate scan results, then delete the image from DB so that it fetches a fresh scan.
            // A fresh scan might be required because other tests in the suite could've run a scan on the same image,
            // and we don't want those results to taint this test
            // TODO: Find a direct way to clear the cache than just forcing a scan
            def dep = DEPLOYMENTS.find { it.getName() == depname }
            assert dep != null

            log.info "Deleting image ${dep.getImage()} from DB"
            ImageService.deleteImages(
                    SearchServiceOuterClass.RawQuery.newBuilder().setQuery("Image:${dep.getImage()}").build(),
                    true)
        }

        and:
        "Create a Policy"
        String policyID = PolicyService.createNewPolicy(policy)
        assert policyID != null

        then:
        "Verify Violation #policyName is triggered"
        withRetry(2, 15) {
            assert waitForViolation(depname, policy.getName(), WAIT_FOR_VIOLATION_TIMEOUT)
        }

        cleanup:
        "Remove Policy #policyName"
        PolicyService.deletePolicy(policyID)

        where:
        "Data inputs are :"
        policyName                            | policy | depname | canRun |  requireFreshScan

        "Image Tag"                           |
                Policy.newBuilder()
                        .setName("TestImageTagPolicy")
                        .setDescription("Test image tag")
                        .setRationale("Test image tag")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("Image Assurance")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Image Tag")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue("nginx-1.12").build())
                                                .build()
                                ).build()
                        ).build()       | DEPLOYMENTNGINX | null | false

        "Image Remote"                        |
                Policy.newBuilder()
                        .setName("TestImageRemotePolicy")
                        .setDescription("Test remote tag")
                        .setRationale("Test remote tag")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("Image Assurance")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Image Remote")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue("rhacs-eng/qa-multi-arch")
                                                        .build()).build()
                                ).build()
                        ).build()  | DEPLOYMENTNGINX | null | false

        "Days since image was created"        |
                Policy.newBuilder()
                        .setName("TestDaysImagecreatedPolicy")
                        .setDescription("TestDaysImagecreated")
                        .setRationale("TestDaysImagecreated")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Image Age")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue("1")
                                                        .build()).build()
                                ).build()
                        ).build()   | DEPLOYMENTNGINX | { containerRuntimeVersion.contains("docker") &&
                                                                       !ClusterService.isAKS() } /* ROX-6994 */ | false

        "Dockerfile Line"                     |
                Policy.newBuilder()
                        .setName("TestDockerFileLinePolicy")
                        .setDescription("TestDockerFileLine")
                        .setRationale("TestDockerFileLine")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Dockerfile Line")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue("RUN=apt-get.*")
                                                        .build()).build()
                                ).build()
                        ).build() | DEPLOYMENTNGINX | { containerRuntimeVersion.contains("docker") &&
                                                                       !ClusterService.isAKS() } /* ROX-6994 */ | false

//        TODO(ROX-3102)
//        "Image is NOT Scanned"     |
//                Policy.newBuilder()
//                        .setName("TestImageNotScannedPolicy")
//                        .setDescription("TestImageNotScanned")
//                        .setRationale("TestImageNotScanned")
//                        .addLifecycleStages(LifecycleStage.DEPLOY)
//                        .addCategories("DevOps Best Practices")
//                        .setDisabled(false)
//                        .setSeverityValue(2)
//                        .setFields(PolicyFields.newBuilder()
//                        .setNoScanExists(true))
//                        .build()            | DEPLOYMENTNGINX | null | false

        "CVE is available"                    |
                Policy.newBuilder()
                        .setName("TestCVEPolicy")
                        .setDescription("TestCVE")
                        .setRationale("TestCVE")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("CVE")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue("CVE-2017-18269")
                                                        .build()).build()
                                ).build()
                        ).build()  | DEPLOYMENTNGINX | null | true

        "Port"                                |
                Policy.newBuilder()
                        .setName("TestPortPolicy")
                        .setDescription("Testport")
                        .setRationale("Testport")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Exposed Port")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue("22")
                                                        .build()).build()
                                ).build()
                        ).build() | DEPLOYMENTNGINX | null | false
        "Port Exposure through Load Balancer" |
                Policy.newBuilder()
                        .setName("TestPortExposurePolicy")
                        .setDescription("Testportexposure")
                        .setRationale("Testportexposure")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Port Exposure Method")
                                                .addAllValues([PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue(EXPOSURE_VALUES[0].toString()).build(),
                                                    PolicyOuterClass.PolicyValue.newBuilder()
                                                            .setValue(EXPOSURE_VALUES[1].toString()).build(),
                                                ])
                                                .build()
                                ).build()
                        ).build() | DEPLOYMENTNGINX_LB | { Env.REMOTE_CLUSTER_ARCH != "ppc64le" &&
                                                           Env.REMOTE_CLUSTER_ARCH != "s390x" } | false
        "Port Exposure by Node Port"         |
                Policy.newBuilder()
                        .setName("TestPortExposurePolicy")
                        .setDescription("Testportexposure")
                        .setRationale("Testportexposure")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Port Exposure Method")
                                                .addAllValues([PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue(EXPOSURE_VALUES[0].toString()).build(),
                                                        PolicyOuterClass.PolicyValue.newBuilder()
                                                                .setValue(EXPOSURE_VALUES[1].toString()).build(),
                                                ])
                                        .build()
                                ).build()
                        ).build() | DEPLOYMENTNGINX_NP | null | false

        "Required Label"                      |
                Policy.newBuilder()
                        .setName("TestLabelPolicy")
                        .setDescription("TestLabel")
                        .setRationale("TestLabel")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Required Label")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue("app1=test1")
                                                        .build()).build()
                                ).build()
                        ).build()           | DEPLOYMENTNGINX | null | false

        "Required Annotations"                |
                Policy.newBuilder()
                        .setName("TestAnnotationPolicy")
                        .setDescription("TestAnnotation")
                        .setRationale("TestAnnotation")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Required Annotation")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue("test=annotation")
                                                        .build()).build()
                                ).build()
                        ).build()       | DEPLOYMENTNGINX | null | false

        "Environment Variable is available"   |
                Policy.newBuilder()
                        .setName("TestEnvironmentPolicy")
                        .setDescription("TestEnvironment")
                        .setRationale("TestEnvironment")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Environment Variable")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue("RAW=CLUSTER_NAME=main")
                                                        .build()).build()
                                ).build()
                        ).build()       | DEPLOYMENTNGINX | null | false

        "Container Port"                      |
                Policy.newBuilder()
                        .setName("TestContainerPortPolicy")
                        .setDescription("TestContainerPort")
                        .setRationale("TestContainerPort")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Exposed Port")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue("22")
                                                        .build()).build()
                                ).build()
                        ).build()       | DEPLOYMENTNGINX | null | false

        "Privileged"                          |
                Policy.newBuilder()
                        .setName("TestPrivilegedPolicy")
                        .setDescription("TestPrivileged")
                        .setRationale("TestPrivileged")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Privileged Container")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue("true")
                                                        .build()).build()
                                ).build()
                        ).build()       | DEPLOYMENTNGINX | null | false

        "Protocol"                            |
                Policy.newBuilder()
                        .setName("TestProtocolPolicy")
                        .setDescription("TestProtocol")
                        .setRationale("TestProtocol")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Exposed Port Protocol")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue("TCP")
                                                        .build()).build()
                                ).build()
                        ).build()       | DEPLOYMENTNGINX | null | false

        "Protocol (case-insensitive)"                            |
                Policy.newBuilder()
                        .setName("TestProtocolPolicy")
                        .setDescription("TestProtocol")
                        .setRationale("TestProtocol")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Exposed Port Protocol")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue("tcp")
                                                        .build()).build()
                                ).build()
                        ).build()   | DEPLOYMENTNGINX | null | false

        "Limits"                              |
                Policy.newBuilder()
                        .setName("TestLimitsPolicy")
                        .setDescription("TestLimits")
                        .setRationale("TestLimits")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Container CPU Limit")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue(">= 0")
                                                        .build()).build()).addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Container Memory Limit")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue(">= 0")
                                                        .build()).build()
                                ).build()
                        ).build() | DEPLOYMENTNGINX | null | false

        "Requests"                            |
                Policy.newBuilder()
                        .setName("TestRequestsPolicy")
                        .setDescription("TestRequests")
                        .setRationale("TestRequests")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Container CPU Request")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue(">= 0")
                                                        .build()).build()).addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Container Memory Request")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue(">= 0")
                                                        .build()).build()
                                ).build()
                        ).build()   | DEPLOYMENTNGINX | null | false

        "VolumeName"                          |
                Policy.newBuilder()
                        .setName("TestVolumeNamePolicy")
                        .setDescription("TestVolumeName")
                        .setRationale("TestVolumeName")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Volume Name")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue("test-writable-volume")
                                                        .build()).build())
                        ).build() | DEPLOYMENTNGINX | null | false

        /*"VolumeType" | @Bug : ROX-884
                  Policy.newBuilder()
                          .setName("TestVolumeTypePolicy")
                          .setDescription("TestVolumeType")
                          .setRationale("TestVolumeType")
                          .addLifecycleStages(LifecycleStage.DEPLOY)
                          .addCategories("DevOps Best Practices")
                          .setDisabled(false)
                          .setSeverityValue(2)
                          .setFields(PolicyFields.newBuilder()
                           .setVolumePolicy(VolumePolicy.newBuilder()
                           .setType("Directory").build()))
                          .build() | DEPLOYMENTNGINX | null | false*/

        "HostMount Writable Volume"           |
                Policy.newBuilder()
                        .setName("TestwritableHostmountPolicy")
                        .setDescription("TestWritableHostMount")
                        .setRationale("TestWritableHostMount")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("Security Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Writable Host Mount")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue("true")
                                                        .build()).build())
                        ).build() | DEPLOYMENTNGINX | null | false

        "Writable Volume"                     |
                Policy.newBuilder()
                        .setName("TestWritableVolumePolicy")
                        .setDescription("TestWritableVolumePolicy")
                        .setRationale("TestWritableVolumePolicy")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("Security Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Writable Mounted Volume")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue("true")
                                                        .build()).build())
                        ).build() | DEPLOYMENTNGINX | null | false

        "RBAC API access"                     |
                Policy.newBuilder()
                        .setName("Test RBAC API Access Policy")
                        .setDescription("Test RBAC API Access Policy")
                        .setRationale("Test RBAC API Access Policy")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("Security Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Minimum RBAC Permissions")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue(Rbac.PermissionLevel.ELEVATED_CLUSTER_WIDE.toString())
                                                        .build()).build())
                        ).build() | DEPLOYMENT_RBAC | null | false
    }

    @Unroll
    @Tag("BAT")
    @Tag("SMOKE")
    def "Verify env var policy configuration for source #envVarSource fails validation"() {
        expect:
        assert !PolicyService.createNewPolicy(Policy.newBuilder()
                        .setName("TestEnvironmentPolicy")
                        .setDescription("TestEnvironment")
                        .setRationale("TestEnvironment")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Environment Variable")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue("KEY=VALUE")
                                                        .build()).build())
                        ).build())

        where:
        "Data inputs are :"
        envVarSource | _
        DeploymentOuterClass.ContainerConfig.EnvironmentConfig.EnvVarSource.SECRET_KEY | _
        DeploymentOuterClass.ContainerConfig.EnvironmentConfig.EnvVarSource.CONFIG_MAP_KEY | _
        DeploymentOuterClass.ContainerConfig.EnvironmentConfig.EnvVarSource.FIELD | _
        DeploymentOuterClass.ContainerConfig.EnvironmentConfig.EnvVarSource.RESOURCE_FIELD | _
    }

    @Unroll
    @Tag("BAT")
    @Tag("PZ")
    def "Verify policy scopes are triggered appropriately: #policyName"() {
        when:
        "Create a Policy"
        String policyID = PolicyService.createNewPolicy(policy)
        assert policyID != null

        and:
        "Create deployments"
        violatedDeployments.each {
            orchestrator.createDeploymentNoWait(it)
        }
        nonViolatedDeployments.each {
            orchestrator.createDeploymentNoWait(it)
        }

        then:
        "Verify Violation #policyName is/is not triggered based on scope"
        violatedDeployments.each {
            assert waitForViolation(it.name, policy.getName(), WAIT_FOR_VIOLATION_TIMEOUT)
        }
        nonViolatedDeployments.each {
            // We can wait for a very short period of time here because if we have the violation deployments
            // we have acknowledged that reassessment of the deployments is in progress
            assert checkForNoViolations(it.name, policy.getName())
        }

        cleanup:
        "Remove Policy #policyName"
        policyID == null ?: PolicyService.deletePolicy(policyID)
        violatedDeployments.each {
            orchestrator.deleteDeployment(it)
        }
        nonViolatedDeployments.each {
            orchestrator.deleteDeployment(it)
        }

        where:
        "Data inputs are :"
        policyName                   | policy | violatedDeployments | nonViolatedDeployments
        "LabelScope"                 |
                Policy.newBuilder()
                        .setName("Test Label Scope")
                        .setDescription("Test Label Scope")
                        .setRationale("Test Label Scope")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Image Tag")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue("latest")
                                                        .build()).build())
                        ).addScope(Scope.newBuilder()
                                .setLabel(Scope.Label.newBuilder()
                                        .setKey("app")
                                        .setValue("qa-test").build()
                                ).build()
                        ).build()             |
                [new Deployment()
                         .setName("label-scope-violation")
                         .addLabel("app", "qa-test")
                         .setImage("quay.io/rhacs-eng/qa-multi-arch-nginx:latest"),]                |
                [new Deployment()
                         .setName("label-scope-non-violation")
                         .setImage("quay.io/rhacs-eng/qa-multi-arch-nginx:latest"),]
        "NamespaceScope"             |
                Policy.newBuilder()
                        .setName("Test Namespace Scope")
                        .setDescription("Test Namespace Scope")
                        .setRationale("Test Namespace Scope")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Image Tag")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue("latest")
                                                        .build()).build())
                        )
                        .addScope(Scope.newBuilder()
                                .setNamespace(Constants.ORCHESTRATOR_NAMESPACE).build()
                        ).build()             |
                [new Deployment()
                         .setName("namespace-scope-violation")
                         .setImage("quay.io/rhacs-eng/qa-multi-arch-nginx:latest"),]                |
                [new Deployment()
                         .setName("namespace-scope-non-violation")
                         .setNamespace("default")
                         .setImage("quay.io/rhacs-eng/qa-multi-arch-nginx:latest"),]
        "ClusterNamespaceLabelScope" |
                Policy.newBuilder()
                        .setName("Test All Scopes in One")
                        .setDescription("Test All Scopes in One")
                        .setRationale("Test All Scopes in One")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Image Tag")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue("latest")
                                                        .build()).build())
                        )
                        .addScope(Scope.newBuilder()
                                .setCluster(ClusterService.getClusterId())
                                .setNamespace(Constants.ORCHESTRATOR_NAMESPACE)
                                .setLabel(Scope.Label.newBuilder()
                                        .setKey("app")
                                        .setValue("qa-test").build()
                                ).build()
                        ).build()             |
                [new Deployment()
                         .setName("all-scope-violation")
                         .addLabel("app", "qa-test")
                         .setImage("quay.io/rhacs-eng/qa-multi-arch-nginx:latest"),]                |
                [new Deployment()
                         .setName("all-scope-non-violation")
                         .setNamespace("default")
                         .addLabel("app", "qa-test")
                         .setImage("quay.io/rhacs-eng/qa-multi-arch-nginx:latest"),]
        "MultipleScopes"             |
                Policy.newBuilder()
                        .setName("Test Multiple Scopes")
                        .setDescription("Test Multiple Scopes")
                        .setRationale("Test Multiple Scopes")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .addPolicySections(
                                PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder()
                                                .setFieldName("Image Tag")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                        .setValue("latest")
                                                        .build()).build())
                        )
                        .addScope(Scope.newBuilder()
                                .setNamespace(Constants.ORCHESTRATOR_NAMESPACE).build()
                        )
                        .addScope(Scope.newBuilder()
                                .setLabel(Scope.Label.newBuilder()
                                        .setKey("app")
                                        .setValue("qa-test").build()
                                ).build()
                        ).build()             |
                [new Deployment()
                         .setName("multiple-scope-violation")
                         .setImage("quay.io/rhacs-eng/qa-multi-arch-nginx:latest"),
                 new Deployment()
                         .setName("multiple-scope-violation2")
                         .setNamespace("default")
                         .addLabel("app", "qa-test")
                         .setImage("quay.io/rhacs-eng/qa-multi-arch-nginx:latest"),]                |
                [new Deployment()
                         .setName("multiple-scope-non-violation")
                         .setNamespace("default")
                         .setImage("quay.io/rhacs-eng/qa-multi-arch-nginx:latest"),]
    }

    @Unroll
    @Tag("BAT")
    @Tag("PZ")
    def "Verify dryRun on a disabled policy generates violations for matching deployments"() {
        when:
        "Initialize a new disabled policy that will match an existing deployment"
        Policy policy = Policy.newBuilder()
                                .setName("TestPrivilegedPolicy")
                                .setDescription("TestPrivileged")
                                .setRationale("TestPrivileged")
                                .addLifecycleStages(LifecycleStage.DEPLOY)
                                .addCategories("DevOps Best Practices")
                                .setDisabled(true)
                                .setSeverityValue(2)
                                .addPolicySections(PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                        PolicyOuterClass.PolicyGroup.newBuilder().setFieldName("Privileged Container")
                                                .addValues(PolicyOuterClass.PolicyValue.newBuilder().
                                                        setValue("true").build())
                                                .build()
                                ).build())
                                .build()

        and:
        "dryRun is called on the policy"
        DryRunResponse dryRunResponse = PolicyService.dryRunPolicy(policy)

        then:
        "Verify dryRun response contains alert/s for matching deployments"
        assert dryRunResponse != null
        assert dryRunResponse.getAlertsCount() > 0
    }
}
