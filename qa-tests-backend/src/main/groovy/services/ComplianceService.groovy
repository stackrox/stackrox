package services

import static io.stackrox.proto.api.v1.ComplianceServiceOuterClass.ComplianceAggregationRequest
import static io.stackrox.proto.api.v1.ComplianceServiceOuterClass.ComplianceStandard
import static io.stackrox.proto.api.v1.ComplianceServiceOuterClass.ComplianceStandardMetadata
import static io.stackrox.proto.api.v1.ComplianceServiceOuterClass.GetComplianceRunResultsRequest
import static io.stackrox.proto.api.v1.ComplianceServiceOuterClass.GetComplianceRunResultsResponse
import groovy.util.logging.Slf4j
import java.nio.charset.StandardCharsets
import javax.net.ssl.HostnameVerifier
import javax.net.ssl.SSLContext

import groovy.transform.CompileStatic
import org.apache.http.client.methods.CloseableHttpResponse
import org.apache.http.client.methods.HttpPost
import org.apache.http.conn.ssl.NoopHostnameVerifier
import org.apache.http.conn.ssl.SSLConnectionSocketFactory
import org.apache.http.conn.ssl.TrustAllStrategy
import org.apache.http.impl.client.CloseableHttpClient
import org.apache.http.impl.client.DefaultHttpRequestRetryHandler
import org.apache.http.impl.client.DefaultServiceUnavailableRetryStrategy
import org.apache.http.impl.client.HttpClients
import org.apache.http.ssl.SSLContextBuilder

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.ComplianceServiceGrpc
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.storage.Compliance
import io.stackrox.proto.storage.Compliance.ComplianceAggregation.Scope
import io.stackrox.proto.storage.Compliance.ComplianceControlResult

import util.Env

@Slf4j
@CompileStatic
class ComplianceService extends BaseService {

    static ComplianceServiceGrpc.ComplianceServiceBlockingStub getComplianceClient() {
        return ComplianceServiceGrpc.newBlockingStub(getChannel())
    }

    static List<ComplianceStandardMetadata> getComplianceStandards() {
        return getComplianceClient().getStandards(null).standardsList
    }

    static ComplianceStandard getComplianceStandardDetails(String complianceId) {
        try {
            return getComplianceClient()
                    .getStandard(Common.ResourceByID.newBuilder()
                    .setId(complianceId).build()).standard
        } catch (Exception e) {
            log.error("Could not find Compliance Standard with ID ${complianceId}", e)
        }
        return null
    }

    static List<ComplianceControlResult> getComplianceResults(RawQuery query = RawQuery.newBuilder().build()) {
        // return getComplianceClient().getComplianceControlResults(query).resultsList
        // the results api is not sending any data yet... send mock data instead for testing purposes
        log.debug(query.getQuery())
        return [
                ComplianceControlResult.newBuilder()
                        .setControlId("fake-1.1.1")
                        .setValue(Compliance.ComplianceResultValue.newBuilder()
                                .addEvidence(
                                        Compliance.ComplianceResultValue.Evidence.newBuilder()
                                                .setState(Compliance.ComplianceState.COMPLIANCE_STATE_SUCCESS)
                                                .setMessage("Your deployments are frobotzed")
                                                .build()
                                ).build()
                        ).build(),
                ComplianceControlResult.newBuilder()
                        .setControlId("fake-1.1.2")
                        .setValue(Compliance.ComplianceResultValue.newBuilder()
                                .addEvidence(
                                        Compliance.ComplianceResultValue.Evidence.newBuilder()
                                                .setState(Compliance.ComplianceState.COMPLIANCE_STATE_FAILURE)
                                                .setMessage("Your nodes are not frobotzed")
                                                .build()
                                ).build()
                        ).build(),
        ]
    }

    static Compliance.ComplianceRunResults triggerComplianceRunAndWaitForResult(String standardId, String clusterId) {
        def runId = ComplianceManagementService.triggerComplianceRunsAndWait(standardId, clusterId).get(standardId)
        def result = getComplianceRunResult(standardId, clusterId, runId).results
        assert result.runMetadata.runId == runId
        return result
    }

    static GetComplianceRunResultsResponse getComplianceRunResult(String standardId, String clusterId,
                                                                  String runId = null) {
        return getComplianceClient().getRunResults(
                GetComplianceRunResultsRequest.newBuilder()
                        .setStandardId(standardId)
                        .setClusterId(clusterId)
                        .setRunId(runId ?: "")
                        .build()
        )
    }

    static getAggregatedResults(Scope unit, List<Scope> groupBy, RawQuery where = RawQuery.newBuilder().build()) {
        return getComplianceClient().getAggregatedResults(
                ComplianceAggregationRequest.newBuilder()
                        .addAllGroupBy(groupBy)
                        .setUnit(unit)
                        .setWhere(where)
                        .build()
        ).resultsList
    }

    static exportComplianceCsv() {
        SSLContext sslContext = SSLContextBuilder
                .create()
                .loadTrustMaterial(new TrustAllStrategy())
                .build()
        HostnameVerifier allowAllHosts = new NoopHostnameVerifier()
        SSLConnectionSocketFactory connectionFactory = new SSLConnectionSocketFactory(sslContext, allowAllHosts)
        int maxRetryCount = 3
        int retryIntervalMs = 5000
        CloseableHttpClient client = HttpClients
                .custom()
                .setRetryHandler(
                        new DefaultHttpRequestRetryHandler(maxRetryCount, true))
                .setServiceUnavailableRetryStrategy(
                        new DefaultServiceUnavailableRetryStrategy(maxRetryCount, retryIntervalMs))
                .setSSLSocketFactory(connectionFactory)
                .build()

        HttpPost httpPost = new HttpPost(
                "https://${Env.mustGetHostname()}:${Env.mustGetPort()}" +
                        "/api/compliance/export/csv")
        String username = Env.mustGetUsername()
        String password = Env.mustGetPassword()

        httpPost.addHeader(
                "Authorization",
                "Basic " +
                        Base64.getEncoder().encodeToString((username + ":" + password).getBytes("UTF-8")))

        def exportPath = System.getProperty("user.dir") + "/export"
        File exportDir = new File(exportPath)
        if (!exportDir.exists()) {
            exportDir.mkdirs()
        }
        def filename = exportPath + "/export.csv"

        try {
            CloseableHttpResponse response = client.execute(httpPost)
            def conn = new InputStreamReader(response.getEntity().getContent(), StandardCharsets.UTF_8)
            new File(filename).withWriter("utf-8") { out ->
                conn.with { inp ->
                    out << inp
                    inp.close()
                }
            }
        } catch (Exception e) {
            log.error(e.toString(), e)
            return ""
        }
        return filename
    }

}
