import groups.Integration
import org.junit.experimental.categories.Category
import groups.BAT
import spock.lang.Unroll
import stackrox.generated.EmptyOuterClass
import stackrox.generated.NotifierServiceOuterClass

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

    @Unroll
    @Category([BAT])
    def "Verify Email Integrations"() {
        when:
        "Create email integration"
        NotifierServiceOuterClass.Notifier notifier = Services.addEmailNotifier(
                "mailgun",
                disableTLS,
                startTLS,
                port
        )

        then:
        "test integration"
        assert Services.testNotifier(notifier) instanceof EmptyOuterClass.Empty

        cleanup:
        "remove notifier"
        if (notifier != null) {
            Services.deleteNotifier(notifier.id)
        }

        where:
        "data"

        port | disableTLS | startTLS

        //  Port 465 tests
        465  | false      | false
        //465  | true       | false    ROX-366
        //465  | true       | true     ROX-366

        // null port (default 465)
        null | false      | false
        //null | true       | false    ROX-366
        //null | true       | true     ROX-366

        // Port 587 tests
        587  | true       | false
        587  | true       | true

        // Cannot add port 25 tests since GCP blocks outgoing
        // connections to port 25
    }
}
