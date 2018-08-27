import static Services.getSearch
import objects.Deployment
import spock.lang.Unroll
import stackrox.generated.SearchServiceOuterClass
import org.junit.experimental.categories.Category
import groups.BAT

class GlobalSearch extends BaseSpecification {
    @Unroll
    @Category(BAT)
    def "Verify Global search : C961"(Deployment deployment, String deploymentname, String value,
                                      SearchServiceOuterClass.SearchCategory category) {
        when:
        "Create a Deployment"
        orchestrator.createDeployment(deployment)

        then:
        "Verify Search can pick query"
        assert getSearch(value, category) > 0

        cleanup:
        "Remove Deployment"
        orchestrator.deleteDeployment(deploymentname)

        where:
        "Data inputs are :"

        deployment | deploymentname | value | category
        new Deployment()
                .setName ("qadeployment")
                .setImage ("nginx")
                .addPort (22)
                .addLabel ( "app", "test" ) |
                "qadeployment" | "Deployment:qadeployment" | SearchServiceOuterClass.SearchCategory.DEPLOYMENTS

        new Deployment()
                .setName ("qaimage")
                .setImage ("nginx")
                .addPort (22)
                .addLabel ( "app", "test" ) |
                "qaimage" | "Image:docker.io/library/nginx:latest" | SearchServiceOuterClass.SearchCategory.IMAGES

        new Deployment()
                .setName ("qapolicy")
                .setImage ("nginx")
                .addPort (22)
                .addLabel ( "app", "test" ) |
                "qapolicy" | "Policy:Latest tag" | SearchServiceOuterClass.SearchCategory.POLICIES

        new Deployment()
                .setName ("qaalert")
                .setImage ("nginx")
                .addPort (22)
                .addLabel ( "app", "test" ) |
                "qaalert" | "Violation:Latest" | SearchServiceOuterClass.SearchCategory.ALERTS
    }
}
