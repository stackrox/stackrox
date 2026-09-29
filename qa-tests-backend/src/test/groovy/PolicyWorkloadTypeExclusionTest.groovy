import static Services.checkForNoActiveViolations
import static Services.waitForViolation

import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.Exclusion
import io.stackrox.proto.storage.PolicyOuterClass.LifecycleStage
import io.stackrox.proto.storage.PolicyOuterClass.Policy

import objects.Deployment
import objects.Job
import services.FeatureFlagService
import services.PolicyService

import org.junit.Assume
import spock.lang.Tag

@Tag("PZ")
class PolicyWorkloadTypeExclusionTest extends BaseSpecification {
    static final private Integer WAIT_FOR_VIOLATION_TIMEOUT = isRaceBuild() ? 450 : 90

    static final private Deployment DEPLOYMENT = new Deployment()
            .setName("wte-deploy-${RUN_ID}")
            .setImagePrefetcherAffinity()
            .setImage(TEST_IMAGE)
            .addLabel("app", "wte")

    // Keep the Job running. The only other Job helper usage creates a short-lived Job that
    // completes and is then deleted from StackRox (DeploymentTest).
    static final private Job JOB = new Job()
            .setName("wte-job-${RUN_ID}")
            .setImagePrefetcherAffinity()
            .setImage(TEST_IMAGE)
            .addLabel("app", "wte")
            .setCommand(["/bin/sh", "-c", "sleep 600"])

    def setupSpec() {
        Assume.assumeTrue(
                "ROX_POLICY_WORKLOAD_TYPE_EXCLUSION is disabled",
                FeatureFlagService.isFeatureFlagEnabled("ROX_POLICY_WORKLOAD_TYPE_EXCLUSION"))

        orchestrator.createDeployment(DEPLOYMENT)
        assert Services.waitForDeployment(DEPLOYMENT)

        def createdJob = orchestrator.createJob(JOB)
        assert createdJob != null
        assert Services.waitForDeploymentByID(createdJob.getMetadata().getUid(), JOB.name)
    }

    def cleanupSpec() {
        orchestrator.deleteDeployment(DEPLOYMENT)
        orchestrator.deleteJob(JOB)
    }

    @Tag("BAT")
    def "exclude-by-type JOB suppresses Job violations and still fires on Deployments"() {
        given:
        def baselineID
        def policyID
        def baselineName = "wte-baseline-${RUN_ID}"
        def policyName = "wte-job-type-${RUN_ID}"

        when:
        "the Job is evaluated like any other workload when no type exclusion is set"
        baselineID = PolicyService.createNewPolicy(imageTagPolicy(baselineName).build())
        assert baselineID != null

        then:
        waitForViolation(DEPLOYMENT.name, baselineName, WAIT_FOR_VIOLATION_TIMEOUT)
        waitForViolation(JOB.name, baselineName, WAIT_FOR_VIOLATION_TIMEOUT)

        when:
        "a matching policy excludes all Jobs"
        policyID = PolicyService.createNewPolicy(imageTagPolicy(policyName)
                .addExclusions(Exclusion.newBuilder()
                        .setExcludeByType(Exclusion.ExcludeByType.newBuilder()
                                .addTypes(Exclusion.WorkloadType.JOB)
                                .build())
                        .build())
                .build())
        assert policyID != null

        then:
        "the Deployment is still in violation and the Job is not"
        waitForViolation(DEPLOYMENT.name, policyName, WAIT_FOR_VIOLATION_TIMEOUT)
        checkForNoActiveViolations(JOB.name, policyName, WAIT_FOR_VIOLATION_TIMEOUT)

        cleanup:
        baselineID == null ?: PolicyService.deletePolicy(baselineID)
        policyID == null ?: PolicyService.deletePolicy(policyID)
    }

    private static Policy.Builder imageTagPolicy(String name) {
        return Policy.newBuilder()
                .setName(name)
                .setDescription("Workload type exclusion e2e")
                .setRationale("Workload type exclusion e2e")
                .addLifecycleStages(LifecycleStage.DEPLOY)
                .addCategories("DevOps Best Practices")
                .setDisabled(false)
                .setSeverityValue(2)
                .addPolicySections(
                        PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                PolicyOuterClass.PolicyGroup.newBuilder()
                                        .setFieldName("Image Tag")
                                        .addValues(PolicyOuterClass.PolicyValue.newBuilder()
                                                .setValue("nginx-2.0.3")
                                                .build())
                                        .build()
                        ).build()
                )
    }
}
