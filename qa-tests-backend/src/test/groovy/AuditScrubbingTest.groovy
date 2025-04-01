import com.google.protobuf.util.JsonFormat
import groovy.json.JsonSlurper
import io.restassured.RestAssured

import io.stackrox.proto.api.v1.AuthproviderService
import io.stackrox.proto.storage.NotifierOuterClass.Notifier

import services.NotifierService
import util.Env
import util.Helpers

import spock.lang.Shared
import spock.lang.Tag
import spock.lang.Unroll

@Tag("BAT")
@Tag("PZ")
class AuditScrubbingTest extends BaseSpecification {

    static private final String BASIC_AUTH_PROVIDER_ID = "4df1b98c-24ed-4073-a9ad-356aec6bb62d"
    static private final String ENDPOINT = "/v1/authProviders/exchangeToken"

    @Shared
    private Notifier notifier

    def setupSpec() {
        def notifierConfig = NotifierService.getWebhookIntegrationConfig(
                "audit-${getClass().name}-${UUID.randomUUID()}", false, "", true, true)
        notifier = NotifierService.addNotifier(notifierConfig)
        assert notifier
        sleep 3000
    }

    private getAuditEntry(String attemptId) {
        def jsonSlurper = new JsonSlurper()
        def url = new URL("http://localhost:8080")
        return Helpers.evaluateWithRetry(30, 1) {
            def get = url.openConnection()
            def objects = jsonSlurper.parseText(get.getInputStream().getText())
            def entry = objects.find {
                def req = it?.data?.audit?.request
                return req?.endpoint == ENDPOINT && req?.payload?.state?.endsWith(attemptId)
            }
            assert entry
            return entry.data.audit
        }
    }

    @Unroll
    def "Verify that audit log entry (#scenario) for ExchangeToken does not contain sensitive data"() {
        given:
        "Assign a random unique ID to recognize this attempt"
        def attemptId = UUID.randomUUID().toString()

        and:
        "Fix base URL"
        def baseURL = "https://${Env.mustGetHostname()}:${Env.mustGetPort()}"

        when:
        "A POST request is made to the ExchangeToken API"
        RestAssured.given()
                .relaxedHTTPSValidation()
                .body(
                        JsonFormat.printer().print(
                                AuthproviderService.ExchangeTokenRequest.newBuilder()
                                    .setExternalToken("username=${username}&password=${password}")
                                    .setState("${BASIC_AUTH_PROVIDER_ID}:${attemptId}")
                                    .setType("basic")))
                .header("Referer", baseURL)
                .when()
                .post("${baseURL}${ENDPOINT}")
                .then().statusCode(expectedStatusCode)
                .extract().body().asString()

        then:
        "Verify that audit log is found"
        def auditLogEntry = getAuditEntry(attemptId)

        and:
        "Verify that audit log contains a scrubbed externalToken field"
        assert !auditLogEntry["request"]["payload"]["externalToken"]

        and:
        "Verify that audit log string representation does not contain the password"
        assert !auditLogEntry.toString().contains(password)

        where:
        "Data inputs are"
        username              | password              | expectedStatusCode | scenario
        "foo"                 | "bar"                 | 403                | "invalid basic auth password"
        Env.mustGetUsername() | Env.mustGetPassword() | 200                | "valid basic auth credentials"
    }

    def cleanupSpec() {
        if (notifier?.id) {
            NotifierService.deleteNotifier(notifier.id)
        }
    }
}
