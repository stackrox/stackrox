package util

import static com.jayway.restassured.RestAssured.given
import com.google.gson.Gson
import com.jayway.restassured.response.Response

class SplunkUtil {

    static Map<String,String> getInfoFromSplunk(String httpsEndPoint) {
        Gson gson = new Gson()
        Response response = getResponseWithTimeout(httpsEndPoint, 300)
        if (response != null ) {
            RawParser parser = gson.fromJson(response.jsonPath().get("result")["_raw"], RawParser)
            RawParser.Policy policyParser = gson.fromJson(parser.policy, RawParser.Policy)
            RawParser.Deployment deploymentParser = gson.fromJson(parser.deployment, RawParser.Deployment)
            def map = new HashMap<String, String>()
            map["preview"] = response.jsonPath().get("preview")
            map["offset"] = response.jsonPath().get("offset")
            map["policy"] = trimQuotes(policyParser.name.toString())
            map["namespace"] = trimQuotes(deploymentParser.namespace.toString())
            map["name"] = trimQuotes(deploymentParser.name.toString())
            map["clusterName"] = trimQuotes(deploymentParser.clusterName.toString())
            map["type"] = trimQuotes(deploymentParser.type.toString())
            map["source"] = response.jsonPath().get("result")["source"]
            map["sourcetype"] = response.jsonPath().get("result")["sourcetype"]
            return map
        }
        return [:]
    }

    static String trimQuotes(String str) {
        return str[1..(str.length()-2)]
    }

    static Map<String,String> waitForSplunkAlerts(String httpsLoadBalancer, int timeoutSeconds) {
        int intervalSeconds = 1
        int iterations = timeoutSeconds / intervalSeconds
        Map resultMap
        Timer t = new Timer(iterations, intervalSeconds)
        while (resultMap == null && t.IsValid()) {
            resultMap = getInfoFromSplunk(httpsLoadBalancer)
        }
        println("Received response from splunk after  ${iterations}  and  ${t.SecondsSince()} seconds")
        return resultMap
    }

    static Response getResponseWithTimeout(String deploymentIP, int timeout) {
        Response response
        int intervalSeconds = 1
        int iterations = timeout / intervalSeconds
        Timer t = new Timer(iterations, intervalSeconds)
        while (response?.jsonPath()?.get("result") == null && t.IsValid()) {
            try {
                response = given().auth().basic("admin", "changeme")
                        .param("search", "search")
                        .param("host", "splunk-collector.qa:8088")
                        .param("output_mode", "json")
                        .get("https://${deploymentIP}:8089/services/search/jobs/export")
                println("Querying loadbalancer ${deploymentIP}")
            }
            catch (UnknownHostException e) {
                println("catching unknownhost exception for KOPS to refresh DNS" + e)
            }
        }
        println("Printing response from ${deploymentIP} " + response?.prettyPrint())
        return response
    }
}
