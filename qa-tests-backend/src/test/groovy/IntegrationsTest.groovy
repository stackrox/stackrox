import objects.Deployment
import org.junit.experimental.categories.Category
import groups.BAT
import spock.lang.Unroll
import io.stackrox.proto.api.v1.EmptyOuterClass
import io.stackrox.proto.storage.NotifierOuterClass

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
        "Verify the Clairify integration"
        assert clairifyId != null

        cleanup:
        "Remove the deployment and integration"
        orchestrator.deleteService("clairify", "stackrox")
        orchestrator.deleteDeployment(new Deployment(name: "clairify", namespace: "stackrox"))
        assert Services.deleteClairifyScanner(clairifyId)
    }
    @Category(BAT)
    def "Verify GCR Integration"() {
        when:
        "Create GCR integration"
        def gcrId = Services.addGcrRegistryAndScanner()
        println "GCR ID is: " + gcrId
        then:
        "Verify the GCR integration"
        assert gcrId != null

        cleanup:
        "Remove gcr integration"
        assert Services.deleteGcrRegistryAndScanner(gcrId)
    }

    @Unroll
    @Category([BAT])
    def "Verify Email Integration"() {
        given:
        "a configuration that is expected to work"
        NotifierOuterClass.Notifier notifier = Services.addEmailNotifier(
                "mailgun",
                disableTLS,
                startTLS,
                port
        )

        when:
        "the integration is tested"
        Object response = Services.testNotifier(notifier)

        then:
        "the API should return an empty message or an error, depending on the config"
        if (shouldSucceed) {
            assert response instanceof EmptyOuterClass.Empty
        } else {
            assert response instanceof io.grpc.StatusRuntimeException
        }

        cleanup:
        "remove notifier"
        if (notifier != null) {
            Services.deleteNotifier(notifier.id)
        }

        where:
        "data"

        port | disableTLS | startTLS | shouldSucceed

        // Port 465 tests
        // This port speaks TLS from the start.
        // (Also test null, since 465 is the default.)
        /////////////////
        // Speaking TLS should work
        465  | false      | false    | true
        null | false      | false    | true
        // Sending STARTTLS is not expected to work when already using TLS
        465  | false      | true     | false
        null | false      | true     | false
        // Speaking non-TLS to a TLS port should fail and not time out, regardless of STARTTLS (see ROX-366)
        465  | true       | false    | false
        465  | true       | true     | false
        null | true       | false    | false
        null | true       | true     | false

        // Port 587 tests
        // At MailGun, this port begins unencrypted and supports STARTTLS.
        /////////////////
        // Starting unencrypted and _not_ using STARTTLS should work
        587  | true       | false    | true
        // Starting unencrypted and using STARTTLS should work
        587  | true       | true     | true
        // Speaking TLS to a non-TLS port should fail whether you use STARTTLS or not.
        587  | false      | false    | false
        587  | false      | true     | false

        // Cannot add port 25 tests since GCP blocks outgoing
        // connections to port 25
    }

}
