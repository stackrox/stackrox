import static Services.waitForViolation
import stackrox.generated.PolicyServiceOuterClass.Policy
import stackrox.generated.PolicyServiceOuterClass.PolicyFields
import stackrox.generated.PolicyServiceOuterClass.ImageNamePolicy
import stackrox.generated.PolicyServiceOuterClass.LifecycleStage
import groups.BAT
import objects.Deployment
import org.junit.experimental.categories.Category
import services.CreatePolicyService
import spock.lang.Unroll

class PolicyConfigurationTest extends BaseSpecification {
    @Unroll
    @Category(BAT)
    def "Verify policy configuration #testName can be triggered"() {
        when:
        "Create a Policy"
        String policyID = CreatePolicyService.createNewPolicy(policy)
        assert policyID != null

        and:
        "Create a Deployment"
        orchestrator.createDeployment(deployment)

        then:
        "Verify Violation #testName is triggered"
        assert waitForViolation(deployment.getName(), policy.getName(), 30)

        cleanup:
        "Remove Deployment and Policy #testName"
        orchestrator.deleteDeployment(deployment.getName())
        CreatePolicyService.deletePolicy(policyID)

        where:
        "Data inputs are :"
        testName | policy | deployment

        "Test Image Tag configure" |
        Policy.newBuilder()
            .setName("testImageTag")
            .setDescription("test image tag")
            .setRationale("test image tag")
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
            .build() |
        new Deployment()
            .setName ("testnginx110")
            .setImage ("nginx:1.10")
            .addPort (22)
            .addLabel ( "app", "test" )

        "Test Latest Tag" |
        Policy.newBuilder()
            .setName("testImageTagLatest")
            .setDescription("qa test")
            .setRationale("qa test")
            .addLifecycleStages(LifecycleStage.DEPLOY)
            .addCategories("Image Assurance")
            .setDisabled(false)
            .setSeverityValue(2)
            .setFields(PolicyFields.newBuilder()
                .setImageName(
                    ImageNamePolicy.newBuilder()
                        .setTag("latest")
                        .build())
                .build())
            .build() |
        new Deployment()
            .setName("testnginxlatest")
            .setImage("nginx:latest")
            .addPort(22)
            .addLabel("app", "test")
    }

}
