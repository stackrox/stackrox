import org.junit.experimental.categories.Category
import groups.BAT
import spock.lang.Unroll
import objects.Deployment
import io.stackrox.proto.api.v1.EmptyOuterClass
import io.stackrox.proto.storage.NotifierOuterClass

class IntegrationsTest extends BaseSpecification {

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

    @Category(BAT)
    def "Verify Splunk Integration"() {
       when:
        "the integration is tested"

        Deployment  deployment =
               new Deployment()
                       .setNamespace("stackrox")
                       .setName("splunk")
                       .setImage("store/splunk/enterprise:latest")
                       .addPort (8000)
                       .addPort (8088)
                       .addAnnotation("test", "annotation")
                       .setEnv([ "SPLUNK_START_ARGS": "--accept-license", "SPLUNK_USER": "root" ])
                       .addLabel("app", "splunk")
                       .setPrivilegedFlag(true)
                       .addVolume("test", "/tmp")
                       .setSkipReplicaWait(true)
                       .addImagePullSecret("stackrox")

        orchestrator.createDeployment(deployment)

        Deployment serviceDeployment = new Deployment()
               .addLabel("app", "splunk")
               .setCreateLoadBalancer(true)
               .setNamespace("stackrox")
               .setName("splunk")
               .setTargetPort(8000)
               .addPort(8000, "TCP")
               .setServiceName("splunk-http")

        orchestrator.createService(serviceDeployment)

        Deployment serviceDeploymentHec = new Deployment()
               .addLabel("app" , "splunk")
               .setCreateLoadBalancer(true)
               .setNamespace("stackrox")
               .setName("splunk")
               .setTargetPort(8088)
               .addPort(8088, "TCP")
               .setServiceName("splunk-hec")

        orchestrator.createService(serviceDeploymentHec)

 then : "the API should return an empty message or an error, depending on the config"

      cleanup:
        "remove Deployment and services"
        orchestrator.deleteDeployment(deployment)
        orchestrator.deleteService( "splunk-hec", "stackrox")
        orchestrator.deleteService( "splunk-http", "stackrox")
    }
}
