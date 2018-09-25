import static Services.getPolicies
import static Services.waitForViolation
import org.junit.Test
import groups.BAT
import org.junit.experimental.categories.Category
import spock.lang.Unroll
import objects.Deployment
import java.util.stream.Collectors

class DefaultPoliciesTest extends BaseSpecification {
    @Test
    @Category(BAT)
    def "Verify policy life cycle"() {
        String deployName = "qalifecycle"
        Deployment deployment = new Deployment()
                .setName(deployName)
                .setImage("nginx:latest")
                .addLabel ( "app", "test" )

        when:
        "Create a custom policy - Using image latest template"
        String policyID = Services.addLatestTagPolicy()
        sleep(5000)
        println("Policy ID :" + policyID)
        assert policyID != null

        and:
        "Create a deployment"
        orchestrator.createDeployment(deployment)

        then:
        "Verify the custom policy is triggered"
        assert waitForViolation(deployName, "qaTestLifeCycle", 1800)

        cleanup:
        "Remove the policy and deployment"
        Services.deletePolicy(policyID)
        orchestrator.deleteDeployment(deployName)
    }

    @Unroll
    @Category(BAT)
    def "Verify policy #policyname is triggered" (String policyname, Deployment deployment,
                                                  String testId) {
        when:
        "Create a Deployment"
        orchestrator.createDeployment(deployment)

        and:
        "Validate if Violation and Deployment is present"
        assert getPolicies().stream()
                .filter { f -> f.getName() == policyname }
                .collect(Collectors.toList()).size() == 1

        then:
        "Verify Violation #policyname is triggered"
        assert waitForViolation(deployment.getName(),  policyname, 30)

        cleanup:
        "Remove Deployment #deploymentName"
        orchestrator.deleteDeployment(deployment.getName())

        where:
        "Data inputs are :"

        policyname | deployment | testId

        "Container Port 22" | new Deployment()
                .setName ("qaport22")
                .setImage ("nginx")
                .addPort (22)
                .addLabel ( "app", "test" ) | "C311"

        "Apache Struts: CVE-2017-5638" | new Deployment()
                .setName ( "qacve" )
                .setImage ( "apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
                .addLabel ( "app", "test" ) | "C938"

        "Heartbleed: CVE-2014-0160" | new Deployment()
                .setName ("qaheartbleed")
                .setImage ("apollo-dtr.rox.systems/legacy-apps/ssl-terminator:latest")
                .addLabel ( "app", "test" ) | "C947"

        "Wget in Image" | new Deployment()
                .setName ("qawget")
                .setImage ("apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
                .addLabel ( "app", "test" ) | "C939"

        "90-Day Image Age" | new Deployment()
                .setName ("qa90days" )
                .setImage ("apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
                .addLabel ("app", "test" ) | "C810"

        "Aptitude Package Manager (apt) in Image" | new Deployment()
                .setName ("qaapt" )
                .setImage ("apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
                .addLabel ( "app", "test" ) | "C931"

        "30-Day Scan Age" | new Deployment()
                .setName ( "qa30days" )
                .setImage ( "apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
                .addLabel ( "app", "test" ) | "C941"

        "Maximum CVSS >= 7" | new Deployment()
                .setName ( "qacvss" )
                .setImage ( "apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
                .addLabel ( "app", "test" ) | "C933"

        "Shellshock: CVE-2014-6271" | new Deployment()
                .setName ("qashellshock" )
                .setImage ("apollo-dtr.rox.systems/legacy-apps/ssl-terminator")
                .addLabel ( "app", "test" ) | "C948"

        "Curl in Image" | new Deployment()
                .setName ("qacurl")
                .setImage ("apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
                .addLabel ( "app", "test" ) | "C948"

        "DockerHub NGINX 1.10" | new Deployment()
                .setName ("qanginx")
                .setImage ("nginx:1.10")
                .addLabel ( "app", "test" ) | "C823"
    }

}
