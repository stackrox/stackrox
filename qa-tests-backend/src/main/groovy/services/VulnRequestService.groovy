package services

import groovy.transform.CompileStatic

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.VulnMgmtReqService
import io.stackrox.proto.api.v1.VulnerabilityRequestServiceGrpc
import io.stackrox.proto.storage.VulnRequests.VulnerabilityRequest

@CompileStatic
class VulnRequestService extends BaseService {
    static VulnerabilityRequestServiceGrpc.VulnerabilityRequestServiceBlockingStub getVulnRequestClient() {
        return VulnerabilityRequestServiceGrpc.newBlockingStub(getChannel())
    }

    static getVulnReq(String reqID) {
        def id = Common.ResourceByID.newBuilder()
                .setId(reqID)
                .build()
        return getVulnRequestClient().getVulnerabilityRequest(id)
    }

    static deferVuln(String cve, String comment, VulnerabilityRequest.Scope scope) {
        def req = VulnMgmtReqService.DeferVulnRequest.newBuilder().
                setCve(cve).
                setScope(scope).
                setComment(comment).
                setExpiresWhenFixed(true).
                build()
        return getVulnRequestClient().deferVulnerability(req)
    }

    static markVulnAsFP(String cve, String comment, VulnerabilityRequest.Scope scope) {
        def req = VulnMgmtReqService.FalsePositiveVulnRequest.newBuilder().
                setCve(cve).
                setScope(scope).
                setComment(comment).
                build()
        return getVulnRequestClient().falsePositiveVulnerability(req)
    }

    static approveRequest(String reqID, String comment) {
        def req = VulnMgmtReqService.ApproveVulnRequest.newBuilder().
                setId(reqID).
                setComment(comment).
                build()
        return getVulnRequestClient().approveVulnerabilityRequest(req)
    }

    static denyRequest(String reqID, String comment) {
        def req = VulnMgmtReqService.DenyVulnRequest.newBuilder().
                setId(reqID).
                setComment(comment).
                build()
        return getVulnRequestClient().denyVulnerabilityRequest(req)
    }

    static cancelReq(String reqID) {
        def id = Common.ResourceByID.newBuilder()
                .setId(reqID)
                .build()
        return getVulnRequestClient().deleteVulnerabilityRequest(id)
    }

    static undoReq(String reqID) {
        def id = Common.ResourceByID.newBuilder()
                .setId(reqID)
                .build()
        def response = getVulnRequestClient().undoVulnerabilityRequest(id)
        // Allow propagation of CVE suppression and invalidation of cache
        sleep(15000 * (ClusterService.isOpenShift4() ? 4 : 1))
        return response
    }

    static globalScope() {
        return VulnerabilityRequest.Scope.newBuilder().
                setGlobalScope(VulnerabilityRequest.Scope.Global.newBuilder()).
                        build()
    }
}
