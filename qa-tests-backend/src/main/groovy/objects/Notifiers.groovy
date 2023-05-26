package objects

import static util.Helpers.withRetry

import com.google.gson.JsonArray
import com.google.gson.JsonObject
import groovy.json.JsonSlurper
import groovy.util.logging.Slf4j

import io.stackrox.proto.storage.NotifierOuterClass
import io.stackrox.proto.storage.PolicyOuterClass.Policy

import common.Constants
import services.NotifierService
import util.Env
import util.SplunkUtil
import util.Timer

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

    void validateViolationNotification(Policy policy, Deployment deployment, boolean strictIntegrationTesting) { }

    void validateNetpolNotification(String yaml, boolean strictIntegrationTesting) { }

    void cleanup() { }

    void validateViolationResolution() { }

    String getId() {
        return notifier.id
    }

    NotifierOuterClass.Notifier getNotifier() {
        return notifier
    }
}

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

    void validateViolationNotification(Policy policy, Deployment deployment, boolean strictIntegrationTesting) {
        // TODO: Replace when https://issues.redhat.com/browse/ROX-12418 is complete
//        String policySeverity = policy.severity.valueDescriptor.toString().split("_")[0].toLowerCase()
//        try {
//            mail.login()
//        } catch (Exception e) {
//            throw new AssumptionViolatedException("Failed to login to GMAIL service... skipping test!: ", e)
//        }
//
//        log.debug "looking for a message with subject containing: ${deployment.name}"
//        Timer t = new Timer(30, 3)
//        Message[] notifications = []
//        while (!notifications && t.IsValid()) {
//            log.debug "checking for messages..."
//            SearchTerm term = new AndTerm(
//                    new FromTerm(new InternetAddress(Constants.EMAIL_NOTIFER_SENDER)),
//                    new SubjectTerm(deployment.name))
//            notifications = mail.searchMessages(term)
//            log.debug notifications*.subject.toString()
//            log.debug "matching messages: ${notifications.size()}"
//        }
//        assert notifications.length > 0 // Should be "== 1" - ROX-4542
//        assert notifications.find {
//            it.content.toString().toLowerCase().contains("severity: ${policySeverity}") }
//        assert notifications.find {
//            containsNoWhitespace(it.content.toString(), "Description:-${policy.description}") }
//        assert notifications.find {
//            containsNoWhitespace(it.content.toString(), "Rationale:-${policy.rationale}") }
//        assert notifications.find {
//            containsNoWhitespace(it.content.toString(), "Remediation:-${policy.remediation}") }
//        assert notifications.find { it.content.toString().contains("ID: ${deployment.deploymentUid}") }
//        assert notifications.find { it.content.toString().contains("Name: ${deployment.name}") }
//        assert notifications.find { it.content.toString().contains("Namespace: ${deployment.namespace}") }
//
//        // Split out so that if recipient email doesn't match, the test will print out all of the emails
//        // Otherwise it'll print notifications.toString which is unreadable
//        def recipients = notifications.collect { it.getAllRecipients()*.toString() }
//        assert recipients.find { it.find { a -> a == this.recipientEmail } }
//
//        mail.logout()
    }

    void validateNetpolNotification(String yaml, boolean strictIntegrationTesting) {
        // TODO: Replace when https://issues.redhat.com/browse/ROX-12418 is complete
//        Timer t = new Timer(30, 3)
//        try {
//            mail.login()
//        } catch (Exception e) {
//            throw new AssumptionViolatedException("Failed to login to GMAIL service... skipping test!: ", e)
//        }
//        Message[] notifications = []
//        while (!notifications && t.IsValid()) {
//            log.debug "checking for messages..."
//            SearchTerm term = new AndTerm(
//                    new FromTerm(new InternetAddress(Constants.EMAIL_NOTIFER_SENDER)),
//                    new SubjectTerm("New network policy YAML for cluster"))
//            notifications = mail.searchMessages(term)
//            log.debug notifications*.subject.toString()
//            log.debug "matching messages: ${notifications.size()}"
//        }
//        assert notifications.length > 0 // Should be "== 1" - ROX-4542
//        assert notifications.find { containsNoWhitespace(it.content.toString(), yaml) }
//        mail.logout()
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

@Slf4j
class PagerDutyNotifier extends Notifier {
    private final baseURL = "https://api.pagerduty.com/incidents"
    private final pagerdutyURL =
            baseURL + "?sort_by=created_at%3Adesc&&limit=1&service_ids[]=PRRAAWO"
    private final pagerdutyToken = Env.mustGetPagerdutyToken()
    private incidentID = null
    private incidentWatcherIndex = 0

    PagerDutyNotifier(String integrationName = "PagerDuty Test") {
        notifier = NotifierService.getPagerDutyIntegrationConfig(integrationName)
        incidentWatcherIndex = getLatestPagerDutyIncident().incidents[0].incident_number
    }

    void validateViolationNotification(Policy policy, Deployment deployment, boolean strictIntegrationTesting) {
        def newIncidents = waitForPagerDutyUpdate(incidentWatcherIndex)
        assert newIncidents != null
        assert newIncidents.incidents[0].description.contains(policy.description)
        incidentID = newIncidents.incidents[0].id
        log.debug "new pagerduty incident ID: ${incidentID}"

        incidentWatcherIndex = getLatestPagerDutyIncident().incidents[0].incident_number
    }

    void validateViolationResolution() {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            log.debug "Waiting for PagerDuty alert resolution"
            def response = getIncident(incidentID)
            if (response.incident.status == "resolved") {
                incidentID = null
                return
            }
        }
        log.debug "PagerDuty alert ${incidentID} was not resolved by StackRox"
        assert incidentID == null
    }

    void cleanup() {
        if (incidentID == null) {
            return
        }
        try {
            JsonObject incident = new JsonObject()
            incident.addProperty("id", incidentID)
            incident.addProperty("type", "incident")
            incident.addProperty("status", "resolved")
            JsonArray incidents = new JsonArray()
            incidents.add(incident)
            JsonObject jsonBody = new JsonObject()
            jsonBody.add("incidents", incidents)

            URL url = new URL(baseURL)
            HttpURLConnection con = (HttpURLConnection) url.openConnection()
            con.setRequestMethod("PUT")
            con.setRequestProperty("Content-Type", "application/json; charset=UTF-8")
            con.setRequestProperty("Accept", "application/vnd.pagerduty+json;version=2")
            con.setRequestProperty("Authorization", "Token token=${pagerdutyToken}")
            con.setRequestProperty("From", "pagerduty-test@stackrox.com")
            con.doOutput = true
            OutputStream os = con.getOutputStream()
            byte[] input = jsonBody.toString().getBytes("utf-8")
            os.write(input, 0, input.length)
            con.getInputStream()
        } catch (Exception e) {
            log.error( "Error resolving PagerDuty incident. " +
                    "This error will be ignored it is not product related", e)
        }
    }

    def resetIncidentWatcherIndex() {
        incidentWatcherIndex = getLatestPagerDutyIncident().incidents[0].incident_number
    }

    private getIncident(String id) {
        try {
            def con = (HttpURLConnection) new URL(baseURL+"/${id}").openConnection()
            con.setRequestMethod("GET")
            con.setRequestProperty("Content-Type", "application/json; charset=UTF-8")
            con.setRequestProperty("Accept", "application/vnd.pagerduty+json;version=2")
            con.setRequestProperty("Authorization", "Token token=${pagerdutyToken}")

            def jsonSlurper = new JsonSlurper()
            return jsonSlurper.parseText(con.getInputStream().getText())
        } catch (Exception e) {
            log.warn "Error getting PagerDuty incidents"
            throw e
        }
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
            log.warn "Error getting PagerDuty incidents"
            throw e
        }
    }

    private waitForPagerDutyUpdate(int preNum) {
        Timer t = new Timer(30, 3)
        while (t.IsValid()) {
            log.debug "Waiting for PagerDuty Update"
            def object = getLatestPagerDutyIncident()
            int curNum = object.incidents[0].incident_number

            if (curNum > preNum) {
                return object
            }
        }
        log.debug "Time out for Waiting for PagerDuty Update"
        return null
    }
}

@Slf4j
class SplunkNotifier extends Notifier {
    def splunkPort

    SplunkNotifier(boolean legacy, String collectorServiceName, int port, String integrationName = "Splunk Test") {
        splunkPort = port
        notifier = NotifierService.getSplunkIntegrationConfig(legacy, collectorServiceName, integrationName)
    }

    def createNotifier() {
        log.debug "validating splunk deployment is ready to accept events before creating notifier..."
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
