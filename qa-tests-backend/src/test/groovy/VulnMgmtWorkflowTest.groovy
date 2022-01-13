import common.Constants
import groups.BAT
import groups.RUNTIME
import io.stackrox.proto.api.v1.SearchServiceOuterClass
import io.stackrox.proto.storage.Cve
import io.stackrox.proto.storage.PolicyOuterClass
import io.stackrox.proto.storage.ScopeOuterClass
import io.stackrox.proto.storage.VulnRequests
import io.stackrox.proto.storage.Vulnerability
import objects.Deployment
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ClusterService
import services.DeploymentService
import services.FeatureFlagService
import services.ImageService
import services.PolicyService
import services.VulnRequestService
import spock.lang.Unroll
import util.Helpers

class VulnMgmtWorkflowTest extends BaseSpecification {

    static final private NGINX_1_10_2_IMAGE = "us.gcr.io/stackrox-ci/nginx:1.10.2"
//    static final private NGINX_1_10_2_IMAGE = "docker.io/vulhub/log4j:2.8.1"

    static final private Deployment CVE_DEPLOYMENT = new Deployment()
            .setName("vulnerable-deploy")
            .setImage(NGINX_1_10_2_IMAGE)
            .addLabel("app", "test")

    static final private Deployment CVE_DEPLOYMENT_FOR_ENFORCE = new Deployment()
            .setName("vulnerable-deploy-enforce")
            .setImage(NGINX_1_10_2_IMAGE)
            .addLabel("app", "test")

    static final private String CVE_TO_DEFER = "CVE-2009-5155"
//    static final private String CVE_TO_DEFER = "CVE-2021-44228"
    static final private String CVE_TO_MARK_FP = "CVE-2007-6755"
//    static final private String CVE_TO_MARK_FP = "CVE-2004-0971"

    @Unroll
    @Category([BAT, RUNTIME])
    def "Verify Vulnerability Requests can transition between states - #requestType - approve?(#approve)"() {
        given:
        "Vuln Management Feature is enabled"
        Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled("ROX_VULN_RISK_MANAGEMENT"))

        and:
        "Requests and vulns currently in DB"

        def reqs = VulnRequestService.listVulnRequests()
        reqs.getRequestInfosList().forEach {
            if (it.getCves().getIds(0) == cve) {
                println "${it.getCves().getIdsList().toString()} -- ${it.getTargetState()} -- ${it.getStatus()} -- expired? ${it.getExpired()}"
            }
        }
        def q = SearchServiceOuterClass.RawQuery.newBuilder().setQuery("Image:${NGINX_1_10_2_IMAGE}").build()
        def listImgs = ImageService.getImages(q)
        if (listImgs != null && listImgs.size() > 0) {
            def imgId = listImgs.first().getId()
            def img = ImageService.getImage(imgId, true)
            img.getScan().getComponentsList().forEach {
                it.getVulnsList().forEach {
                    if (it.getCve() == cve) {
                        println "${it.getCve()} -- Suppressed? ${it.getSuppressed()} -- State: ${it.getState()}"
                    }
                }
            }
        } else {
            println "No image found"
        }


        when:
        "A user requests a vuln be deferred or marked as FP"
        def vulnReq = createPendingVulnRequest(requestType, cve, VulnRequestService.globalScope())

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
        assert req.getTargetState() == requestType
        if (approve) {
            assert !req.getExpired()
        } else {
            assert req.getExpired()
        }
        assert req.getCves().getIdsCount() == 1
        assert req.getCves().getIds(0) ==
                (requestType == Cve.VulnerabilityState.DEFERRED ? CVE_TO_DEFER: CVE_TO_MARK_FP)
        assert req.getCommentsCount() == 2
        assert req.getComments(0).getMessage() == "${requestType} me" &&
                req.getComments(1).getMessage() == "actioned"

        cleanup:
        if (approve) {
            VulnRequestService.undoReq(id)
        }

        where:
        "Data inputs are:"

        requestType | cve | approve

        Cve.VulnerabilityState.DEFERRED  | CVE_TO_DEFER | true
        Cve.VulnerabilityState.DEFERRED  | CVE_TO_DEFER | false
        Cve.VulnerabilityState.FALSE_POSITIVE  | CVE_TO_MARK_FP | true
        Cve.VulnerabilityState.FALSE_POSITIVE | CVE_TO_MARK_FP | false
    }

    @SuppressWarnings('LineLength') // the test cases are too annoying to break up into multiple lines
    @Unroll
    @Category([BAT, RUNTIME])
    def "Vulnerabilities with approved requests don't trigger policies - #msg"() {
        given:
        "Vuln Management Feature is enabled"
        Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled("ROX_VULN_RISK_MANAGEMENT"))

        and:
        "Policy created on a CVE"
        def policy = createCVEPolicy("e2e-vuln-${requestType}", cve, false)
        def policyId = PolicyService.createNewPolicy(policy)
        assert policyId

        when:
        "CVE is deferred/marked as FP"
        def vulnReq = createPendingVulnRequest(requestType, cve, requestScope)
        VulnRequestService.approveRequest(vulnReq.getId(), "approved")
        // Maximum time to wait for propagation to sensor
        Helpers.sleepWithRetryBackoff(5000 * (ClusterService.isOpenShift4() ? 4 : 1))

        and:
        "A deployment with an image with the CVE is deployed"
        orchestrator.createDeployment(CVE_DEPLOYMENT)
        // CVE needs to be saved into the DB
        Helpers.sleepWithRetryBackoff(1000)

        then:
        "Policy shouldn't cause a violation"
        def violations = Services.getViolationsByDeploymentID(
                CVE_DEPLOYMENT.getDeploymentUid(), policy.getName(), false, 60)
        assert VulnRequestService.getVulnReq(vulnReq.getId()) != null && (violations == null || violations.size() == 0)

        cleanup:
        if (policyId) {
            PolicyService.deletePolicy(policyId)
        }
        orchestrator.deleteDeployment(CVE_DEPLOYMENT)
        VulnRequestService.undoReq(vulnReq.getId())

        where:
        "Data inputs are:"

        requestType | requestScope | cve | msg

        Cve.VulnerabilityState.DEFERRED  | VulnRequestService.globalScope() | CVE_TO_DEFER | "deferred global scope"
        Cve.VulnerabilityState.DEFERRED  | VulnRequestService.imageScope(NGINX_1_10_2_IMAGE, true) | CVE_TO_DEFER | "deferred image scope with wildcard"
        Cve.VulnerabilityState.DEFERRED  | VulnRequestService.imageScope(NGINX_1_10_2_IMAGE, false) | CVE_TO_DEFER | "deferred image scope without wildcard"
        Cve.VulnerabilityState.FALSE_POSITIVE | VulnRequestService.globalScope() | CVE_TO_MARK_FP | "false positive global scope"
        Cve.VulnerabilityState.FALSE_POSITIVE | VulnRequestService.imageScope(NGINX_1_10_2_IMAGE, true) | CVE_TO_MARK_FP | "false positive image scope with wildcard"
        Cve.VulnerabilityState.FALSE_POSITIVE | VulnRequestService.imageScope(NGINX_1_10_2_IMAGE, false) | CVE_TO_MARK_FP | "false positive image scope without wildcard"
    }

    @Unroll
    @Category([BAT, RUNTIME])
    def "Policies with enforcement aren't enforced once a vulnerability has an approved request - #requestType"() {
        given:
        "Vuln Management Feature is enabled"
        Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled("ROX_VULN_RISK_MANAGEMENT"))

        and:
        "Policy created on a CVE with enforcement"
        def policyId = PolicyService.createNewPolicy(
                createCVEPolicy("e2e-vuln-${requestType}-enforce", cve, true)
        )
        assert policyId
        // Maximum time to wait for propagation to sensor
        Helpers.sleepWithRetryBackoff(5000 * (ClusterService.isOpenShift4() ? 4 : 1))

        and:
        "Deployment is scaled to zero due to policy enforcement"
        orchestrator.createDeploymentNoWait(CVE_DEPLOYMENT_FOR_ENFORCE)

        def replicaCount = orchestrator.getDeploymentReplicaCount(CVE_DEPLOYMENT_FOR_ENFORCE)
        orchestrator.getDeploymentStatus(CVE_DEPLOYMENT_FOR_ENFORCE)

        def violations = Services.getViolationsWithTimeout(
                CVE_DEPLOYMENT_FOR_ENFORCE.name, "e2e-vuln-${requestType}-enforce", 60).collect {
            it.getPolicy().name
        }
        println "Policies violated: ${violations.toString()}"

        def startTime = System.currentTimeMillis()
        while (replicaCount > 0 && (System.currentTimeMillis() - startTime) < 60000) {
            replicaCount = orchestrator.getDeploymentReplicaCount(CVE_DEPLOYMENT_FOR_ENFORCE)
            orchestrator.getDeploymentStatus(CVE_DEPLOYMENT_FOR_ENFORCE)

            def violations2 = Services.getViolationsWithTimeout(
                    CVE_DEPLOYMENT_FOR_ENFORCE.name, "e2e-vuln-${requestType}-enforce", 60).collect {
                it.getPolicy().name
            }
            println "Policies violated: ${violations2.toString()}"

            sleep 1000
        }
        assert replicaCount == 0

        when:
        "CVE is deferred/marked as FP"
        def vulnReq = createPendingVulnRequest(requestType, cve, VulnRequestService.globalScope())
        // Approve
        VulnRequestService.approveRequest(vulnReq.getId(), "approved")
        // Maximum time to wait for propagation to sensor
        Helpers.sleepWithRetryBackoff(5000 * (ClusterService.isOpenShift4() ? 4 : 1))

        then:
        "Deployment is not blocked due to policy enforcement"
        assert orchestrator.createDeploymentNoWait(CVE_DEPLOYMENT_FOR_ENFORCE)

        cleanup:
        if (policyId) {
            PolicyService.deletePolicy(policyId)
        }
        orchestrator.deleteDeployment(CVE_DEPLOYMENT_FOR_ENFORCE)
        VulnRequestService.undoReq(vulnReq.getId())

        where:
        "Data inputs are:"

        requestType | cve

        Cve.VulnerabilityState.DEFERRED  | CVE_TO_DEFER
        Cve.VulnerabilityState.FALSE_POSITIVE | CVE_TO_MARK_FP
    }

//    @Category([BAT, RUNTIME])
//    def "Vulnerabilities with approved requests don't show up in risk factors"() {
//        given:
//        "Vuln Management Feature is enabled"
//        Assume.assumeTrue(FeatureFlagService.isFeatureFlagEnabled("ROX_VULN_RISK_MANAGEMENT"))
//
//        and:
//        "A deployment with an image with a known CVE is deployed"
//        orchestrator.createDeployment(CVE_DEPLOYMENT)
//
//        and:
//        "Intial risk is fetched"
//        def initialRisk = Services.getDeploymentWithRisk(CVE_DEPLOYMENT.deploymentUid).getRisk()
//        assert initialRisk.getResultsCount() > 0
//        def imgRiskMessage = initialRisk.getResultsList().find {
//            it.name == "Image Vulnerabilities"
//        }.getFactors(0).getMessage()
//
//        def matches = imgRiskMessage =~ /Image \"${NGINX_1_10_2_IMAGE}\" contains (\d+) CVEs with severities (.+)$/
//        def initialCVECount = matches[0][1]
//
//        when:
//        "CVE is deferred/marked as FP"
//        def vulnReq = createPendingVulnRequest(Cve.VulnerabilityState.DEFERRED, CVE_TO_DEFER, VulnRequestService.globalScope())
//
//        // Approve
//        VulnRequestService.approveRequest(vulnReq.getId(), "approved")
//
//        // Maximum time to wait for propagation to sensor
//        Helpers.sleepWithRetryBackoff(5000 * (ClusterService.isOpenShift4() ? 4 : 1))
//
//        then:
//        "Risk should be updated to have one fewer CVE"
//        def updatedRisk = Services.getDeploymentWithRisk(CVE_DEPLOYMENT.deploymentUid).getRisk()
//        assert updatedRisk.getResultsCount() > 0
//        def updatedRiskMessage = updatedRisk.getResultsList().find {
//            it.name == "Image Vulnerabilities"
//        }.getFactors(0).getMessage()
//
//        def updatedMatches = updatedRiskMessage =~ /Image \"${NGINX_1_10_2_IMAGE}\" contains (\d+) CVEs with severities (.+)$/
//        def updatedCVECount = updatedMatches[0][1]
//
//        assert ((initialCVECount - updatedCVECount) == 1) &&
//                imgRiskMessage == "" && updatedRiskMessage == ""
//
//        cleanup:
//        orchestrator.deleteDeployment(CVE_DEPLOYMENT)
//        VulnRequestService.undoReq(vulnReq.getId())
//    }

    def createCVEPolicy(String name, String cve, boolean enforce) {
        def builder = PolicyOuterClass.Policy.newBuilder()
                .setName(name)
                .addLifecycleStages(PolicyOuterClass.LifecycleStage.DEPLOY)
                .addCategories("Test")
                .setDisabled(false)
                .setSeverity(PolicyOuterClass.Severity.CRITICAL_SEVERITY)
                .addScope(
                        ScopeOuterClass.Scope.newBuilder().setNamespace(Constants.ORCHESTRATOR_NAMESPACE).build()
                )
                .addPolicySections(
                        PolicyOuterClass.PolicySection.newBuilder().addPolicyGroups(
                                PolicyOuterClass.PolicyGroup.newBuilder()
                                        .setFieldName("CVE")
                                        .addValues(PolicyOuterClass.PolicyValue.newBuilder().setValue(cve))
                        )
                )

        if (enforce) {
            builder.addEnforcementActions(PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT)
        }

        return builder.build()
    }
    def createPendingVulnRequest(Cve.VulnerabilityState requestType, String cve,
                                 VulnRequests.VulnerabilityRequest.Scope scope) {
        if (requestType == Cve.VulnerabilityState.DEFERRED) {
            return VulnRequestService.deferVuln(cve, "${requestType} me", scope).getRequestInfo()
        }
        return VulnRequestService.markVulnAsFP(cve, "${requestType} me", scope).getRequestInfo()
    }
}
