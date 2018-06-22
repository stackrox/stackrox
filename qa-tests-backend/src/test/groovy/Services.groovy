import static com.jayway.restassured.RestAssured.given

class Services {
    static getPolicies(String indexIP) {
        final String ENDPOINT = "https://${indexIP}/v1/policies?query="

        def response =
            given().header("Content-Type", "application/json")
                .get("${ENDPOINT}")
                .then()
                .statusCode(200)
                .extract().body().asString()
        return response
    }

    static getViolations(String indexIP) {
        final String ENDPOINT = "https://${indexIP}/v1/alerts/summary/groups?query=&stale=false"

        def response =
            given().header("Content-Type", "application/json")
                   .get("${ENDPOINT}")
                   .then()
                   .statusCode(200)
                   .extract().body().asString()
        return response
    }

}
