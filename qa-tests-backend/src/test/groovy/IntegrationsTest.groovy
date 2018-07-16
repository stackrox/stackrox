import org.junit.Test

class IntegrationsTest extends BaseSpecification {

    @Test
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
