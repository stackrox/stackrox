package util

import common.Constants
import org.codehaus.groovy.runtime.powerassert.PowerAssertionError
import org.junit.AssumptionViolatedException
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

    static boolean determineRetry(Throwable failure) {
        if (failure instanceof AssumptionViolatedException) {
            return false
        }

        retryAttempt++
        def willRetry = retryAttempt <= MAX_RETRY_ATTEMTPS
        if (willRetry) {
            println "An exception occurred which will cause a retry: " + failure
        }

        if (Env.IN_CI) {
            if (retryAttempt == 1) {
                collectLogsForFailure()
            } else {
                println "Will not collect logs after retry runs."
            }
        }

        if (willRetry) {
            println "Test Failed... Attempting Retry #${retryAttempt}"
        }
        return willRetry
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

    // withDo is like with, but returns a void so can safely be used in tests.
    static void withDo(Object self, Closure closure) {
        self.with(closure)
    }

    private static void collectLogsForFailure() {
        try {
            def logDir = new File(Constants.FAILURE_LOG_DIR + "/k8s-service-logs")
            if (logDir.exists() && logDir.listFiles().size() >= Constants.FAILURE_LOG_LIMIT) {
                println "Log capture limit reached. Not collecting for this failure."
                return
            }
            def collectionDir = logDir.getAbsolutePath() + "/" + UUID.randomUUID()
            println "Will collect logs for this failure under ${collectionDir}"
            println "./scripts/ci/collect-qa-service-logs.sh ${collectionDir}".execute(null, new File("..")).text
        }
        catch (Exception e) {
            println "Could not collect logs: ${e}"
        }
    }
}
