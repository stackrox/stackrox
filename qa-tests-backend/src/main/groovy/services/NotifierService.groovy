package services

import common.Constants
import io.stackrox.proto.api.v1.NotifierServiceGrpc
import io.stackrox.proto.api.v1.NotifierServiceOuterClass
import io.stackrox.proto.storage.Common
import io.stackrox.proto.storage.NotifierOuterClass
import util.Env

class NotifierService extends BaseService {
    private static final PAGERDUTY_API_KEY = "9e2d142a2946419c9192a0b224dd811b"

    static getNotifierClient() {
        return NotifierServiceGrpc.newBlockingStub(getChannel())
    }

    static addNotifier(NotifierOuterClass.Notifier notifier) {
        try {
            return getNotifierClient().postNotifier(notifier)
        } catch (Exception e) {
            println "Failed to add notifier..."
            throw e
        }
    }

    static testNotifier(NotifierOuterClass.Notifier notifier) {
        try {
            getNotifierClient().testNotifier(notifier)
            return true
        } catch (Exception e) {
            println e.toString()
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
            println e.toString()
        }
    }

    static NotifierOuterClass.Notifier getEmailIntegrationConfig(
            String name,
            disableTLS = false,
            startTLS = NotifierOuterClass.Email.AuthMethod.DISABLED,
            Integer port = null) {
        NotifierOuterClass.Notifier.Builder builder =
                NotifierOuterClass.Notifier.newBuilder()
                        .setEmail(NotifierOuterClass.Email.newBuilder())
        builder
                .setType("email")
                .setName(name)
                .setLabelKey("mailgun")
                .setLabelDefault("stackrox.qa@gmail.com")
                .setEnabled(true)
                .setUiEndpoint(getStackRoxEndpoint())
                .setEmail(builder.getEmailBuilder()
                        .setUsername("automation@mailgun.rox.systems")
                        .setPassword(Env.mustGet("MAILGUN_PASSWORD"))
                        .setSender(Constants.EMAIL_NOTIFER_SENDER)
                        .setFrom(Constants.EMAIL_NOTIFER_FROM)
                        .setDisableTLS(disableTLS)
                        .setStartTLSAuthMethod(startTLS)
                )
        port == null ?
                builder.getEmailBuilder().setServer("smtp.mailgun.org") :
                builder.getEmailBuilder().setServer("smtp.mailgun.org:" + port)
        return builder.build()
    }

    static NotifierOuterClass.Notifier getWebhookIntegrationConfig(
            String name,
            Boolean enableTLS,
            String caCert,
            Boolean skipTLSVerification,
            Boolean auditLoggingEnabled)  {
        NotifierOuterClass.GenericOrBuilder genericBuilder =  NotifierOuterClass.Generic.newBuilder()
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

    static NotifierOuterClass.Notifier getSlackIntegrationConfig(String name) {
        return NotifierOuterClass.Notifier.newBuilder()
                .setType("slack")
                .setName(name)
                .setLabelKey("#slack-test")
                .setLabelDefault("https://hooks.slack.com/services/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")
                .setEnabled(true)
                .setUiEndpoint(getStackRoxEndpoint())
                .build()
    }

    static NotifierOuterClass.Notifier getJiraIntegrationConfig(String name) {
        return NotifierOuterClass.Notifier.newBuilder()
                .setType("jira")
                .setName(name)
                .setLabelKey("AJIT")
                .setLabelDefault("AJIT")
                .setEnabled(true)
                .setUiEndpoint(getStackRoxEndpoint())
                .setJira(NotifierOuterClass.Jira.newBuilder()
                        .setUsername("k+automation@stackrox.com")
                        .setPassword("xvOOtL7nCOANMbD7ed0522B5")
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
                .setLabelDefault(
                        "https://outlook.office.com/webhook/8a021ef7-9845-449a-a0c0-7bf85eab3955@" +
                                "6aec22ae-2b26-45bd-b17f-d60e89828e89/IncomingWebhook/9bb3b3574ea2" +
                                "4655b6482116848bf175/6de97827-1fef-4f8c-a8ab-edac7629df89")
                .setEnabled(true)
                .setUiEndpoint(getStackRoxEndpoint())
                .build()
    }

    static NotifierOuterClass.Notifier getPagerDutyIntegrationConfig(String name) {
        return NotifierOuterClass.Notifier.newBuilder()
                .setType("pagerduty")
                .setName(name)
                .setEnabled(true)
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
            String name)  throws Exception {
        String splunkIntegration = "splunk-Integration"
        String prePackagedToken = "00000000-0000-0000-0000-000000000000"

        return NotifierOuterClass.Notifier.newBuilder()
                .setType("splunk")
                .setName(name)
                .setLabelKey(splunkIntegration)
                .setLabelDefault(splunkIntegration)
                .setEnabled(true)
                .setUiEndpoint(getStackRoxEndpoint())
                .setSplunk(NotifierOuterClass.Splunk.newBuilder()
                        .setHttpToken(prePackagedToken)
                        .setInsecure(true)
                        .setHttpEndpoint(String.format(
                                "https://${serviceName}.qa:8088%s",
                                legacy ? "/services/collector/event" : "")))
                .build()
    }
}
