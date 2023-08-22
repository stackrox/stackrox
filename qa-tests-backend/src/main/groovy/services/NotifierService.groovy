package services

import common.Constants
import groovy.util.logging.Slf4j
import io.stackrox.proto.api.v1.NotifierServiceGrpc
import io.stackrox.proto.api.v1.NotifierServiceOuterClass
import io.stackrox.proto.storage.Common
import io.stackrox.proto.storage.NotifierOuterClass
import util.Env
import util.MailServer

@Slf4j
class NotifierService extends BaseService {
    // FIXME(ROX-7589): this should be secret
    // private static final PAGERDUTY_API_KEY = Env.mustGetPagerdutyApiKey()
    private static final String PAGERDUTY_API_KEY = null

    static getNotifierClient() {
        return NotifierServiceGrpc.newBlockingStub(getChannel())
    }

    static addNotifier(NotifierOuterClass.Notifier notifier) {
        return getNotifierClient().postNotifier(notifier)
    }

    static testNotifier(NotifierOuterClass.Notifier notifier) {
        try {
            getNotifierClient().testNotifier(notifier)
            return true
        } catch (Exception e) {
            log.error("error testing notifier", e)
            return false
        }
    }

    static deleteNotifier(String id) {
        try {
            getNotifierClient().deleteNotifier(
                    NotifierServiceOuterClass.DeleteNotifierRequest.newBuilder()
                            .setId(id)
                            .setForce(true)
                            .build()
            )
        } catch (Exception e) {
            log.error("error deleting notifier", e)
        }
    }

    static NotifierOuterClass.Notifier getEmailIntegrationConfig(
            String name,
            String server,
            boolean sendAuthCreds = true,
            boolean disableTLS = false,
            NotifierOuterClass.Email.AuthMethod startTLS = NotifierOuterClass.Email.AuthMethod.DISABLED) {
        NotifierOuterClass.Notifier.Builder builder =
                NotifierOuterClass.Notifier.newBuilder()
                        .setEmail(NotifierOuterClass.Email.newBuilder())

        def emailBuilder = builder.getEmailBuilder()
                .setSender(Constants.EMAIL_NOTIFER_SENDER)
                .setFrom(Constants.EMAIL_NOTIFER_FROM)
                .setDisableTLS(disableTLS)
                .setStartTLSAuthMethod(startTLS)

        if (sendAuthCreds) {
            emailBuilder.setUsername(MailServer.MAILSERVER_USER).setPassword(MailServer.MAILSERVER_PASS)
        } else {
            emailBuilder.setAllowUnauthenticatedSmtp(!sendAuthCreds)
        }

        builder
                .setType("email")
                .setName(name)
                .setLabelKey("email_label")
                .setLabelDefault(Constants.EMAIL_NOTIFIER_RECIPIENT)
                .setUiEndpoint(getStackRoxEndpoint())
                .setEmail(emailBuilder)
        builder.getEmailBuilder().setServer(server)
        return builder.build()
    }

    static NotifierOuterClass.Notifier getWebhookIntegrationConfig(
            String name,
            Boolean enableTLS,
            String caCert,
            Boolean skipTLSVerification,
            Boolean auditLoggingEnabled) {
        NotifierOuterClass.GenericOrBuilder genericBuilder = NotifierOuterClass.Generic.newBuilder()
                .setEndpoint("http://webhookserver.stackrox:8080")
                .setCaCert(caCert)
                .setSkipTLSVerify(skipTLSVerification)
                .setAuditLoggingEnabled(auditLoggingEnabled)
                .setUsername("admin")
                .setPassword("admin")
                .addHeaders(
                        Common.KeyValuePair.newBuilder().setKey("headerkey").setValue("headervalue").build()
                )
                .addExtraFields(Common.KeyValuePair.newBuilder().setKey("fieldkey").setValue("fieldvalue").build())
        if (enableTLS) {
            genericBuilder.setEndpoint("https://webhookserver.stackrox:8443")
        }

        return NotifierOuterClass.Notifier.newBuilder()
                .setName(name)
                .setType("generic")
                .setGeneric(genericBuilder.build())
                .setUiEndpoint("localhost:8000")
                .build()
    }

    static NotifierOuterClass.Notifier getSlackIntegrationConfig(String name, String labelKey) {
        return NotifierOuterClass.Notifier.newBuilder()
                .setType("slack")
                .setName(name)
                .setLabelKey(labelKey)
                .setLabelDefault(Env.mustGetSlackMainWebhook())
                .setUiEndpoint(getStackRoxEndpoint())
                .build()
    }

    static NotifierOuterClass.Notifier getJiraIntegrationConfig(String name) {
        return NotifierOuterClass.Notifier.newBuilder()
                .setType("jira")
                .setName(name)
                .setLabelKey("AJIT")
                .setLabelDefault("AJIT")
                .setUiEndpoint(getStackRoxEndpoint())
                .setJira(NotifierOuterClass.Jira.newBuilder()
                        .setUsername("k+automation@stackrox.com")
                        .setPassword("fix-me-ROX-7460-and-this-should-be-secret")
                        .setUrl("https://stack-rox.atlassian.net")
                        .setIssueType("Task")
                )
                .build()
    }

    static NotifierOuterClass.Notifier getTeamsIntegrationConfig(String name) {
        return NotifierOuterClass.Notifier.newBuilder()
                .setType("teams")
                .setName(name)
                .setLabelKey("#teams-test")
                .setLabelDefault("fix-me-ROX-8145-and-this-should-be-secret")
                .setUiEndpoint(getStackRoxEndpoint())
                .build()
    }

    static NotifierOuterClass.Notifier getPagerDutyIntegrationConfig(String name) {
        return NotifierOuterClass.Notifier.newBuilder()
                .setType("pagerduty")
                .setName(name)
                .setUiEndpoint("https://localhost:8000")
                .setPagerduty(NotifierOuterClass.PagerDuty.newBuilder()
                        .setApiKey(PAGERDUTY_API_KEY))
                .build()
    }

    /**
     * This function add a notifier for Splunk.
     *
     * @param legacy Does this integration provide the full URL path or just the base
     * @param name Splunk Integration name
     */
    static NotifierOuterClass.Notifier getSplunkIntegrationConfig(
            boolean legacy,
            String serviceName,
            String name) throws Exception {
        String splunkIntegration = "splunk-Integration"
        String prePackagedToken = "00000000-0000-0000-0000-000000000000"

        return NotifierOuterClass.Notifier.newBuilder()
                .setType("splunk")
                .setName(name)
                .setLabelKey(splunkIntegration)
                .setLabelDefault(splunkIntegration)
                .setUiEndpoint(getStackRoxEndpoint())
                .setSplunk(NotifierOuterClass.Splunk.newBuilder()
                        .setDerivedSourceType(true)
                        .setHttpToken(prePackagedToken)
                        .setInsecure(true)
                        .setHttpEndpoint(String.format(
                                "https://${serviceName}.qa:8088%s",
                                legacy ? "/services/collector/event" : "")))
                .build()
    }

    /**
     * This function adds a notifier for Syslog.
     *
     * @param port Syslog service port number
     * @param name Syslog Integration name
     */
    static NotifierOuterClass.Notifier getSyslogIntegrationConfig(
            String serviceName,
            int port,
            String name) throws Exception {
        String syslogIntegration = "syslog-Integration"

        return NotifierOuterClass.Notifier.newBuilder()
                .setType("syslog")
                .setName(name)
                .setLabelKey(syslogIntegration)
                .setLabelDefault(syslogIntegration)
                .setUiEndpoint(getStackRoxEndpoint())
                .setSyslog(NotifierOuterClass.Syslog.newBuilder()
                        .setTcpConfig(NotifierOuterClass.Syslog.TCPConfig.newBuilder()
                                .setHostname("${serviceName}.qa")
                                .setPort(port)
                                .setSkipTlsVerify(true)
                                .build()
                        )
                        .setMessageFormat(NotifierOuterClass.Syslog.MessageFormat.CEF)
                        .build()
                )
                .build()
    }
}
