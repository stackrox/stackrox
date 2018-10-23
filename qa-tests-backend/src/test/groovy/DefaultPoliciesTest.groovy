import static Services.getPolicies
import static Services.getViolations
import static Services.waitForViolation
import stackrox.generated.AlertServiceOuterClass

import groups.BAT
import org.junit.experimental.categories.Category
import spock.lang.Unroll
import objects.Deployment
import java.util.stream.Collectors

class DefaultPoliciesTest extends BaseSpecification {

    // Deployment names
    static final private String NGINX_LATEST = "qadefpolnginxlatest"
    static final private String STRUTS = "qadefpolstruts"
    static final private String SSL_TERMINATOR = "qadefpolsslterm"
    static final private String NGINX_1_10 = "qadefpolnginx110"
    static final private String K8S_DASHBOARD = "kubernetes-dashboard"

    static final private List<Deployment> DEPLOYMENTS = [
        new Deployment()
            .setName (NGINX_LATEST)
            .setImage ("nginx")
            .addPort (22)
            .addLabel ("app", "test"),
        new Deployment()
            .setName(STRUTS)
            .setImage("apollo-dtr.rox.systems/legacy-apps/struts-app:latest")
            .addLabel("app", "test"),
        new Deployment()
            .setName(SSL_TERMINATOR)
            .setImage("apollo-dtr.rox.systems/legacy-apps/ssl-terminator:latest")
            .addLabel("app", "test"),
        new Deployment()
            .setName(NGINX_1_10)
            .setImage("nginx:1.10")
            .addLabel("app", "test"),
    ]

    def setupSpec() {
        orchestrator.batchCreateDeployments(DEPLOYMENTS)
        for (Deployment deployment : DEPLOYMENTS) {
            assert Services.waitForDeployment(deployment.getDeploymentUid())
        }
    }

    def cleanupSpec() {
        for (Deployment deployment : DEPLOYMENTS) {
            orchestrator.deleteDeployment(deployment.getName())
        }
    }

    @Unroll
    @Category(BAT)
    def "Verify policy #policyName is triggered" (String policyName, String deploymentName,
                                                  String testId) {
        when:
        "Validate if policy is present"
        assert getPolicies().stream()
                .filter { f -> f.getName() == policyName }
                .collect(Collectors.toList()).size() == 1

        then:
        "Verify Violation for #policyName is triggered"
        assert waitForViolation(deploymentName,  policyName, 30)

        where:
        "Data inputs are:"

        policyName                                    | deploymentName | testId

        "Latest tag"                                  | NGINX_LATEST   | ""

        "Container Port 22"                           | NGINX_LATEST   | "C311"

        "Apache Struts: CVE-2017-5638"                | STRUTS         | "C938"

        "Heartbleed: CVE-2014-0160"                   | SSL_TERMINATOR | "C947"

        "Wget in Image"                               | STRUTS         | "C939"

        "90-Day Image Age"                            | STRUTS         | "C810"

        "Aptitude Package Manager (apt) in Image"     | STRUTS         | "C931"

        "30-Day Scan Age"                             | STRUTS         | "C941"

        "CVSS >= 7"                                   | STRUTS         | "C933"

        "Shellshock: CVE-2014-6271"                   | SSL_TERMINATOR | "C948"

        "Curl in Image"                               | STRUTS         | "C948"

        "DockerHub NGINX 1.10"                        | NGINX_1_10     | "C823"

        "Kubernetes Dashboard Deployed"               | K8S_DASHBOARD  | ""
    }

    @Category(BAT)
    def "Verify that StackRox services don't trigger alerts"() {
        expect:
        "Verify policies are not violated within the stackrox namespace"
        getViolations(
                AlertServiceOuterClass.ListAlertsRequest.newBuilder().setQuery("Namespace:stackrox").build()
        ).size() == 0
    }

    @Category(BAT)
    def "Verify that built-in services don't trigger unexpected alerts"() {
        expect:
        "Verify unexpected policies are not violated within the kube-system namespace"
        getViolations(
          AlertServiceOuterClass.ListAlertsRequest.newBuilder()
            .setQuery("Namespace:kube-system+Policy:!Kubernetes Dashboard").build()
        ).size() == 0
    }
}
