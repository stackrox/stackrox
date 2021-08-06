package util

import com.slack.api.Slack
import com.slack.api.webhook.WebhookResponse

class SlackUtil {
    static final String token = Env.mustGetSlackFixableVulnsChannel()

    static boolean sendMessage(String message, String webhook = token) {
        Slack slack = Slack.getInstance()

        WebhookResponse response = slack.send(webhook, "{\"text\":\"${message}\"}")
        if (response.code == 200) {
            println "Sent slack message successfully!"
            return true
        }

        println "Failed to send Slack message: ${response.body}"
        println "The message was: ${message}"
        return false
    }
}
