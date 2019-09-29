import static Services.getPolicies
import static Services.waitForViolation
import groups.BAT
import groups.SMOKE
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
                    .setImage ("nginx@sha256:204a9a8e65061b10b92ad361dd6f406248404fe60efd5d6a8f2595f18bb37aad")
                    .addLabel ( "app", "test" )
                    .setCommand(["sh" , "-c" , "apt-get -y update && sleep 600"]),
            new Deployment()
                    .setName (DEPLOYMENTAPT)
                    .setImage ("redis@sha256:96be1b5b6e4fe74dfe65b2b52a0fee254c443184b34fe448f3b3498a512db99e")
                    .addLabel ( "app", "test" )
                    .setCommand(["sh" , "-c" , "apt -y update && sleep 600"]),
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
    @Category([BAT, SMOKE])
    def "Verify runtime policy : #policyName can be triggered - #depName"() {
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

        DEPLOYMENTAPTGET | "Ubuntu Package Manager Execution"

        DEPLOYMENTAPT | "Ubuntu Package Manager Execution"
    }

}
