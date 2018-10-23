import static Services.getPolicies
import static Services.waitForViolation
import groups.BAT
import objects.Deployment
import org.junit.experimental.categories.Category
import spock.lang.Unroll
import java.util.stream.Collectors

class RuntimePolicyTest extends BaseSpecification  {
    static final private String DEPLOYMENTAPTGET = "runtimenginx"
    static final private String DEPLOYMENTAPT = "runtimeredis"

    static final private List<Deployment> DEPLOYMENTS = [
            new Deployment()
                    .setName (DEPLOYMENTAPTGET)
                    .setImage ("nginx@sha256:9ad0746d8f2ea6df3a17ba89eca40b48c47066dfab55a75e08e2b70fc80d929e")
                    .addLabel ( "app", "test" )
                    .setCommand(["sh" , "-c" , "apt-get -y update"]),
            new Deployment()
                    .setName (DEPLOYMENTAPT)
                    .setImage ("redis@sha256:911f976312f503692709ad9534f15e2564a0967f2aa6dd08a74c684fb1e53e1a")
                    .addLabel ( "app", "test" )
                    .setCommand(["sh" , "-c" , "apt -y update"]),
    ]

    def setupSpec() {
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deploymentId : DEPLOYMENTS) {
            assert Services.waitForDeployment(deploymentId.getDeploymentUid())
        }
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment.getName())
        }
    }

    @Unroll
    @Category(BAT)
    def "Verify runtime policy : #policyName can be triggered"() {
        when:
        "Validate if policy is present"
        assert getPolicies().stream()
                .filter { f -> f.getName() == policyName }
                .collect(Collectors.toList()).size() == 1

        then:
        "Verify Violation is triggered"
        assert waitForViolation(depName, policyName, 60)

        where:
        "Data inputs are :"

        depName | policyName

        DEPLOYMENTAPTGET | "apt-get Execution"

        DEPLOYMENTAPT | "apt Execution"

        DEPLOYMENTAPTGET | "dpkg Execution"
    }

}
