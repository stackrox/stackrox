package services

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.SearchServiceOuterClass.RawQuery
import io.stackrox.proto.storage.Compliance
import io.stackrox.proto.storage.Compliance.ComplianceControlResult
import org.apache.http.client.methods.CloseableHttpResponse
import org.apache.http.client.methods.HttpPost
import org.apache.http.conn.ssl.NoopHostnameVerifier
import org.apache.http.conn.ssl.SSLConnectionSocketFactory
import org.apache.http.conn.ssl.TrustAllStrategy
import org.apache.http.impl.client.CloseableHttpClient
import org.apache.http.impl.client.HttpClients
import org.apache.http.ssl.SSLContextBuilder
import util.Env
import v1.ComplianceServiceGrpc
import v1.ComplianceServiceOuterClass
import v1.ComplianceServiceOuterClass.GetComplianceRunResultsResponse
import v1.ComplianceServiceOuterClass.ComplianceAggregation.Scope
import v1.ComplianceServiceOuterClass.ComplianceStandard
import v1.ComplianceServiceOuterClass.ComplianceStandardMetadata

import javax.net.ssl.HostnameVerifier
import javax.net.ssl.SSLContext
import java.nio.charset.StandardCharsets

class ComplianceService extends BaseService {

    static getComplianceClient() {
        return ComplianceServiceGrpc.newBlockingStub(getChannel())
    }

    static List<ComplianceStandardMetadata> getComplianceStandards() {
        return getComplianceClient().getStandards().standardsList
    }

    static ComplianceStandard getComplianceStandardDetails(String complianceId) {
        try {
            return getComplianceClient()
                    .getStandard(Common.ResourceByID.newBuilder()
                    .setId(complianceId).build()).standard
        } catch (Exception e) {
            println "Could not find Compliance Standard with ID ${complianceId}: ${e.toString()}"
        }
    }

    static List<ComplianceControlResult> getComplianceResults(RawQuery query = RawQuery.newBuilder().build()) {
        // return getComplianceClient().getComplianceControlResults(query).resultsList
        // the results api is not sending any data yet... send mock data instead for testing purposes
        println query
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

    static GetComplianceRunResultsResponse getComplianceRunResult(String standardId, String clusterId) {
        return getComplianceClient().getRunResults(
                ComplianceServiceOuterClass.GetComplianceRunResultsRequest.newBuilder()
                        .setStandardId(standardId)
                        .setClusterId(clusterId)
                        .build()
        )
    }

    static getAggregatedResults(Scope unit, List<Scope> groupBy, RawQuery where = RawQuery.newBuilder().build()) {
        return getComplianceClient().getAggregatedResults(
                ComplianceServiceOuterClass.ComplianceAggregation.Request.newBuilder()
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
        CloseableHttpClient client = HttpClients
                .custom()
                .setSSLSocketFactory(connectionFactory)
                .build()

        HttpPost httpPost = new HttpPost(
                "https://${Env.mustGetHostname()}:${Env.mustGetPort()}" +
                        "/api/compliance/export/csv")
        String username = System.getenv("ROX_USERNAME") ?: ""
        String password = System.getenv("ROX_PASSWORD") ?: ""

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
            e.toString()
            return ""
        }
        return filename
    }

}
