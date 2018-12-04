import static Services.waitForViolation
import stackrox.generated.PolicyServiceOuterClass.Policy
import stackrox.generated.PolicyServiceOuterClass.PolicyFields
import stackrox.generated.PolicyServiceOuterClass.ImageNamePolicy
import stackrox.generated.PolicyServiceOuterClass.LifecycleStage
import stackrox.generated.PolicyServiceOuterClass.DockerfileLineRuleField
import stackrox.generated.PolicyServiceOuterClass.PortPolicy
import stackrox.generated.PolicyServiceOuterClass.ResourcePolicy
import stackrox.generated.PolicyServiceOuterClass.KeyValuePolicy
import stackrox.generated.PolicyServiceOuterClass.NumericalPolicy
import stackrox.generated.PolicyServiceOuterClass.Comparator
import stackrox.generated.PolicyServiceOuterClass.VolumePolicy
import groups.BAT
import objects.Deployment
import org.junit.experimental.categories.Category
import services.CreatePolicyService
import spock.lang.Unroll

class PolicyConfigurationTest extends BaseSpecification {
    static final private String DEPLOYMENTNGINX = "deploymentnginx"
    static final private String DEPLOYMENTREMOTE = "deploymentremote"
    //static final private String DEPLOYMENTREGISTRY = "deploymentregistry"
    static final private String DEPLOYMENTAGE = "deploymentage"
    static final private String DEPLOYMENTDOCKERFILE = "deploymentdockerfileandnoscan"
    static final private String STRUTS = "qadefpolstruts"
    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                    .setName(DEPLOYMENTNGINX)
                    .setImage("nginx:latest")
                    .addPort(22, "TCP")
                    .addAnnotation("test", "annotation")
                    .setEnv(["CLUSTER_NAME": "main"])
                    .addLabel("app", "test")
                    .setPrivilegedFlag(true)
                    .addLimits("cpu", "0")
                    .addLimits("memory", "0")
                    .addRequest("memory", "0")
                    .addRequest("cpu", "0")
                    .addVolMountName("test")
                    .addVolName("test")
                    .addMountPath("/tmp")
                    .setSkipReplicaWait(true),

            new Deployment()
                    .setName(DEPLOYMENTREMOTE)
                    .setImage("apollo-dtr.rox.systems/legacy-apps/ssl-terminator:latest")
                    .addLabel("app", "test"),
            new Deployment()
                    .setName(DEPLOYMENTAGE)
                    .setImage("nginx:1.10")
                    .addLabel("app", "test"),
            new Deployment()
                    .setName(DEPLOYMENTDOCKERFILE)
                    .setImage("nginx:1.7.9")
                    .addLabel("app", "test"),
            /* new Deployment()
                     .setName (DEPLOYMENTREGISTRY)
                     .setImage ("us.gcr.io/ultra-current-825/apache-dns:latest")
                     .addLabel ( "app", "test" ),*/
            new Deployment()
                    .setName(STRUTS)
                    .setImage("apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
                    .addLabel("app", "test"),
    ]

    def setupSpec() {
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deploymentId : DEPLOYMENTS) {
            assert Services.waitForDeployment(deploymentId)
        }
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment)
        }
    }

    @Unroll
    @Category(BAT)
    def "Verify policy configuration #policyName can be triggered"() {
        when:
        "Create a Policy"
        String policyID = CreatePolicyService.createNewPolicy(policy)
        assert policyID != null

        then:
        "Verify Violation #policyName is triggered"
        assert waitForViolation(depname, policy.getName(), 60)

        cleanup:
        "Remove Policy #policyName"
        CreatePolicyService.deletePolicy(policyID)

        where:
        "Data inputs are :"
        policyName                 | policy | depname

        "Image Tag"                |
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
                                .setTag("1.10")
                                .build())
                        .build())
                        .build()            | DEPLOYMENTAGE

        /*"Image Registry" |
                Policy.newBuilder()
                        .setName("TestImageRegistryPolicy")
                        .setDescription("Test registry tag")
                        .setRationale("Test registry tag")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("Image Assurance")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                        .setImageName(
                        ImageNamePolicy.newBuilder()
                                .setRegistry("us.gcr.io")
                                .build())
                        .build())
                        .build() | DEPLOYMENTREGISTRY */

        "Image Remote"             |
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
                                .setRemote("legacy-apps")
                                .build())
                        .build())
                        .build()            | DEPLOYMENTREMOTE

        "Days since Image created" |
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
                        .build()            | DEPLOYMENTAGE

        "Days since Image scanned" |
                Policy.newBuilder()
                        .setName("TestDaysImagescannedPolicy")
                        .setDescription("TestDaysImagescanned")
                        .setRationale("TestDaysImagescanned")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                        .setScanAgeDays(30)
                        .build())
                        .build()            | DEPLOYMENTREMOTE

        "Dockerfile Line"          |
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
                        .setValue("apt-get")
                        .setInstruction("RUN")
                        .build()))
                        .build()            | DEPLOYMENTDOCKERFILE

        "Image is NOT Scanned"     |
                Policy.newBuilder()
                        .setName("TestImageNotScannedPolicy")
                        .setDescription("TestImageNotScanned")
                        .setRationale("TestImageNotScanned")
                        .addLifecycleStages(LifecycleStage.DEPLOY)
                        .addCategories("DevOps Best Practices")
                        .setDisabled(false)
                        .setSeverityValue(2)
                        .setFields(PolicyFields.newBuilder()
                        .setNoScanExists(true))
                        .build()            | DEPLOYMENTDOCKERFILE

        "CVE is available"         |
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
                        .build()            | STRUTS

        "Port"                     |
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
                        .build()            | DEPLOYMENTNGINX

        "Required Label"           |
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
                        .build()            | DEPLOYMENTNGINX

        "Required Annotations"     |
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
                        .build()            | DEPLOYMENTNGINX

        "Environment is available" |
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
                        .setValue("main").build()))
                        .build()            | DEPLOYMENTNGINX

        "Container Port"           |
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
                        .build()            | DEPLOYMENTNGINX

        "Privileged"               |
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
                        .build()            | DEPLOYMENTNGINX

        "Protocol"                 |
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
                        .build()            | DEPLOYMENTNGINX

        "Limits"                   |
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
                        .build()            | DEPLOYMENTNGINX

        "Requests"                 |
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
                        .build()            | DEPLOYMENTNGINX
        "VolumeName"               |
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
                        .setName("test").build()))
                        .build()            | DEPLOYMENTNGINX

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
    }
}
