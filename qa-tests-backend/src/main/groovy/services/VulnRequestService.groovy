package services

import io.stackrox.proto.api.v1.Common
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.api.v1.VulnReqService
import io.stackrox.proto.api.v1.VulnerabilityRequestServiceGrpc
import io.stackrox.proto.storage.VulnRequests.VulnerabilityRequest
import util.Helpers

class VulnRequestService extends BaseService {
    static getVulnRequestClient() {
        return VulnerabilityRequestServiceGrpc.newBlockingStub(getChannel())
    }

    static listVulnRequests() {
        return getVulnRequestClient().listVulnerabilityRequests(SearchServiceOuterClass.RawQuery.newBuilder().build())
    }

    static getVulnReq(String reqID) {
        def id = Common.ResourceByID.newBuilder()
                .setId(reqID)
                .build()
        return getVulnRequestClient().getVulnerabilityRequest(id)
    }

    static deferVuln(String cve, String comment, VulnerabilityRequest.Scope scope) {
        def req = VulnReqService.DeferVulnRequest.newBuilder().
                setCve(cve).
                setScope(scope).
                setComment(comment).
                setExpiresWhenFixed(true).
                build()
        return getVulnRequestClient().deferVulnerability(req)
    }

    static markVulnAsFP(String cve, String comment, VulnerabilityRequest.Scope scope) {
        def req = VulnReqService.FalsePositiveVulnRequest.newBuilder().
                setCve(cve).
                setScope(scope).
                setComment(comment).
                build()
        return getVulnRequestClient().falsePositiveVulnerability(req)
    }

    static approveRequest(String reqID, String comment) {
        def req = VulnReqService.ApproveVulnRequest.newBuilder().
                setId(reqID).
                setComment(comment).
                build()
        return getVulnRequestClient().approveVulnerabilityRequest(req)
    }

    static denyRequest(String reqID, String comment) {
        def req = VulnReqService.DenyVulnRequest.newBuilder().
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
        Helpers.sleepWithRetryBackoff(15000 * (ClusterService.isOpenShift4() ? 4 : 1))
        return response
    }

    static globalScope() {
        return VulnerabilityRequest.Scope.newBuilder().
                setGlobalScope(VulnerabilityRequest.Scope.Global.newBuilder()).
                        build()
    }

    static imageScope(String fullImageName, boolean allTags) {
        def imageParts = fullImageName.split(":")
        def tag = allTags ? ".*" : imageParts[1]
        def idx = imageParts[0].indexOf('/')

        def imageBuilder = VulnerabilityRequest.Scope.Image.newBuilder().
                setRegistry(imageParts[0].substring(0, idx)).
                setRemote(imageParts[0].substring(idx+1)).
                setTag(tag)

        return VulnerabilityRequest.Scope.newBuilder().
                setImageScope(imageBuilder).
                build()
    }
}
