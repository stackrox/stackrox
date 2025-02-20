package util

import java.nio.file.Path
import java.nio.file.Paths
import java.text.SimpleDateFormat

import groovy.transform.CompileDynamic
import groovy.transform.CompileStatic
import groovy.util.logging.Slf4j
import io.fabric8.kubernetes.client.KubernetesClientException
import org.codehaus.groovy.runtime.powerassert.PowerAssertionError
import org.javers.core.Javers
import org.javers.core.JaversBuilder
import org.spockframework.runtime.SpockAssertionError

import common.Constants

import org.junit.AssumptionViolatedException

// Helpers defines useful helper methods. Is mixed in to every object in order to be visible everywhere.
@Slf4j
@CompileStatic
class Helpers {
    static <V> V evaluateWithRetry(int retries, int pauseSecs, Closure<V> closure) {
        for (int i = 0; i < retries; i++) {
            try {
                return closure()
            } catch (Exception | PowerAssertionError | SpockAssertionError t) {
                log.debug("Caught exception. Retrying in ${pauseSecs}s (attempt ${i} of ${retries}): " + t)
            }
            sleep pauseSecs * 1000
        }
        return closure()
    }

    static <V> void withRetry(int retries, int pauseSecs, Closure<V> closure) {
        evaluateWithRetry(retries, pauseSecs, closure)
    }

    static <V> V evaluateWithK8sClientRetry(int retries, int pauseSecs, Closure<V> closure) {
        for (int i = 0; i < retries; i++) {
            try {
                return closure()
            } catch (KubernetesClientException t) {
                log.debug("Caught k8 client exception. Retrying in ${pauseSecs}s", t)
            }
            sleep pauseSecs * 1000
        }
        return closure()
    }

    static <V> void withK8sClientRetry(int retries, int pauseSecs, Closure<V> closure) {
        evaluateWithK8sClientRetry(retries, pauseSecs, closure)
    }

    static boolean waitForTrue(int retries, int intervalSeconds, Closure closure) {
        if (!trueWithin(retries, intervalSeconds, closure)) {
            throw new RuntimeException("All ${retries} attempts failed, could not reach desired state")
        }
        return true
    }

    static boolean trueWithin(int retries, int intervalSeconds, Closure closure) {
        Timer t = new Timer(retries, intervalSeconds)
        int attempt = 0
        while (t.IsValid()) {
            attempt++
            if (closure()) {
                return true
            }
            log.debug "Attempt ${attempt} failed, retrying"
        }
        return false
    }

    static boolean containsNoWhitespace(Object ignored, String baseString, String subString) {
        return baseString.replaceAll("\\s", "").contains(subString.replaceAll("\\s", ""))
    }

    static String getStackRoxEndpoint(Object ignored) {
        return "https://" + Env.mustGetHostname() + ":" + Env.mustGetPort()
    }

    @CompileDynamic
    static void collectDebugForFailure(Throwable exception) {
        if (!collectDebug()) {
            return
        }

        if (exception && (exception instanceof AssumptionViolatedException ||
                exception.getMessage()?.contains("org.junit.AssumptionViolatedException"))) {
            log.info("Won't collect logs for: " + exception)
            return
        }

        if (exception && exception.getMessage()?.contains("Ignored via @IgnoreIf")) {
            log.info("Won't collect logs for: " + exception)
            return
        }

        if (exception) {
            log.error("An exception occurred in test", exception)
        }

        try {
            def date = new Date()
            def sdf = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss", Locale.US)

            def debugDir = new File(Env.QA_TEST_DEBUG_LOGS)
            if (debugDir.exists() && debugDir.listFiles().size() >= Constants.FAILURE_DEBUG_LIMIT) {
                log.info "${sdf.format(date)} Debug capture limit reached. Not collecting for this failure."
                return
            }

            def collectionDir = debugDir.getAbsolutePath() + "/" + UUID.randomUUID()

            log.debug "${sdf.format(date)} Will collect various stackrox logs for this failure under ${collectionDir}/"

            shellCmd("./scripts/ci/collect-service-logs.sh stackrox ${collectionDir}/stackrox-k8s-logs")
            shellCmd("./scripts/ci/collect-service-logs.sh kube-system ${collectionDir}/kube-system-k8s-logs")
            shellCmd("./scripts/ci/collect-qa-service-logs.sh ${collectionDir}/qa-k8s-logs")
            shellCmd("./scripts/ci/collect-splunk-logs.sh ${Constants.SPLUNK_TEST_NAMESPACE} "+
                     "${collectionDir}/splunk-logs")
            shellCmd("./scripts/grab-data-from-central.sh ${collectionDir}/central-data")
        }
        catch (Exception e) {
            log.error( "Could not collect logs", e)
        }
    }

    // collectImageScanForDebug(image) - a best effort debug tool to get a complete image scan.
    static void collectImageScanForDebug(String image, String saveName) {
        if (!collectDebug()) {
            return
        }

        log.debug "Will scan ${image} to ${saveName}"

        try {
            Path p = Paths.get(Env.QA_TEST_DEBUG_LOGS)
            Path imageScans = p.resolve('image-scans')
            new File(imageScans.toAbsolutePath().toString()).mkdirs()

            Process proc = "./scripts/ci/roxctl.sh image scan -i ${image} -a".execute([], new File(".."))
            String output = imageScans.resolve(saveName).toAbsolutePath()
            FileWriter sout = new FileWriter(output)
            StringBuilder serr = new StringBuilder()

            proc.waitForProcessOutput(sout, serr)
            proc.waitFor()

            if (proc.exitValue() != 0) {
                log.warn "Failed to scan the image. Exit: ${proc.exitValue()}\nStderr: $serr"
            }

            // closing the FileWriter will ensure internal buffer is flushed to file
            sout.close()
        }
        catch (Exception e) {
            log.error("Could not collect image details", e)
        }
    }


    static void shellCmd(String cmd) {
        shellCmdExitValue(cmd)
    }

    static int shellCmdExitValue(String cmd) {
        StringBuilder sout = new StringBuilder()
        StringBuilder serr = new StringBuilder()
        final List inheritEnv = null
        Process proc = cmd.execute(inheritEnv, new File(".."))
        proc.consumeProcessOutput(sout, serr)
        proc.waitFor()
        log.debug "Ran: ${cmd}\nExit: ${proc.exitValue()}\nStdout: $sout\nStderr: $serr"
        return proc.exitValue()
    }

    private static boolean collectDebug() {
        if ((Env.IN_CI || Env.GATHER_QA_TEST_DEBUG_LOGS) && (Env.QA_TEST_DEBUG_LOGS != "")) {
            return true
        }

        log.warn("Debug collection will be skipped. "+
                 "[CI: ${Env.IN_CI},"+
                 " GATHER_QA_TEST_DEBUG_LOGS: ${Env.GATHER_QA_TEST_DEBUG_LOGS},"+
                 " QA_TEST_DEBUG_LOGS: ${Env.QA_TEST_DEBUG_LOGS}]")

        return false
    }

    static void compareAnnotations(Map<String, String> orchestratorAnnotations,
                                   Map<String, String> stackroxAnnotations) {
        if (stackroxAnnotations == orchestratorAnnotations) {
            return
        }

        Map<String, String> orchestratorTruncated = new HashMap<>(orchestratorAnnotations)
        Map<String, String> stackroxTruncated = new HashMap<>(stackroxAnnotations)
        orchestratorAnnotations.keySet().each {  String name ->
            if (orchestratorTruncated[name].length() > Constants.STACKROX_ANNOTATION_TRUNCATION_LENGTH) {
                // Assert that the stackrox node has an entry for that annotation
                assert stackroxTruncated.get(name) && stackroxTruncated[name].length() > 0

                // Remove the annotation because the logic for truncation tries to maintain words and
                // is more complicated than we'd like to test
                log.info "Removing long annotation value from comparison: " +
                         "key: ${name}, length: ${orchestratorTruncated[name].length()}"
                stackroxTruncated.remove(name)
                orchestratorTruncated.remove(name)
            }
        }

        if (stackroxTruncated == orchestratorTruncated) {
            return
        }

        log.info "There is an annotation difference"
        // Javers helps provide an useful error in the test log
        Javers javers = JaversBuilder.javers().build()
        def diff = javers.compare(stackroxTruncated, orchestratorTruncated)
        assert diff.changes.size() == 0
        assert diff.changes.size() != 0 // should not get here
    }
}
