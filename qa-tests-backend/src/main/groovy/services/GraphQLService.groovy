package services

import groovy.json.JsonOutput
import groovy.json.JsonSlurper
import groovy.transform.ToString
import groovy.util.logging.Slf4j
import javax.net.ssl.HostnameVerifier
import javax.net.ssl.SSLContext
import org.apache.http.HttpResponse
import org.apache.http.StatusLine
import org.apache.http.client.methods.HttpPost
import org.apache.http.conn.ssl.NoopHostnameVerifier
import org.apache.http.conn.ssl.SSLConnectionSocketFactory
import org.apache.http.conn.ssl.TrustAllStrategy
import org.apache.http.entity.StringEntity
import org.apache.http.impl.client.CloseableHttpClient
import org.apache.http.impl.client.DefaultHttpRequestRetryHandler
import org.apache.http.impl.client.DefaultServiceUnavailableRetryStrategy
import org.apache.http.impl.client.HttpClients
import org.apache.http.ssl.SSLContextBuilder
import util.Env

@Slf4j
class GraphQLService {
    // Top level service object functionality
    /////////////////////////////////////////

    private final AuthorizedPoster aPoster

    GraphQLService() {
        String username = Env.mustGetUsername()
        String password = Env.mustGetPassword()
        this.aPoster = new AuthorizedPoster(username, password, new Poster(getAddr()))
    }

    GraphQLService(String apiToken) {
        this.aPoster = new AuthorizedPoster(apiToken, new Poster(getAddr()))
    }

    Response Call(String query, Map variables) {
        def sampleMap = ["query": query, "variables": variables]
        Response response = this.aPoster.CallPost(sampleMap)
        if (response.hasNoErrors()) {
            return response
        }

        log.warn("There were errors in the graph QL response: ${response}")

        return response
    }

    // Response value type for GQL requests. Since the return is not tied to any data structure,
    // we return a Map generated from the JSON returned as the response.
    @ToString
    static class Response {
        private final int code
        private final Object value
        private final List<String> errors

        Response() {
            this.code = 0
            this.value = new Object()
            this.errors = new ArrayList<>()
        }

        Response(int code, Object value, List<String> errors) {
            this.code = code
            this.value = value
            this.errors = errors
        }

        int getCode() {
            return this.code
        }

        Object getValue() {
            return this.value
        }

        List<String> getErrors() {
            return this.errors
        }

        Boolean hasNoErrors() {
            return this.code == 200 && (this.errors == null || this.errors.size() == 0)
        }
    }

    // Class that knows how to provide authentication information when making POST requests.
    static private class AuthorizedPoster {
        private final Tuple2<String, String> authHeaderContents
        private final Poster poster

        AuthorizedPoster(String username, String password, Poster poster) {
            this.authHeaderContents = new Tuple2<String, String>(
                "Authorization",
                String.format("Basic %s", Base64
                    .getEncoder()
                    .encodeToString((username + ":" + password).getBytes("UTF-8")))
            )
            this.poster = poster
        }

        AuthorizedPoster(String apiToken, Poster poster) {
            this.authHeaderContents = new Tuple2<String, String>("Authorization", "Bearer " + apiToken)
            this.poster = poster
        }

        Response CallPost(Map content) {
            return this.poster.CallPost([authHeaderContents], content)
        }
    }

    // Class that knows how to create a GQL POST request.
    static private class Poster {
        private final List<Tuple2<String, String>> defaultHeaders
        private final String addr

        private static final Integer MAX_LOG_CHARS = 1024

        Poster(String addr) {
            this.addr = addr
            this.defaultHeaders = [new Tuple2<String, String>("Content-Type", "application/json")]
        }

        Response CallPost(List<Tuple2<String, String>> headers, Map content)  {
            CloseableHttpClient client = buildClient()
            HttpPost httpPost = buildRequest(headers, content)

            HttpResponse response = client.execute(httpPost)
            return parseResponse(response)
        }

        private Response parseResponse(HttpResponse response)  {
            def bsa = new ByteArrayOutputStream()
            response.getEntity().writeTo(bsa)
            def status = response.getStatusLine()
            log.debug "GraphQL response: $status: " + (
                bsa.size() < MAX_LOG_CHARS ? bsa : bsa.toString().take(MAX_LOG_CHARS) + "...")
            if (status.statusCode != 200) {
                return new Response(status.statusCode, null, [bsa.toString()])
            }
            def returnedValue = new JsonSlurper().parseText(bsa.toString())

            return new Response(status.getStatusCode(), returnedValue.data, returnedValue.errors)
        }

        private CloseableHttpClient buildClient()  {
            // Create connection with SSL information.
            SSLContext sslContext = SSLContextBuilder
                .create()
                .loadTrustMaterial(new TrustAllStrategy())
                .build()
            HostnameVerifier allowAllHosts = new NoopHostnameVerifier()
            SSLConnectionSocketFactory connectionFactory = new SSLConnectionSocketFactory(sslContext, allowAllHosts)
            CloseableHttpClient client = HttpClients
                    .custom()
                    .setSSLSocketFactory(connectionFactory)
                    .setRetryHandler(new DefaultHttpRequestRetryHandler(3, true))
                    .setServiceUnavailableRetryStrategy(new DefaultServiceUnavailableRetryStrategy())
                    .build()
            return client
        }

        private HttpPost buildRequest(List<Tuple2<String, String>> headers, Map content)  {
            HttpPost httpPost = new HttpPost(addr)
            for (Tuple2<String, String> header : headers) {
                httpPost.addHeader(header.getFirst(), header.getSecond())
            }
            for (Tuple2<String, String> header : this.defaultHeaders) {
                httpPost.addHeader(header.getFirst(), header.getSecond())
            }
            def jsonContent = new JsonOutput().toJson(content)
            log.debug "GraphQL query: " + (
                jsonContent.length() < MAX_LOG_CHARS ? jsonContent : jsonContent.take(MAX_LOG_CHARS) + "...")
            httpPost.setEntity(new StringEntity(jsonContent))
            return httpPost
        }
    }

    // Helper function that retreives the address.
    static private String getAddr() {
        return String.format("https://%s:%d%s", Env.mustGetHostname(), Env.mustGetPort(), "/api/graphql")
    }
}
