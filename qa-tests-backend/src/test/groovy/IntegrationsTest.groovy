import groups.Integration
import org.junit.experimental.categories.Category
import groups.BAT

class IntegrationsTest extends BaseSpecification {
    @Category(BAT)
    def "Verify Clairify Integration"() {
        when:
        "Create Clairify deployment"
        orchestrator.createClairifyDeployment()
        sleep(20000)

        and:
        "Create Clairify integration"
        def clairifyId = Services.addClairifyScanner(orchestrator.getClairifyEndpoint())

        then:
        "Verify the integration succeed"
        assert clairifyId != null

        cleanup:
        "Remove the deployment and integration"
        orchestrator.deleteService("clairify", "stackrox")
        orchestrator.deleteDeployment("clairify", "stackrox")
        if (clairifyId != null) {
            Services.deleteClairifyScanner(clairifyId)
        }
    }
}
