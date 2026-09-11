import static io.restassured.RestAssured.given
import static util.Helpers.evaluateWithRetry

import io.restassured.RestAssured
import io.restassured.config.HttpClientConfig
import io.restassured.config.SSLConfig

import util.Env

import spock.lang.Tag
import spock.lang.Unroll

@Tag("BAT")
class CLIDownloadTest extends BaseSpecification {

    // Magic bytes to identify binary formats
    private static final byte[] LINUX_BINARY_MAGIC   = [0x7f, 0x45, 0x4c, 0x46] as byte[]
    private static final byte[] DARWIN_BINARY_MAGIC  = [(byte) 0xcf, (byte) 0xfa, (byte) 0xed, (byte) 0xfe] as byte[]
    private static final byte[] WINDOWS_BINARY_MAGIC = [0x4d, 0x5a] as byte[]

    @Unroll
    def "GET /api/cli/download/#filename returns a valid #platform binary"() {
        when:
        def response = evaluateWithRetry(3, 5) {
            return given()
                .config(RestAssured.config()
                    .httpClient(HttpClientConfig.httpClientConfig()
                        .setParam("http.connection.timeout", 30_000)
                        .setParam("http.socket.timeout", 120_000))
                    .sslConfig(SSLConfig.sslConfig()
                        .relaxedHTTPSValidation()
                        .allowAllHostnames()))
                .header("Authorization", "Bearer ${allAccessToken}")
                .when()
                .get("https://${Env.mustGetHostname()}:${Env.mustGetPort()}/api/cli/download/${filename}")
        }

        then:
        assert response.statusCode == 200
        byte[] body = response.body.asByteArray()
        assert body.length > 0
        assert body[0..<magic.length] == magic.toList()

        where:
        filename                    | platform          | magic
        "roxctl-linux-amd64"        | "linux/amd64"     | LINUX_BINARY_MAGIC
        "roxctl-linux-arm64"        | "linux/arm64"     | LINUX_BINARY_MAGIC
        "roxctl-linux-ppc64le"      | "linux/ppc64le"   | LINUX_BINARY_MAGIC
        "roxctl-linux-s390x"        | "linux/s390x"     | LINUX_BINARY_MAGIC
        "roxctl-darwin-amd64"       | "darwin/amd64"    | DARWIN_BINARY_MAGIC
        "roxctl-darwin-arm64"       | "darwin/arm64"    | DARWIN_BINARY_MAGIC
        "roxctl-windows-amd64.exe"  | "windows/amd64"   | WINDOWS_BINARY_MAGIC
    }

    def "HEAD /api/cli/download/roxctl-linux-amd64 returns headers with no body"() {
        when:
        def response = given()
            .config(RestAssured.config()
                .sslConfig(SSLConfig.sslConfig()
                    .relaxedHTTPSValidation()
                    .allowAllHostnames()))
            .header("Authorization", "Bearer ${allAccessToken}")
            .when()
            .head("https://${Env.mustGetHostname()}:${Env.mustGetPort()}/api/cli/download/roxctl-linux-amd64")

        then:
        assert response.statusCode == 200
        assert response.header("Content-Length").toLong() > 0
        assert response.body.asByteArray().length == 0
    }

    def "POST /api/cli/download/roxctl-linux-amd64 returns 405 method not allowed"() {
        when:
        def response = given()
            .config(RestAssured.config()
                .sslConfig(SSLConfig.sslConfig()
                    .relaxedHTTPSValidation()
                    .allowAllHostnames()))
            .header("Authorization", "Bearer ${allAccessToken}")
            .when()
            .post("https://${Env.mustGetHostname()}:${Env.mustGetPort()}/api/cli/download/roxctl-linux-amd64")

        then:
        assert response.statusCode == 405
    }

    def "GET /api/cli/download/roxctl-linux-amd64 without authentication returns 401"() {
        when:
        def response = given()
            .config(RestAssured.config()
                .sslConfig(SSLConfig.sslConfig()
                    .relaxedHTTPSValidation()
                    .allowAllHostnames()))
            .when()
            .get("https://${Env.mustGetHostname()}:${Env.mustGetPort()}/api/cli/download/roxctl-linux-amd64")

        then:
        assert response.statusCode == 401
    }
}
