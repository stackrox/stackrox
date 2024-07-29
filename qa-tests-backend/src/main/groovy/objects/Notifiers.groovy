package objects

import groovy.json.JsonSlurper
import groovy.util.logging.Slf4j

import io.stackrox.proto.storage.NotifierOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.Policy

import common.Constants
import services.NotifierService
import util.SplunkUtil

@Slf4j
class Notifier {
    NotifierOuterClass.Notifier notifier

    def createNotifier() {
        notifier = NotifierService.addNotifier(notifier)
    }

    def deleteNotifier() {
        if (notifier?.id) {
            NotifierService.deleteNotifier(notifier.id)
            notifier = NotifierOuterClass.Notifier.newBuilder(notifier).setId("").build()
        }
    }

    def testNotifier() {
        return NotifierService.testNotifier(notifier)
    }

    void validateViolationNotification(Policy policy, Deployment deployment, boolean strictIntegrationTesting) {
        log.debug("Nothing to validate")
    }

    void validateNetpolNotification(String yaml, boolean strictIntegrationTesting) {
        log.debug("Nothing to validate")
    }

    String getId() {
        return notifier.id
    }

    NotifierOuterClass.Notifier getNotifier() {
        return notifier
    }
}

@Slf4j
class EmailNotifier extends Notifier {
    private final String recipientEmail

    EmailNotifier(
            String integrationName = "Email Test",
            String server,
            boolean sendAuthCreds = true,
            boolean disableTLS = false,
            NotifierOuterClass.Email.AuthMethod startTLS = NotifierOuterClass.Email.AuthMethod.DISABLED,
            String recipientEmail = Constants.EMAIL_NOTIFIER_RECIPIENT) {
        this.recipientEmail = recipientEmail
        notifier = NotifierService.getEmailIntegrationConfig(integrationName, server,
                sendAuthCreds, disableTLS, startTLS)
    }

    def deleteNotifier() {
        if (notifier?.id) {
            NotifierService.deleteNotifier(notifier.id)
            notifier = NotifierOuterClass.Notifier.newBuilder(notifier).setId("").build()
        }
    }

    //TODO(ROX-12418): Implement validateViolationNotification and validateNetpolNotification)
}

class GenericNotifier extends Notifier {
    GenericNotifier(
            String integrationName = "Generic Test",
            boolean enableTLS = false,
            String caCert = "",
            boolean skipTLSVerification = false,
            boolean auditLoggingEnabled = false) {
        notifier = NotifierService.getWebhookIntegrationConfig(
                integrationName, enableTLS, caCert, skipTLSVerification, auditLoggingEnabled)
    }

    static getMostRecentViolationAndValidateCommonFields() {
        def get = new URL("http://localhost:8080").openConnection()
        def jsonSlurper = new JsonSlurper()
        def object = jsonSlurper.parseText(get.getInputStream().getText())
        def generic = object[-1]
        assert generic["headers"]["Headerkey"] == ["headervalue"]
        assert generic["headers"]["Content-Type"] == ["application/json"]
        assert generic["headers"]["Authorization"] == ["Basic YWRtaW46YWRtaW4="]
        assert generic["data"]["fieldkey"] == "fieldvalue"

        return generic
    }

    void validateViolationNotification(Policy policy, Deployment deployment, boolean strictIntegrationTesting) {
        def generic = getMostRecentViolationAndValidateCommonFields()

        assert generic["data"]["alert"]["policy"]["name"] == policy.name
        assert generic["data"]["alert"]["deployment"]["name"] == deployment.name
    }

    void validateNetpolNotification(String yaml, boolean strictIntegrationTesting) {
        def generic = getMostRecentViolationAndValidateCommonFields()

        assert generic["data"]["networkpolicy"]["yaml"] == yaml
    }
}

class SlackNotifier extends Notifier {
    SlackNotifier(String integrationName = "Slack Test", String labelKey = "#acs-slack-integration-testing") {
        notifier = NotifierService.getSlackIntegrationConfig(integrationName, labelKey)
    }
}

@Slf4j
class SplunkNotifier extends Notifier {
    def splunkPort

    SplunkNotifier(String collectorServiceName, int port, String integrationName = "Splunk Test") {
        splunkPort = port
        def hecToken = SplunkUtil.createHECToken(splunkPort)
        log.info("Using HEC ingest token: ${hecToken}")
        notifier = NotifierService.getSplunkIntegrationConfig(collectorServiceName, integrationName, hecToken)
    }

    def createNotifier() {
        notifier = NotifierService.addNotifier(notifier)
    }

    void validateViolationNotification(Policy policy, Deployment deployment, boolean strictIntegrationTesting) {
        def response = SplunkUtil.waitForSplunkAlerts(splunkPort, "search sourcetype=stackrox-alert " + policy.name)

        log.info("Verifying data in Splunk")
        assert response.find { it.deployment.id == deployment.deploymentUid }
        assert response.find { it.deployment.name == deployment.name }
        assert response.find { it.deployment.namespace == deployment.namespace }
        assert response.find { it.deployment.type == "Deployment" }
        assert response.find { it.policy.name == policy.name }
    }
}

@Slf4j
class SyslogNotifier extends Notifier {
    SyslogNotifier(String serviceName, int port, String integrationName = "Syslog Test") {
        notifier = NotifierService.getSyslogIntegrationConfig(serviceName, port, integrationName)
    }

    def createNotifier() {
        notifier = NotifierService.addNotifier(notifier)
    }

    def testNotifier() {
        return NotifierService.testNotifier(notifier)
    }
}
