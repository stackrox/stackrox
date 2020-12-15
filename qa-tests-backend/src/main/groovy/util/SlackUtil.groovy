package util

import com.slack.api.Slack
import com.slack.api.webhook.WebhookResponse

class SlackUtil {
    static final String token = Env.mustGetSlackFixableVulnsChannel()

    static sendMessage(String message, String webhook = token) {
        Slack slack = Slack.getInstance()

        WebhookResponse response = slack.send(webhook, "{\"text\":\"${message}\"}")
        if (response.code == 200) {
            println "Sent slack message successfully!"
        } else {
            println "Failed to send Slack message: ${response.body}"
        }
    }
}
