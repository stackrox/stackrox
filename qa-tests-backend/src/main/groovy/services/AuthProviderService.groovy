package services

import static io.restassured.RestAssured.given

import groovy.transform.CompileStatic
import io.restassured.config.RestAssuredConfig
import groovy.util.logging.Slf4j

import util.Helpers
import util.Keys

import javax.net.ssl.SSLContext
import java.security.SecureRandom

import io.restassured.config.SSLConfig
import org.apache.http.conn.ssl.SSLSocketFactory
import util.Env

import io.stackrox.proto.api.v1.AuthProviderServiceGrpc
import io.stackrox.proto.api.v1.AuthproviderService
import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.storage.AuthProviderOuterClass

@Slf4j
@CompileStatic
class AuthProviderService extends BaseService {
    static AuthProviderServiceGrpc.AuthProviderServiceBlockingStub getAuthProviderService() {
        return AuthProviderServiceGrpc.newBlockingStub(getChannel())
    }

    static AuthproviderService.GetAuthProvidersResponse getAuthProviders() {
        AuthproviderService.GetAuthProvidersResponse ret = null
        Helpers.withRetry(3, 2) {
            ret = getAuthProviderService().getAuthProviders(
                    AuthproviderService.GetAuthProvidersRequest.newBuilder().build()
            )
        }
        return ret
    }

    static private AuthProviderOuterClass.AuthProvider getAuthProvider(String id) {
        AuthProviderOuterClass.AuthProvider ret = null
        Helpers.withRetry(3, 2) {
            ret = getAuthProviderService().getAuthProvider(
                    AuthproviderService.GetAuthProviderRequest.newBuilder().setId(id).build()
            )
        }
        return ret
    }

    static createAuthProvider(String name, String type, Map<String, String> config, String defaultRole = null) {
        try {
            def authProviderId = getAuthProviderService().postAuthProvider(
                    AuthproviderService.PostAuthProviderRequest.newBuilder().setProvider(
                            AuthProviderOuterClass.AuthProvider.newBuilder()
                                    .setName(name)
                                    .setType(type)
                                    .putAllConfig(config)
                                    .setEnabled(true)
                    ).build()
            ).id

            return authProviderId
        } catch (Exception e) {
            log.error("Failed to create auth provider", e)
        }
    }

    static deleteAuthProvider(String id) {
        getAuthProviderService().deleteAuthProvider(Common.DeleteByIDWithForce.newBuilder().setId(id).build())
    }

    static getAuthProviderLoginToken(String id) {
        String loginUrl = getAuthProvider(id).loginUrl

        def sslContext = SSLContext.getInstance("TLS")
        sslContext.init(Keys.keyManagerFactory().keyManagers, Keys.trustManagerFactory().trustManagers,
                new SecureRandom())

        def socketFactory = new SSLSocketFactory(sslContext, SSLSocketFactory.ALLOW_ALL_HOSTNAME_VERIFIER)

        def location = loginUrl
        // There are two redirects: first from the generic URL to the auth provider's URL, and then from the auth
        // provider's URL to the token response URL.
        for (int i = 0; i < 2; i++) {
            def response =
                    given().header("Content-Type", "application/json")
                            .config(RestAssuredConfig.newConfig().sslConfig(
                            SSLConfig.sslConfig().with().sslSocketFactory(socketFactory)
                                    .and().allowAllHostnames()))
                            .when()
                            .redirects().follow(false)
                            .get("https://${Env.mustGetHostname()}:${Env.mustGetPort()}${location}")
            location = response.getHeader("Location")
        }
        def fullURL = new URL("https://${Env.mustGetHostname()}:${Env.mustGetPort()}${location}")
        def token = ""
        fullURL.ref.split("&").each {
            def values = it.split("=")
            if (values[0] == "token") {
                token = values[1]
            }
        }
        assert token != "" : "Could not determine token for cert"
        return token
    }
}
