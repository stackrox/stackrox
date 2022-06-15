import static com.jayway.restassured.RestAssured.given
import com.jayway.restassured.config.RestAssuredConfig
import com.jayway.restassured.config.SSLConfig
import groups.BAT
import io.stackrox.proto.api.v1.ApiTokenService.GenerateTokenResponse
import io.stackrox.proto.storage.RoleOuterClass
import io.stackrox.proto.storage.RoleOuterClass.Role
import java.util.zip.ZipEntry
import java.util.zip.ZipInputStream
import org.junit.experimental.categories.Category
import services.RoleService
import spock.lang.Shared
import spock.lang.Unroll
import util.Env

@Category(BAT)
class DiagnosticBundleTest extends BaseSpecification {

    @Shared
    private String debugLogsReaderRoleName
    @Shared
    private GenerateTokenResponse adminToken
    @Shared
    private GenerateTokenResponse debugLogsReaderToken
    @Shared
    private GenerateTokenResponse noAccessToken
    @Shared
    private Role noAccessRole

    def setupSpec() {
        adminToken = services.ApiTokenService.generateToken(UUID.randomUUID().toString(), "Admin")
        debugLogsReaderRoleName = UUID.randomUUID()
        RoleService.createRoleWithScopeAndPermissionSet(debugLogsReaderRoleName,
                UNRESTRICTED_SCOPE_ID,
                [
                        "DebugLogs": RoleOuterClass.Access.READ_ACCESS,
                        "Cluster": RoleOuterClass.Access.READ_ACCESS,
                ]
        )
        debugLogsReaderToken = services.ApiTokenService.generateToken(UUID.randomUUID().toString(),
                debugLogsReaderRoleName)
        Map<String, RoleOuterClass.Access> resourceToAccess =
                [
                        "DebugLogs": RoleOuterClass.Access.NO_ACCESS,
                        "Cluster": RoleOuterClass.Access.NO_ACCESS,
                ]

        noAccessRole = RoleService.createRoleWithScopeAndPermissionSet("No Access Test Role - ${RUN_ID}",
            UNRESTRICTED_SCOPE_ID, resourceToAccess)
        noAccessToken = services.ApiTokenService.generateToken(UUID.randomUUID().toString(), noAccessRole.name)
    }

    def cleanupSpec() {
        if (adminToken != null) {
            services.ApiTokenService.revokeToken(adminToken.metadata.id)
        }
        if (debugLogsReaderToken != null) {
            services.ApiTokenService.revokeToken(debugLogsReaderToken.metadata.id)
        }
        if (noAccessToken != null) {
            services.ApiTokenService.revokeToken(noAccessToken.metadata.id)
        }
        if (noAccessRole != null) {
            RoleService.deleteRole(noAccessRole.name)
        }
        RoleService.deleteRole(debugLogsReaderRoleName)
    }

    @Unroll
    def "Test that diagnostic bundle download #desc"() {
        when:
        "Making a request for the diagnostic bundle"

        String token
        switch (authMethod) {
            case "noAccess":
                token = noAccessToken.token
                break
            case "debugLogsRead":
                token = debugLogsReaderToken.token
                break
            case "adminAccess":
                token = adminToken.token
                break
            default:
                token = null
        }
        def headers = new HashMap<String, String>()
        if (token) {
            headers.put("Authorization", "Bearer " + token)
        }

        def response = given()
                .config(RestAssuredConfig.newConfig()
                    .sslConfig(SSLConfig.sslConfig().relaxedHTTPSValidation().allowAllHostnames()))
                .headers(headers)
                .when()
                .get("https://${Env.mustGetHostname()}:${Env.mustGetPort()}/api/extensions/diagnostics")

        then:
        "Check that response is as expected"
        assert response.statusCode == statusCode

        if (statusCode == 200) {
            def foundK8sInfo = false
            def zis = new ZipInputStream(response.body.asInputStream())
            try {
                ZipEntry entry
                while ((entry = zis.nextEntry) != null) {
                    log.info "Found file ${entry.name}"
                    if (entry.name == "kubernetes/remote/stackrox/sensor/deployment-sensor.yaml") {
                        foundK8sInfo = true
                    }
                }
            } finally {
                zis.close()
            }
            assert foundK8sInfo
        }

        cleanup:

        where:
        "Data inputs are"
        statusCode | authMethod      | desc
        401        | ""              | "does not succeed without auth"
        403        | "noAccess"      | "does not succeed with no access token"
        200        | "debugLogsRead" | "succeeds with debug logs reader token"
        200        | "adminAccess"   | "succeeds with admin access token"
    }
}
