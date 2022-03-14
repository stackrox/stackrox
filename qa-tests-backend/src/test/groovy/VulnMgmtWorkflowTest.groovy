import groups.BAT
import groups.RUNTIME
import io.stackrox.proto.storage.Cve
import io.stackrox.proto.storage.VulnRequests
import objects.Deployment
import org.junit.experimental.categories.Category
import services.VulnRequestService
import spock.lang.Unroll

class VulnMgmtWorkflowTest extends BaseSpecification {

    static final private NGINX_1_10_2_IMAGE = "us.gcr.io/stackrox-ci/nginx:1.10.2"

    static final private Deployment CVE_DEPLOYMENT = new Deployment()
            .setName("vulnerable-deploy")
            .setImage(NGINX_1_10_2_IMAGE)
            .addLabel("app", "test")

    static final private String CVE_TO_DEFER = "CVE-2005-2541"
    static final private String CVE_TO_MARK_FP = "CVE-2007-6755"

    def setupSpec() {
        orchestrator.createDeployment(CVE_DEPLOYMENT)
    }

    def cleanupSpec() {
        orchestrator.deleteDeployment(CVE_DEPLOYMENT)
    }

    @Unroll
    @Category([BAT, RUNTIME])
    def "Verify Vulnerability Requests can transition between states - #requestType - approve?(#approve)"() {
        when:
        "A user requests a vuln be deferred or marked as FP"
        VulnRequests.VulnerabilityRequest vulnReq
        if (requestType == "defer") {
            vulnReq = VulnRequestService.deferVuln(
                    CVE_TO_DEFER, "${requestType} me", VulnRequestService.globalScope()).
                    getRequestInfo()
        } else {
            vulnReq = VulnRequestService.markVulnAsFP(
                    CVE_TO_MARK_FP, "${requestType} me", VulnRequestService.globalScope()).
                    getRequestInfo()
        }

        def id = vulnReq.getId()

        assert vulnReq.getStatus() == VulnRequests.RequestStatus.PENDING
        assert !vulnReq.getExpired()

        and:
        "The request is approved or denied"
        if (approve) {
            VulnRequestService.approveRequest(id, "actioned")
        } else {
            VulnRequestService.denyRequest(id, "actioned")
        }

        then:
        "The request should be in the corresponding state with appropriate comments"
        def req = VulnRequestService.getVulnReq(id).getRequestInfo()

        assert req.getStatus() == (approve ? VulnRequests.RequestStatus.APPROVED : VulnRequests.RequestStatus.DENIED)
        assert req.getTargetState() ==
                (requestType == "defer" ? Cve.VulnerabilityState.DEFERRED : Cve.VulnerabilityState.FALSE_POSITIVE)
        if (approve) {
            assert !req.getExpired()
        } else {
            assert req.getExpired()
        }
        assert req.getCves().getIdsCount() == 1
        assert req.getCves().getIds(0) == (requestType == "defer" ? CVE_TO_DEFER: CVE_TO_MARK_FP)
        assert req.getCommentsCount() == 2
        assert req.getComments(0).getMessage() == "${requestType} me" &&
                req.getComments(1).getMessage() == "actioned"

        cleanup:
        if (approve) {
            VulnRequestService.undoReq(id)
        }

        where:
        "Data inputs are:"

        requestType | approve

        "defer" | true
        "defer" | false
        "fp" | true
        "fp" | false
    }
}
