package objects

import common.Constants
import groovy.json.JsonSlurper
import io.stackrox.proto.storage.NotifierOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.Policy
import services.ClusterService
import services.NotifierService
import util.MailService
import util.SplunkUtil
import util.Timer

import javax.mail.Message

class Notifier {
    NotifierOuterClass.Notifier notifier

    def createNotifier() {
        notifier = NotifierService.addNotifier(notifier)
    }

    def deleteNotifier() {
        if (notifier?.id) {
            NotifierService.deleteNotifier(notifier.id)
        }
    }

    def testNotifier() {
        return NotifierService.testNotifier(notifier)
    }

    void validateViolationNotification(Policy policy, Deployment deployment) { }

    void validateNetpolNotification(String yaml) { }

    String getId() {
        return notifier.id
    }

    NotifierOuterClass.Notifier getNotifier() {
        return notifier
    }
}

class EmailNotifier extends Notifier {
    private final MailService mail = new MailService()

    EmailNotifier(
            String integrationName = "Email Test",
            disableTLS = false,
            startTLS = NotifierOuterClass.Email.AuthMethod.DISABLED,
            Integer port = null) {
        mail.login("imap.gmail.com", "stackrox.qa@gmail.com", System.getenv("EMAIL_NOTIFIER_PASSWORD"))
        notifier = NotifierService.getEmailIntegrationConfig(integrationName, disableTLS, startTLS, port)
    }

    void validateViolationNotification(Policy policy, Deployment deployment) {
        String policySeverity = policy.severity.valueDescriptor.toString().split("_")[0].toLowerCase()

        Timer t = new Timer(10, 3)
        Message[] notifications = []
        while (!notifications && t.IsValid()) {
            notifications = mail.getMessagesFromSender(Constants.EMAIL_NOTIFER_FULL_FROM).findAll {
                it.subject.contains(policy.name) &&
                        it.subject.contains(deployment.name) }
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
    }

    void validateNetpolNotification(String yaml) {
        Timer t = new Timer(10, 3)
        Message[] notifications = []
        while (!notifications && t.IsValid()) {
            notifications = mail.getMessagesFromSender(Constants.EMAIL_NOTIFER_FULL_FROM).findAll {
                it.subject.contains("New network policy YAML for cluster") }
        }
        assert notifications.length > 0 // Should be "== 1" - ROX-4542
        assert notifications.find { containsNoWhitespace(it.content.toString(), yaml) }
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

    void validateViolationNotification(Policy policy, Deployment deployment) {
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

    void validateNetpolNotification(String yaml) {
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

    void validateViolationNotification(Policy policy, Deployment deployment) {
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
    def splunkLbIp = ""

    SplunkNotifier(boolean legacy, String lbIp, String integrationName = "Splunk Test") {
        splunkLbIp = lbIp
        notifier = NotifierService.getSplunkIntegrationConfig(legacy, integrationName)
    }

    void validateViolationNotification(Policy policy, Deployment deployment) {
        def response = SplunkUtil.waitForSplunkAlerts(splunkLbIp, 60)

        assert response.get("offset") == 0
        assert response.get("preview") == false
        assert response.get("namespace") == deployment.namespace
        assert response.get("name") == deployment.name
        assert response.get("type") == "Deployment"
        assert response.get("clusterName") == ClusterService.getCluster().name
        assert response.get("policy") == policy.name
        assert response.get("sourcetype") == "_json"
        assert response.get("source") == "stackrox"
    }
}
