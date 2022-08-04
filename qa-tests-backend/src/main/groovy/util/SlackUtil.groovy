package util

import com.slack.api.Slack
import com.slack.api.webhook.WebhookResponse
import groovy.util.logging.Slf4j

@Slf4j
class SlackUtil {
    static final String token = Env.mustGetSlackFixableVulnsChannel()

    static boolean sendMessage(String message, String webhook = token) {
        Slack slack = Slack.getInstance()

        WebhookResponse response = slack.send(webhook, "{\"text\":\"${message}\"}")
        if (response.code == 200) {
            log.debug "Sent slack message successfully!"
            return true
        }

        log.warn "Failed to send Slack message: ${response.body}"
        log.debug "The message was: ${message}"
        return false
    }
}
