package util

import static com.jayway.restassured.RestAssured.given

import com.google.gson.GsonBuilder
import objects.SplunkAlert
import objects.SplunkAlertRaw
import objects.SplunkAlerts
import objects.SplunkSearch

import com.google.gson.Gson
import com.jayway.restassured.response.Response

class SplunkUtil {
    static final private Gson GSON = new GsonBuilder().create()

    static List<SplunkAlert> getSplunkAlerts(int port, String searchId) {
        Response response = getSearchResults(port, searchId)
        SplunkAlerts alerts = GSON.fromJson(response.asString(), SplunkAlerts)

        def returnAlerts = []
        for (SplunkAlertRaw raw : alerts.results) {
            returnAlerts.add(GSON.fromJson(raw._raw, SplunkAlert))
        }
        return returnAlerts
    }

    static List<SplunkAlert> waitForSplunkAlerts(int port, int timeoutSeconds) {
        int intervalSeconds = 3
        int iterations = timeoutSeconds / intervalSeconds
        List results = []
        Timer t = new Timer(iterations, intervalSeconds)
        while (results.size() == 0 && t.IsValid()) {
            def searchId = createSearch(port)
            results = getSplunkAlerts(port, searchId)
        }
        return results
    }

    static Response getSearchResults(int port, String searchId) {
        Response response
        try {
            response = given().auth().basic("admin", "changeme")
                    .param("output_mode", "json")
                    .get("https://127.0.0.1:${port}/services/search/jobs/${searchId}/events")
        }
        catch (Exception e) {
            println("catching unknownhost exception for KOPS and other intermittent connection issues" + e)
        }
        println "Printing response from https://127.0.0.1:${port} " + response?.prettyPrint()
        return response
    }

    static String createSearch(int port) {
        Response response
        try {
            withRetry(20, 3) {
                response = given().auth().basic("admin", "changeme")
                        .formParam("search", "search")
                        .param("output_mode", "json")
                        .post("https://127.0.0.1:${port}/services/search/jobs")
            }
        }
        catch (Exception e) {
            println("catching unknownhost exception for KOPS and other intermittent connection issues" + e)
        }
        println "New Search created: ${GSON.fromJson(response.asString(), SplunkSearch).sid}"
        return GSON.fromJson(response.asString(), SplunkSearch).sid
    }
}
