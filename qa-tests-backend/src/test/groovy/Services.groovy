
import com.google.gson.JsonObject
import static com.jayway.restassured.RestAssured.given

class Services {
  def static getPolicies(String indexIP) {
      String ENDPOINT = "https://${indexIP}/v1/policies?query="

      def response =
          given().header("Content-Type", "application/json")
                 .get("${ENDPOINT}")
                 .then()
                 .statusCode(200)
                 .extract().body().asString()
      return response
  }


    def static getViolations(String indexIP) {
        String ENDPOINT = "https://${indexIP}/v1/alerts/summary/groups?query=&stale=false"

        def response =
            given().header("Content-Type", "application/json")
                   .get("${ENDPOINT}")
                   .then()
                   .statusCode(200)
                   .extract().body().asString()
        return response
    }

}
