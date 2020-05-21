import static Services.waitForViolation

import services.ImageService
import util.Timer
import objects.K8sPolicyRule
import objects.K8sRole
import objects.K8sRoleBinding
import objects.K8sServiceAccount
import objects.K8sSubject
import io.stackrox.proto.storage.DeploymentOuterClass
import io.stackrox.proto.storage.PolicyOuterClass
import objects.Service
import common.Constants
import objects.Volume
import io.stackrox.proto.storage.Rbac
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import io.stackrox.proto.storage.PolicyOuterClass.PolicyFields
import io.stackrox.proto.storage.PolicyOuterClass.ImageNamePolicy
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.PolicyOuterClass.DockerfileLineRuleField
import io.stackrox.proto.storage.PolicyOuterClass.PortPolicy
import io.stackrox.proto.storage.PolicyOuterClass.ResourcePolicy
import io.stackrox.proto.storage.PolicyOuterClass.KeyValuePolicy
import io.stackrox.proto.storage.PolicyOuterClass.NumericalPolicy
import io.stackrox.proto.storage.PolicyOuterClass.Comparator
import io.stackrox.proto.storage.PolicyOuterClass.VolumePolicy
import io.stackrox.proto.storage.ScopeOuterClass.Scope
import groups.BAT
import groups.SMOKE
import objects.Deployment
import org.junit.experimental.categories.Category
import services.ClusterService
import services.CreatePolicyService
import spock.lang.Unroll

class PolicyConfigurationTest extends BaseSpecification {
    static final private String DEPLOYMENTNGINX = "deploymentnginx"
    static final private String DNS = "qaapachedns"
    static final private String STRUTS = "qadefpolstruts"
    static final private String DEPLOYMENTNGINX_LB = "deploymentnginx-lb"
    static final private String DEPLOYMENTNGINX_NP = "deploymentnginx-np"
    static final private String DEPLOYMENT_RBAC = "deployment-rbac"
    static final private String SERVICE_ACCOUNT_NAME = "policy-config-sa"
    static final private String NGINX_LATEST_WITH_DIGEST_NAME = "nginx-1-17-with-tag-and-digest"
    static final private String NGINX_LATEST_NAME = "nginx-latest"
    private static final String CLUSTER_ROLE_NAME = "policy-config-role"

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
                    .setImage("nginx:1.7.9")
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
                    .setImage("apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
                    .addLabel("app", "test"),
            new Deployment()
                    .setName(DNS)
                    .setImage("apollo-dtr.rox.systems/legacy-apps/apache-dns:latest")
                    .addLabel("app", "test"),
            new Deployment()
                    .setName(DEPLOYMENTNGINX_LB)
                    .setImage("nginx:1.7.9")
                    .addPort(22, "TCP")
                    .addAnnotation("test", "annotation")
                    .setEnv(["CLUSTER_NAME": "main"])
                    .addLabel("app", "test")
                    .setCreateLoadBalancer(true).setExposeAsService(true),
            new Deployment()
                    .setName(DEPLOYMENTNGINX_NP)
                    .setImage("nginx:1.7.9")
                    .addPort(22, "TCP")
                    .addAnnotation("test", "annotation")
                    .setEnv(["CLUSTER_NAME": "main"])
                    .addLabel("app", "test"),
            new Deployment()
                    .setName(DEPLOYMENT_RBAC)
                    .setNamespace(Constants.ORCHESTRATOR_NAMESPACE)
                    .setServiceAccountName(SERVICE_ACCOUNT_NAME)
                    .setImage("nginx:1.15.4-alpine")
                    .setSkipReplicaWait(true),
    ]

    static final private Deployment NGINX_WITH_DIGEST = new Deployment()
            .setName(NGINX_LATEST_WITH_DIGEST_NAME)
            .setImage("nginx:1.17@sha256:86ae264c3f4acb99b2dee4d0098c40cb8c46dcf9e1148f05d3a51c4df6758c12")
            .setCommand(["sleep", "60000"])

    static final private Deployment NGINX_LATEST = new Deployment()
            .setName(NGINX_LATEST_NAME)
            .setImage("nginx:latest@sha256:86ae264c3f4acb99b2dee4d0098c40cb8c46dcf9e1148f05d3a51c4df6758c12")
            .setCommand(["sleep", "60000"])

    static final private Service NPSERVICE = new Service(DEPLOYMENTS.find { it.name == DEPLOYMENTNGINX_NP })
            .setType(Service.Type.NODEPORT)

    def setupSpec() {
        NEW_CLUSTER_ROLE.setRules(new K8sPolicyRule(resources: ["nodes"], apiGroups: [""], verbs: ["list"]))
        orchestrator.createServiceAccount(NEW_SA)
        orchestrator.createClusterRole(NEW_CLUSTER_ROLE)
        orchestrator.createClusterRoleBinding(NEW_CLUSTER_ROLE_BINDING)
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deploymentId : DEPLOYMENTS) {
            assert Services.waitForDeployment(deploymentId)
        }
        orchestrator.createService(NPSERVICE)
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

    @Category(BAT)
    def "Verify name violations with same ID as existing image are still triggered"() {
        given:
        "Create a busybox deployment has same ID as latest"
        orchestrator.createDeployment(NGINX_WITH_DIGEST)

        when:
        Timer t = new Timer(60, 1)
        def image
        while (image == null && t.IsValid()) {
            image = ImageService.getImage("sha256:86ae264c3f4acb99b2dee4d0098c40cb8c46dcf9e1148f05d3a51c4df6758c12")
        }
        assert image != null

        and:
        "Run busybox latest with same digest as previous image"
        orchestrator.createDeployment(NGINX_LATEST)

        then:
        "Ensure that the latest tag violation shows up"
        def hasViolation = waitForViolation(NGINX_LATEST_NAME, "Latest Tag")
        println "Has violation ${hasViolation}"
        assert hasViolation

        cleanup:
        "Remove the deployments"
        orchestrator.deleteDeployment(NGINX_WITH_DIGEST)
        orchestrator.deleteDeployment(NGINX_LATEST)
    }

    @Category(BAT)
    def "Verify lastUpdated field is updated correctly for policy - ROX-3971 production bug"() {
        given:
        "Create a copy of a Latest Tag"
        PolicyOuterClass.Policy.Builder policy = Services.getPolicyByName("Latest tag").toBuilder()
        def name = policy.name + new Random().nextInt(2000)
        policy.setName(name)
                .setId("") // set ID to empty so that a new policy is created and not overwrite the original latest tag
                .build()
        def policyId = CreatePolicyService.createNewPolicy(policy.build())
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
        policyId == null ?: CreatePolicyService.deletePolicy(policyId)
    }

    @Unroll
    @Category([BAT, SMOKE])
    def "Verify policy configuration #policyName can be triggered"() {
        when:
        "Create a Policy"
        String policyID = CreatePolicyService.createNewPolicy(policy)
        assert policyID != null

        then:
        "Verify Violation #policyName is triggered"
        assert waitForViolation(depname, policy.getName(), 90)

        cleanup:
        "Remove Policy #policyName"
        CreatePolicyService.deletePolicy(policyID)

        where:
        "Data inputs are :"
        policyName                            | policy | depname

        "Image Tag"                           |
                Policy.newBuilder()
                        .setName("TestImageTagPolicy")
                        .setDescription("Test image tag")
                        .setRationale("Test image tag")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("Image Assurance")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setImageName(
                                        ImageNamePolicy.newBuilder()
                                                .setTag("1.7.9")
                                                .build())
                                .build())
                        .build()                       | DEPLOYMENTNGINX

        "Image Remote"                        |
                Policy.newBuilder()
                        .setName("TestImageRemotePolicy")
                        .setDescription("Test remote tag")
                        .setRationale("Test remote tag")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("Image Assurance")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setImageName(
                                        ImageNamePolicy.newBuilder()
                                                .setRemote("library/nginx")
                                                .build())
                                .build())
                        .build()                       | DEPLOYMENTNGINX

        "Days since image was created"        |
                Policy.newBuilder()
                        .setName("TestDaysImagecreatedPolicy")
                        .setDescription("TestDaysImagecreated")
                        .setRationale("TestDaysImagecreated")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setImageAgeDays(1)
                                .build())
                        .build()                       | DEPLOYMENTNGINX

        "Days since image was last scanned"   |
                Policy.newBuilder()
                        .setName("TestDaysImagescannedPolicy")
                        .setDescription("TestDaysImagescanned")
                        .setRationale("TestDaysImagescanned")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setScanAgeDays(1)
                                .build())
                        .build()                       | DNS

        "Dockerfile Line"                     |
                Policy.newBuilder()
                        .setName("TestDockerFileLinePolicy")
                        .setDescription("TestDockerFileLine")
                        .setRationale("TestDockerFileLine")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setLineRule(DockerfileLineRuleField.newBuilder()
                                        .setValue("apt-get.*")
                                        .setInstruction("RUN")
                                        .build()))
                        .build()                       | DEPLOYMENTNGINX
//
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
//                        .build()            | DEPLOYMENTNGINX

        "CVE is available"                    |
                Policy.newBuilder()
                        .setName("TestCVEPolicy")
                        .setDescription("TestCVE")
                        .setRationale("TestCVE")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setCve("CVE-2017-5638"))
                        .build()                       | STRUTS

        "Port"                                |
                Policy.newBuilder()
                        .setName("TestPortPolicy")
                        .setDescription("Testport")
                        .setRationale("Testport")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setPortPolicy(PortPolicy.newBuilder()
                                        .setPort(22).build()))
                        .build()                       | DEPLOYMENTNGINX
        "Port Exposure through Load Balancer" |
                Policy.newBuilder()
                        .setName("TestPortExposurePolicy")
                        .setDescription("Testportexposure")
                        .setRationale("Testportexposure")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setPortExposurePolicy(PolicyOuterClass.PortExposurePolicy.newBuilder()
                                        .addAllExposureLevels(EXPOSURE_VALUES)))
                        .build()                       | DEPLOYMENTNGINX_LB
        "Port Exposure by  Node Port"         |
                Policy.newBuilder()
                        .setName("TestPortExposurePolicy")
                        .setDescription("Testportexposure")
                        .setRationale("Testportexposure")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setPortExposurePolicy(PolicyOuterClass.PortExposurePolicy.newBuilder()
                                        .addAllExposureLevels(EXPOSURE_VALUES)))
                        .build()                       | DEPLOYMENTNGINX_NP

        "Required Label"                      |
                Policy.newBuilder()
                        .setName("TestLabelPolicy")
                        .setDescription("TestLabel")
                        .setRationale("TestLabel")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setRequiredLabel(KeyValuePolicy.newBuilder()
                                        .setKey("app1")
                                        .setValue("test1").build()))
                        .build()                       | DEPLOYMENTNGINX

        "Required Annotations"                |
                Policy.newBuilder()
                        .setName("TestAnnotationPolicy")
                        .setDescription("TestAnnotation")
                        .setRationale("TestAnnotation")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setRequiredAnnotation(KeyValuePolicy.newBuilder()
                                        .setKey("test")
                                        .setValue("annotation").build()))
                        .build()                       | DEPLOYMENTNGINX

        "Environment Variable is available"   |
                Policy.newBuilder()
                        .setName("TestEnvironmentPolicy")
                        .setDescription("TestEnvironment")
                        .setRationale("TestEnvironment")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setEnv(KeyValuePolicy.newBuilder()
                                        .setKey("CLUSTER_NAME")
                                        .setValue("main")
                                        .setEnvVarSource(DeploymentOuterClass.
                                                ContainerConfig.EnvironmentConfig.EnvVarSource.RAW)
                                        .build()))
                        .build()                       | DEPLOYMENTNGINX

        "Container Port"                      |
                Policy.newBuilder()
                        .setName("TestContainerPortPolicy")
                        .setDescription("TestContainerPort")
                        .setRationale("TestContainerPort")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setPortPolicy(PortPolicy.newBuilder()
                                        .setPort(22)).build())
                        .build()                       | DEPLOYMENTNGINX

        "Privileged"                          |
                Policy.newBuilder()
                        .setName("TestPrivilegedPolicy")
                        .setDescription("TestPrivileged")
                        .setRationale("TestPrivileged")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setPrivileged(true))
                        .build()                       | DEPLOYMENTNGINX

        "Protocol"                            |
                Policy.newBuilder()
                        .setName("TestProtocolPolicy")
                        .setDescription("TestProtocol")
                        .setRationale("TestProtocol")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setPortPolicy(PortPolicy.newBuilder()
                                        .setProtocol("TCP").build()))
                        .build()                       | DEPLOYMENTNGINX

        "Limits"                              |
                Policy.newBuilder()
                        .setName("TestLimitsPolicy")
                        .setDescription("TestLimits")
                        .setRationale("TestLimits")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setContainerResourcePolicy(ResourcePolicy.newBuilder()
                                        .setCpuResourceLimit(NumericalPolicy.newBuilder()
                                                .setOp(Comparator.EQUALS)
                                                .setValue(0).build())
                                        .setMemoryResourceLimit(NumericalPolicy.newBuilder()
                                                .setOp(Comparator.EQUALS)
                                                .setValue(0).build())))
                        .build()                       | DEPLOYMENTNGINX

        "Requests"                            |
                Policy.newBuilder()
                        .setName("TestRequestsPolicy")
                        .setDescription("TestRequests")
                        .setRationale("TestRequests")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setContainerResourcePolicy(ResourcePolicy.newBuilder()
                                        .setMemoryResourceRequest(NumericalPolicy.newBuilder()
                                                .setOp(Comparator.EQUALS)
                                                .setOpValue(0).build())
                                        .setCpuResourceRequest(NumericalPolicy.newBuilder()
                                                .setOp(Comparator.EQUALS)
                                                .setValue(0).build())))
                        .build()                       | DEPLOYMENTNGINX
        "VolumeName"                          |
                Policy.newBuilder()
                        .setName("TestVolumeNamePolicy")
                        .setDescription("TestVolumeName")
                        .setRationale("TestVolumeName")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setVolumePolicy(VolumePolicy.newBuilder()
                                        .setName("test-writable-volume").build()))
                        .build()                       | DEPLOYMENTNGINX

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
                          .build() | DEPLOYMENTNGINX*/
        "HostMount Writable Volume"           |
                Policy.newBuilder()
                        .setName("TestwritableHostmountPolicy")
                        .setDescription("TestWritableHostMount")
                        .setRationale("TestWritableHostMount")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("Security Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setHostMountPolicy(PolicyOuterClass.HostMountPolicy.newBuilder()
                                        .setReadOnly(false)).build())
                        .build()                       | DEPLOYMENTNGINX
        "Writable Volume"                     |
                Policy.newBuilder()
                        .setName("TestWritableVolumePolicy")
                        .setDescription("TestWritableVolumePolicy")
                        .setRationale("TestWritableVolumePolicy")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("Security Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setVolumePolicy(
                                        PolicyOuterClass.VolumePolicy.newBuilder().setReadOnly(false).build())
                                .build())
                        .build()                       | DEPLOYMENTNGINX
        "RBAC API access"                     |
                Policy.newBuilder()
                        .setName("Test RBAC API Access Policy")
                        .setDescription("Test RBAC API Access Policy")
                        .setRationale("Test RBAC API Access Policy")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("Security Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setPermissionPolicy(PolicyOuterClass.PermissionPolicy.newBuilder()
                                        .setPermissionLevel(Rbac.PermissionLevel.ELEVATED_CLUSTER_WIDE)))
                        .build()                       | DEPLOYMENT_RBAC
    }

    @Unroll
    @Category(BAT)
    def "Verify policy scopes are triggered appropriately: #policyName"() {
        when:
        "Create a Policy"
        String policyID = CreatePolicyService.createNewPolicy(policy)
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
            assert waitForViolation(it.name, policy.getName(), 90)
        }
        nonViolatedDeployments.each {
            // We can wait for a very short period of time here because if we have the violation deployments
            // we have acknowledged that reassessment of the deployments is in progress
            assert !waitForViolation(it.name, policy.getName(), 5)
        }

        cleanup:
        "Remove Policy #policyName"
        policyID == null ?: CreatePolicyService.deletePolicy(policyID)
        violatedDeployments.each {
            it.deploymentUid == null ?: orchestrator.deleteDeployment(it)
        }
        nonViolatedDeployments.each {
            it.deploymentUid == null ?: orchestrator.deleteDeployment(it)
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
                        .setFields(PolicyFields.newBuilder()
                                .setImageName(ImageNamePolicy.newBuilder()
                                        .setTag("latest").build()
                                ).build()
                        )
                        .addScope(Scope.newBuilder()
                                .setLabel(Scope.Label.newBuilder()
                                        .setKey("app")
                                        .setValue("qa-test").build()
                                ).build()
                        ).build()             |
                [new Deployment()
                         .setName("label-scope-violation")
                         .addLabel("app", "qa-test")
                         .setImage("nginx:latest"),]                |
                [new Deployment()
                         .setName("label-scope-non-violation")
                         .setImage("nginx:latest"),]
        "NamespaceScope"             |
                Policy.newBuilder()
                        .setName("Test Namespace Scope")
                        .setDescription("Test Namespace Scope")
                        .setRationale("Test Namespace Scope")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setImageName(ImageNamePolicy.newBuilder()
                                        .setTag("latest").build()
                                ).build()
                        )
                        .addScope(Scope.newBuilder()
                                .setNamespace(Constants.ORCHESTRATOR_NAMESPACE).build()
                        ).build()             |
                [new Deployment()
                         .setName("namespace-scope-violation")
                         .setImage("nginx:latest"),]                |
                [new Deployment()
                         .setName("namespace-scope-non-violation")
                         .setNamespace("default")
                         .setImage("nginx:latest"),]
        "ClusterNamespaceLabelScope" |
                Policy.newBuilder()
                        .setName("Test All Scopes in One")
                        .setDescription("Test All Scopes in One")
                        .setRationale("Test All Scopes in One")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setImageName(ImageNamePolicy.newBuilder()
                                        .setTag("latest").build()
                                ).build()
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
                         .setImage("nginx:latest"),]                |
                [new Deployment()
                         .setName("all-scope-non-violation")
                         .setNamespace("default")
                         .addLabel("app", "qa-test")
                         .setImage("nginx:latest"),]
        "MultipleScopes"             |
                Policy.newBuilder()
                        .setName("Test Multiple Scopes")
                        .setDescription("Test Multiple Scopes")
                        .setRationale("Test Multiple Scopes")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                                .setImageName(ImageNamePolicy.newBuilder()
                                        .setTag("latest").build()
                                ).build()
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
                         .setImage("nginx:latest"),
                 new Deployment()
                         .setName("multiple-scope-violation2")
                         .setNamespace("default")
                         .addLabel("app", "qa-test")
                         .setImage("nginx:latest"),]                |
                [new Deployment()
                         .setName("multiple-scope-non-violation")
                         .setNamespace("default")
                         .setImage("nginx:latest"),]
    }
}
