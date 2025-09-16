package util

import static io.restassured.RestAssured.given
import static util.Helpers.withRetry

import com.google.gson.Gson
import com.google.gson.GsonBuilder
import groovy.transform.TupleConstructor
import groovy.util.logging.Slf4j
import io.fabric8.kubernetes.client.LocalPortForward
import io.restassured.response.Response
import orchestratormanager.Kubernetes

import objects.Deployment
import objects.Service
import objects.SplunkAlert
import objects.SplunkAlertRaw
import objects.SplunkAlerts
import objects.SplunkHECEntry
import objects.SplunkHECTokens
import objects.SplunkHealthEntry
import objects.SplunkHealthResults
import objects.SplunkSearch

import org.junit.AssumptionViolatedException

@Slf4j
class SplunkUtil {
    public static final String SPLUNK_ADMIN_PASSWORD = "helloworld"
    private static final Gson GSON = new GsonBuilder().create()
    private static final Map<String, String> ENV_VARIABLES = ["SPLUNK_START_ARGS" : "--accept-license",
        "SPLUNK_USER": "root",
        "SPLUNK_PASSWORD"   : SPLUNK_ADMIN_PASSWORD,
        // This is required to get splunk 8.1.2 to start in an OpenShift crio environment
        // https://docs.splunk.com/Documentation/Splunk/7.0.3/Troubleshooting/FSLockingIssues#
        // Splunk_Enterprise_does_not_start_due_to_unusable_filesystem
        // See https://github.com/splunk/splunk-ansible/issues/349
        "SPLUNK_LAUNCH_CONF": "OPTIMISTIC_ABOUT_FILE_LOCKING=1",]

    static List<SplunkAlert> getSplunkAlerts(int port, String searchId) {
        Response response = getSearchResults(port, searchId)
        SplunkAlerts alerts = GSON.fromJson(response.asString(), SplunkAlerts)

        def returnAlerts = []
        for (SplunkAlertRaw raw : alerts.results) {
            returnAlerts.add(GSON.fromJson(raw._raw, SplunkAlert))
        }
        return returnAlerts
    }

    static List<String> getSplunkSyslogs(int port, String searchId) {
        Response response = getSearchResults(port, searchId)
        // Not actually SplunkAlerts, just a list of response strings.
        SplunkAlerts responseItems = GSON.fromJson(response.asString(), SplunkAlerts)
        def syslogStrings = []
        for (SplunkAlertRaw raw : responseItems.results) {
            syslogStrings.add(raw._raw)
        }
        return syslogStrings
    }

    static List<SplunkAlert> waitForSplunkAlerts(int port, String search = "search") {
        log.info("Waiting for data to arrive in Splunk for search query: " + search)
        List results = []
        withRetry(40, 15) {
            def searchId
            searchId = createSearch(port, search)
            results = getSplunkAlerts(port, searchId)
            assert results.size() > 0
        }
        log.info("Data arrived in Splunk!")
        return results
    }

    static List<String> waitForSplunkSyslog(int port, int timeoutSeconds) {
        int intervalSeconds = 3
        int iterations = timeoutSeconds / intervalSeconds
        List results = []
        Exception exception = null
        Timer t = new Timer(iterations, intervalSeconds)
        while (results.size() == 0 && t.IsValid()) {
            def searchId = null
            try {
                searchId = createSearch(port, "search source=\"yeet syslogs\"")
                exception = null
            } catch (Exception e) {
                exception = e
            }
            results = getSplunkSyslogs(port, searchId)
        }

        if (exception) {
            throw exception
        }
        return results
    }

    static Response getSearchResults(int port, String searchId) {
        Response response = null
        withRetry(20, 3) {
            response = given().auth()
                    .basic("admin", SPLUNK_ADMIN_PASSWORD)
                    .param("output_mode", "json")
                    .get("https://127.0.0.1:${port}/services/search/jobs/${searchId}/events")
        }
        return response
    }

    static String createSearch(int port, String search = "search") {
        Response response = null
        withRetry(6, 15) {
            response = given()
                    .auth()
                    .basic("admin", SPLUNK_ADMIN_PASSWORD)
                    .formParam("search", search)
                    .formParam("output_mode", "json")
                    .post("https://127.0.0.1:${port}/services/search/jobs")
        }

        log.debug response?.asString()
        def searchId = GSON.fromJson(response?.asString(), SplunkSearch)?.sid
        if (searchId == null) {
            log.debug "Failed to generate new search. SearchId is null..."
            throw new AssumptionViolatedException("Failed to create new Splunk search!")
        } else {
            log.debug "New Search created: ${searchId}"
            return searchId
        }
    }

    static String createHECToken(int port, String tokenName = "stackrox") {
        Response response = null
        withRetry(20, 15) {
            log.info("Attempting to create new HEC ingest token")
            response =  given().auth()
                .basic("admin", SPLUNK_ADMIN_PASSWORD)
                .formParam("name", tokenName)
                .formParam("output_mode", "json")
                .post("https://127.0.0.1:${port}/servicesNS/nobody/search/data/inputs/http")
        }
        def hec = unmarshalHEC(response?.asString())
        assert hec.size() == 36 // Splunk has a unified token format based on a hash-like
        return hec
    }

    static String unmarshalHEC(String response) {
        SplunkHECTokens tokens = GSON.fromJson(response, SplunkHECTokens)
        // This structure only ever contains one entry in the array, so we return after parsing the first one
        for (SplunkHECEntry entry: tokens.entry) {
            return entry.content.token
        }
    }

    static SplunkDeployment createSplunk(Kubernetes orchestrator, String namespace) {
        def uid = UUID.randomUUID()
        def deploymentName = "splunk-${uid}"
        Deployment deployment
        Service collectorSvc
        Service syslogSvc
        LocalPortForward splunkPortForward
        try {
            deployment =
                    new Deployment()
                            .setNamespace(namespace)
                            .setName(deploymentName)
                            .setImage("quay.io/rhacs-eng/qa:splunk-test-repo-9-0-5")
                            .addPort(8000)
                            .addPort(8088)
                            .addPort(8089)
                            .addPort(514)
                            .setEnv(ENV_VARIABLES)
                            .addLabel("app", deploymentName)
            orchestrator.createDeployment(deployment)

            collectorSvc = new Service("splunk-collector-${uid}", namespace)
                    .addLabel("app", deploymentName)
                    .addPort(8088, "TCP")
                    .setType(Service.Type.CLUSTERIP)
            orchestrator.createService(collectorSvc)

            syslogSvc = new Service("splunk-syslog-${uid}", namespace)
                    .addLabel("app", deploymentName)
                    .addPort(514, "TCP")
                    .setType(Service.Type.CLUSTERIP)
            orchestrator.createService(syslogSvc)

            splunkPortForward = orchestrator.createPortForward(8089, deployment)
        } catch (Exception e) {
            log.info("Something bad happened, will run cleanup before failing", e)
            if (syslogSvc) {
                orchestrator.deleteService(syslogSvc.name, syslogSvc.namespace)
            }
            if (collectorSvc) {
                orchestrator.deleteService(collectorSvc.name, collectorSvc.namespace)
            }
            if (deployment) {
                orchestrator.deleteDeployment(deployment)
            }
            throw e
        }
        return new SplunkDeployment(uid, collectorSvc, splunkPortForward, syslogSvc, deployment)
    }

    static void waitForSplunkBoot(int port) {
        log.info("Waiting for Splunk to boot...")
        Response response = null
        withRetry(30, 10) {
            response = given().auth()
                    .basic("admin", SPLUNK_ADMIN_PASSWORD)
                    .param("output_mode", "json")
                    .get("https://127.0.0.1:${port}/services/server/health/splunkd/details")
            assert response != null
            log.debug("Splunkd status: \n" + response?.asString())
            SplunkHealthResults results = GSON.fromJson(response?.asString(), SplunkHealthResults)
            for (SplunkHealthEntry entry: results.entry) {
                // Splunk status is modeled as green/yellow/red.
                // We're only interested in the Index Processor and Search Scheduler,
                // as overall health can be degraded because of IOWait or disk storage.
                assert entry.content.features.searchScheduler.health == "green"
                // The IndexProcessor can be yellow because of free disk space. That's okay for the short-lived test.
                assert (entry.content.features.indexProcessor.health == "green" ||
                        entry.content.features.indexProcessor.health == "yellow")
            }
            log.info("Splunk has completed booting")
        }
    }

    static void tearDownSplunk(Kubernetes orchestrator, SplunkDeployment splunkDeployment) {
        def imagePullSecrets = splunkDeployment.deployment.getImagePullSecret()
        for (String secret : imagePullSecrets) {
            orchestrator.deleteSecret(secret, splunkDeployment.deployment.namespace)
        }
        orchestrator.deleteService(splunkDeployment.syslogSvc.name, splunkDeployment.syslogSvc.namespace)
        orchestrator.deleteService(splunkDeployment.collectorSvc.name, splunkDeployment.collectorSvc.namespace)
        orchestrator.deleteDeployment(splunkDeployment.deployment)
    }

    static void postToSplunk(int port, String path, Map<String, String> parameters) {
        withRetry(20, 30) {
            given().auth().basic("admin", SPLUNK_ADMIN_PASSWORD)
                    .relaxedHTTPSValidation()
                    .params(parameters)
                    .post("https://localhost:${port}${path}")
        }
    }

    @TupleConstructor
    static class SplunkDeployment {
        UUID uid
        Service collectorSvc
        LocalPortForward splunkPortForward
        Service syslogSvc
        Deployment deployment
    }
}
