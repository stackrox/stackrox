import static Services.waitForViolation
import stackrox.generated.PolicyServiceOuterClass.Policy
import stackrox.generated.PolicyServiceOuterClass.PolicyFields
import stackrox.generated.PolicyServiceOuterClass.ImageNamePolicy
import stackrox.generated.PolicyServiceOuterClass.LifecycleStage
import stackrox.generated.PolicyServiceOuterClass.DockerfileLineRuleField
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
    static final private String DEPLOYMENTSCANAGE = "deploymentscanage"
    static final private String DEPLOYMENTDOCKERFILE = "deploymentdockerfileandnoscan"

    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                    .setName (DEPLOYMENTNGINX)
                    .setImage ("nginx:1.10")
                    .addPort (22)
                    .addLabel ( "app", "test" ),

            new Deployment()
                    .setName (DEPLOYMENTREMOTE)
                    .setImage ("apollo-dtr.rox.systems/legacy-apps/ssl-terminator:latest")
                    .addLabel ( "app", "test" ),

            new Deployment()
                    .setName (DEPLOYMENTAGE)
                    .setImage ("nginx:1.10")
                    .addLabel ( "app", "test" ),

            new Deployment()
                    .setName (DEPLOYMENTSCANAGE)
                    .setImage ("apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
                    .addLabel ( "app", "test" ),

            new Deployment()
                    .setName (DEPLOYMENTDOCKERFILE)
                    .setImage ("nginx:1.7.9")
                    .addLabel ( "app", "test" ),

           /* new Deployment()
                    .setName (DEPLOYMENTREGISTRY)
                    .setImage ("us.gcr.io/ultra-current-825/apache-dns:latest")
                    .addLabel ( "app", "test" ),*/
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
        assert waitForViolation(depname,  policy.getName(), 600)

        cleanup:
        "Remove Policy #policyName"
        CreatePolicyService.deletePolicy(policyID)

        where:
        "Data inputs are :"
        policyName | policy | depname

        "Image Tag" |
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
                        .build() | DEPLOYMENTNGINX

       /* "Image Registry" |
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

        "Image Remote" |
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
                        .build() | DEPLOYMENTREMOTE

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
                        .build() | DEPLOYMENTAGE

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
                        .build() | DEPLOYMENTSCANAGE

        "Dockerfile Line" |
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
                        .build() | DEPLOYMENTDOCKERFILE

        "Image is NOT Scanned" |
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
                        .build() | DEPLOYMENTDOCKERFILE
    }

}
