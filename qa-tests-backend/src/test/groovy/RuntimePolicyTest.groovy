import static Services.getPolicies
import static Services.waitForResolvedViolation
import static Services.waitForViolation

import java.util.stream.Collectors

import io.stackrox.proto.storage.PolicyOuterClass

import objects.Deployment
import services.PolicyService

import spock.lang.Tag
import spock.lang.Unroll

class RuntimePolicyTest extends BaseSpecification  {
    static final private String DEPLOYMENTAPTGET = "runtimenginx"
    static final private String DEPLOYMENTAPT = "runtimeredis"

    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                    .setName (DEPLOYMENTAPTGET)
                    .setImage ("quay.io/rhacs-eng/qa-multi-arch:nginx-"+
                               "204a9a8e65061b10b92ad361dd6f406248404fe60efd5d6a8f2595f18bb37aad")
                    .addLabel ( "app", "test" )
                    .setCommand(["sh" , "-c" , "apt-get -y update || true && sleep 600"]),
            new Deployment()
                    .setName (DEPLOYMENTAPT)
                    .setImage ("quay.io/rhacs-eng/qa-multi-arch:redis-"+
                               "96be1b5b6e4fe74dfe65b2b52a0fee254c443184b34fe448f3b3498a512db99e")
                    .addLabel ( "app", "test" )
                    .setCommand(["sh" , "-c" , "apt -y update || true && sleep 600"]),
    ]

    static final private DEPLOYMENTREMOVAL =  new Deployment()
            .setName ("runtimeremoval")
            .setImage ("quay.io/rhacs-eng/qa-multi-arch:redis-" +
                    "96be1b5b6e4fe74dfe65b2b52a0fee254c443184b34fe448f3b3498a512db99e")
            .addLabel ( "app", "test" )
            .setCommand(["sh" , "-c" , "apt -y update || true && sleep 600"])

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
    @Tag("BAT")
    @Tag("SMOKE")
    @Tag("PZ")
    def "Verify runtime policy : #policyName can be triggered - #depName"() {
        when:
        "Validate if policy is present"
        assert getPolicies().stream()
                .filter { f -> f.getName() == policyName }
                .collect(Collectors.toList()).size() == 1

        then:
        "Verify Violation is triggered"
        assert waitForViolation(depName, policyName, 66)

        where:
        "Data inputs are :"

        depName | policyName

        DEPLOYMENTAPTGET | "Ubuntu Package Manager Execution"

        DEPLOYMENTAPT | "Ubuntu Package Manager Execution"
    }

    @Unroll
    @Tag("BAT")
    @Tag("PZ")
    def "Verify runtime alert violations are resolved once policy is removed"() {
        given:
        "Create runtime alert"
        def policy = PolicyOuterClass.Policy.newBuilder()
                .addLifecycleStages(PolicyOuterClass.LifecycleStage.RUNTIME)
                .addCategories("Test")
                .setDisabled(false)
                .setSeverityValue(2)
                .setName("runtime-removal-policy")
                .setEventSource(PolicyOuterClass.EventSource.DEPLOYMENT_EVENT)
                .addPolicySections(PolicyOuterClass.PolicySection.newBuilder()
                .addPolicyGroups(
                    PolicyOuterClass.PolicyGroup.newBuilder()
                            .setFieldName("Process Name")
                            .setBooleanOperator(PolicyOuterClass.BooleanOperator.AND)
                           .addValues(
                           PolicyOuterClass.PolicyValue.newBuilder()
                                .setValue("apt")
                                .build()
                    )
                ).build())
                .build()
        def policyID = PolicyService.createNewPolicy(policy)
        orchestrator.createDeployment(DEPLOYMENTREMOVAL)

        when:
        "Verify violation triggered then remove the policy"
        assert waitForViolation(DEPLOYMENTREMOVAL.name, policy.name, 66)
        PolicyService.deletePolicy(policyID)

        then:
        "Verify Violation is removed"
        assert waitForResolvedViolation(DEPLOYMENTREMOVAL.name, policy.name, 66)

        cleanup:
        orchestrator.deleteDeployment(DEPLOYMENTREMOVAL)
    }

}
