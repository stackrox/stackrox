package util

import org.codehaus.groovy.runtime.powerassert.PowerAssertionError
import org.spockframework.runtime.SpockAssertionError

// Helpers defines useful helper methods. Is mixed in to every object in order to be visible everywhere.
class Helpers {
    private static final int MAX_RETRY_ATTEMTPS = 2
    private static int retryAttempt = 0

    static <V> V evaluateWithRetry(Object ignored, int retries, int pauseSecs, Closure<V> closure) {
        for (int i = 0; i < retries; i++) {
            try {
                return closure()
            } catch (Exception | PowerAssertionError | SpockAssertionError t) {
                println "Caught exception: ${t}. Retrying in ${pauseSecs}s"
            }
            sleep pauseSecs * 1000
        }
        return closure()
    }

    static void withRetry(Object ignored, int retries, int pauseSecs, Closure<?> closure) {
        evaluateWithRetry(retries, pauseSecs, closure)
    }

    static boolean determineRetry() {
        retryAttempt++
        if (retryAttempt <= MAX_RETRY_ATTEMTPS) {
            println "Test Failed... Attempting Retry #${retryAttempt}"
            return true
        }
        return false
    }

    static void resetRetryAttempts() {
        retryAttempt = 0
    }

    static boolean containsNoWhitespace(Object ignored, String baseString, String subString) {
        return baseString.replaceAll("\\s", "").contains(subString.replaceAll("\\s", ""))
    }

    static String getStackRoxEndpoint(Object ignored) {
        return "https://" + Env.mustGetHostname() + ":" + Env.mustGetPort()
    }
}
