package objects

import common.Constants
import groovy.json.JsonSlurper
import io.stackrox.proto.storage.NotifierOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import org.junit.Assume
import services.NotifierService
import util.Env
import util.MailService
import util.SplunkUtil
import util.Timer

import javax.mail.Message
import javax.mail.internet.InternetAddress
import javax.mail.search.AndTerm
import javax.mail.search.FromTerm
import javax.mail.search.SearchTerm
import javax.mail.search.SubjectTerm

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

    void validateViolationNotification(Policy policy, Deployment deployment, boolean strictIntegrationTesting) { }

    void validateNetpolNotification(String yaml, boolean strictIntegrationTesting) { }

    String getId() {
        return notifier.id
    }

    NotifierOuterClass.Notifier getNotifier() {
        return notifier
    }
}

class EmailNotifier extends Notifier {
    private final MailService mail =
            new MailService("imap.gmail.com", "stackrox.qa@gmail.com", Env.mustGet("EMAIL_NOTIFIER_PASSWORD"))

    EmailNotifier(
            String integrationName = "Email Test",
            disableTLS = false,
            startTLS = NotifierOuterClass.Email.AuthMethod.DISABLED,
            Integer port = null) {
        notifier = NotifierService.getEmailIntegrationConfig(integrationName, disableTLS, startTLS, port)
    }

    def deleteNotifier() {
        if (notifier?.id) {
            NotifierService.deleteNotifier(notifier.id)
            notifier = NotifierOuterClass.Notifier.newBuilder(notifier).setId("").build()
        }
        mail.logout()
    }

    void validateViolationNotification(Policy policy, Deployment deployment, boolean strictIntegrationTesting) {
        String policySeverity = policy.severity.valueDescriptor.toString().split("_")[0].toLowerCase()
        try {
            mail.login()
        } catch (Exception e) {
            if (strictIntegrationTesting) {
                throw(e)
            }
            Assume.assumeNoException("Failed to login to GMAIL service... skipping test!: ", e)
        }

        Timer t = new Timer(30, 3)
        Message[] notifications = []
        while (!notifications && t.IsValid()) {
            println "checking for messages..."
            SearchTerm term = new AndTerm(
                    new FromTerm(new InternetAddress(Constants.EMAIL_NOTIFER_SENDER)),
                    new SubjectTerm(deployment.deploymentUid))
            notifications = mail.searchMessages(term)
            println notifications*.subject.toString()
            println "matching messages: ${notifications.size()}"
        }
        assert notifications.length > 0 // Should be "== 1" - ROX-4542
        assert notifications.find {
            it.content.toString().toLowerCase().contains("severity: ${policySeverity}") }
        assert notifications.find {
            containsNoWhitespace(it.content.toString(), "Description:-${policy.description}") }
        assert notifications.find {
            containsNoWhitespace(it.content.toString(), "Rationale:-${policy.rationale}") }
        assert notifications.find {
            containsNoWhitespace(it.content.toString(), "Remediation:-${policy.remediation}") }
        assert notifications.find { it.content.toString().contains("ID: ${deployment.deploymentUid}") }
        assert notifications.find { it.content.toString().contains("Name: ${deployment.name}") }
        assert notifications.find { it.content.toString().contains("Namespace: ${deployment.namespace}") }
        mail.logout()
    }

    void validateNetpolNotification(String yaml, boolean strictIntegrationTesting) {
        Timer t = new Timer(30, 3)
        try {
            mail.login()
        } catch (Exception e) {
            if (strictIntegrationTesting) {
                throw(e)
            }
            Assume.assumeNoException("Failed to login to GMAIL service... skipping test!: ", e)
        }
        Message[] notifications = []
        while (!notifications && t.IsValid()) {
            println "checking for messages..."
            SearchTerm term = new AndTerm(
                    new FromTerm(new InternetAddress(Constants.EMAIL_NOTIFER_SENDER)),
                    new SubjectTerm("New network policy YAML for cluster"))
            notifications = mail.searchMessages(term)
            println notifications*.subject.toString()
            println "matching messages: ${notifications.size()}"
        }
        assert notifications.length > 0 // Should be "== 1" - ROX-4542
        assert notifications.find { containsNoWhitespace(it.content.toString(), yaml) }
        mail.logout()
    }
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

    void validateViolationNotification(Policy policy, Deployment deployment, boolean strictIntegrationTesting) {
        def get = new URL("http://localhost:8080").openConnection()
        def jsonSlurper = new JsonSlurper()
        def object = jsonSlurper.parseText(get.getInputStream().getText())
        def generic = object[-1]

        assert generic["headers"]["Headerkey"] == ["headervalue"]
        assert generic["headers"]["Content-Type"] == ["application/json"]
        assert generic["headers"]["Authorization"] == ["Basic YWRtaW46YWRtaW4="]
        assert generic["data"]["fieldkey"] == "fieldvalue"
        assert generic["data"]["alert"]["policy"]["name"] == policy.name
        assert generic["data"]["alert"]["deployment"]["name"] == deployment.name
    }

    void validateNetpolNotification(String yaml, boolean strictIntegrationTesting) {
        def get = new URL("http://localhost:8080").openConnection()
        def jsonSlurper = new JsonSlurper()
        def object = jsonSlurper.parseText(get.getInputStream().getText())
        def generic = object[-1]

        assert generic["headers"]["Headerkey"] == ["headervalue"]
        assert generic["headers"]["Content-Type"] == ["application/json"]
        assert generic["headers"]["Authorization"] == ["Basic YWRtaW46YWRtaW4="]
        assert generic["data"]["fieldkey"] == "fieldvalue"
        assert generic["data"]["networkpolicy"]["yaml"] == yaml
    }
}

class SlackNotifier extends Notifier {
    SlackNotifier(String integrationName = "Slack Test") {
        notifier = NotifierService.getSlackIntegrationConfig(integrationName)
    }
}

class JiraNotifier extends Notifier {
    JiraNotifier(String integrationName = "Jira Test") {
        notifier = NotifierService.getJiraIntegrationConfig(integrationName)
    }
}

class TeamsNotifier extends Notifier {
    TeamsNotifier(String integrationName = "Teams Test") {
        notifier = NotifierService.getTeamsIntegrationConfig(integrationName)
    }
}

class PagerDutyNotifier extends Notifier {
    private final pagerdutyURL = "https://api.pagerduty.com/incidents?sort_by=created_at%3Adesc&&limit=1"
    private final pagerdutyToken = "qWT6sXfp_Lvz-pddxcCg"
    private incidentWatcherIndex = 0

    PagerDutyNotifier(String integrationName = "PagerDuty Test") {
        notifier = NotifierService.getPagerDutyIntegrationConfig(integrationName)
        incidentWatcherIndex = getLatestPagerDutyIncident().incidents[0].incident_number
    }

    void validateViolationNotification(Policy policy, Deployment deployment, boolean strictIntegrationTesting) {
        def newIncidents = waitForPagerDutyUpdate(incidentWatcherIndex)
        assert newIncidents != null
        assert newIncidents.incidents[0].description.contains(policy.description)
    }

    def resetIncidentWatcherIndex() {
        incidentWatcherIndex = getLatestPagerDutyIncident().incidents[0].incident_number
    }

    private getLatestPagerDutyIncident() {
        try {
            def con = (HttpURLConnection) new URL(pagerdutyURL).openConnection()
            con.setRequestMethod("GET")
            con.setRequestProperty("Content-Type", "application/json; charset=UTF-8")
            con.setRequestProperty("Accept", "application/vnd.pagerduty+json;version=2")
            con.setRequestProperty("Authorization", "Token token=${pagerdutyToken}")

            def jsonSlurper = new JsonSlurper()
            return jsonSlurper.parseText(con.getInputStream().getText())
        } catch (Exception e) {
            println "Error getting PagerDuty incidents"
            throw e
        }
    }

    private waitForPagerDutyUpdate(int preNum) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            println "Waiting for PagerDuty Update"
            def object = getLatestPagerDutyIncident()
            int curNum = object.incidents[0].incident_number

            if (curNum > preNum) {
                return object
            }
        }
        println "Time out for Waiting for PagerDuty Update"
        return null
    }
}

class SplunkNotifier extends Notifier {
    def splunkPort

    SplunkNotifier(boolean legacy, String collectorServiceName, int port, String integrationName = "Splunk Test") {
        splunkPort = port
        notifier = NotifierService.getSplunkIntegrationConfig(legacy, collectorServiceName, integrationName)
    }

    def createNotifier() {
        println "validating splunk deployment is ready to accept events before creating notifier..."
        withRetry(20, 2) {
            SplunkUtil.createSearch(splunkPort)
        }
        notifier = NotifierService.addNotifier(notifier)
    }

    void validateViolationNotification(Policy policy, Deployment deployment, boolean strictIntegrationTesting) {
        def response = SplunkUtil.waitForSplunkAlerts(splunkPort, 30)

        assert response.find { it.deployment.id == deployment.deploymentUid }
        assert response.find { it.deployment.name == deployment.name }
        assert response.find { it.deployment.namespace == deployment.namespace }
        assert response.find { it.deployment.type == "Deployment" }
        assert response.find { it.policy.name == policy.name }
        assert response.find { it.policy.description == policy.description }
        assert response.find { it.policy.remediation == policy.remediation }
        assert response.find { it.policy.rationale == policy.rationale }
    }
}
