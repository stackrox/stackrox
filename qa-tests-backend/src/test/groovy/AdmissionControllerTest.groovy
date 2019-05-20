import groups.BAT
import io.stackrox.proto.storage.ClusterOuterClass.AdmissionControllerConfig
import io.stackrox.proto.storage.PolicyOuterClass
import objects.Deployment
import orchestratormanager.OrchestratorTypes
import org.junit.Assume
import org.junit.experimental.categories.Category
import services.ClusterService
import spock.lang.Shared
import spock.lang.Unroll
import util.Env

class AdmissionControllerTest extends BaseSpecification {
    @Shared
    private List<PolicyOuterClass.EnforcementAction> latestTagEnforcements
    @Shared
    private List<PolicyOuterClass.EnforcementAction> cvssEnforcements
    @Shared
    private String gcrId

    static final private String GCR_NGINX         = "qagcrnginx"
    static final private String BUSYBOX_NO_BYPASS = "busybox-no-bypass"
    static final private String BUSYBOX_BYPASS    = "busybox-bypass"

    private final static String LATEST_TAG = "Latest tag"
    private final static String CVSS = "Fixable CVSS >= 7"

    static final private Deployment GCR_NGINX_DEPLOYMENT = new Deployment()
            .setName(GCR_NGINX)
            .setImage("us.gcr.io/stackrox-ci/nginx:1.10")
            .addLabel("app", "test")

    static final private Deployment BUSYBOX_NO_BYPASS_DEPLOYMENT = new Deployment()
            .setName(BUSYBOX_NO_BYPASS)
            .setImage("busybox:latest")
            .addLabel("app", "test")

    static final private Deployment BUSYBOX_BYPASS_DEPLOYMENT = new Deployment()
            .setName(BUSYBOX_BYPASS)
            .setImage("busybox:latest")
            .addLabel("app", "test")
            .addAnnotation("admission.stackrox.io/break-glass", "yay")

    def setupSpec() {
        latestTagEnforcements = Services.updatePolicyEnforcement(
                LATEST_TAG,
                [PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,]
        )

        cvssEnforcements = Services.updatePolicyEnforcement(
                CVSS,
                [PolicyOuterClass.EnforcementAction.SCALE_TO_ZERO_ENFORCEMENT,]
        )

        gcrId = Services.addGcrRegistryAndScanner()
        assert gcrId != null
    }

    def cleanupSpec() {
        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                .setEnabled(false)
                .build()

        assert ClusterService.updateAdmissionController(ac)

        Services.updatePolicyEnforcement(
                LATEST_TAG,
                latestTagEnforcements
        )

        Services.updatePolicyEnforcement(
                CVSS,
                cvssEnforcements
        )
        assert Services.deleteGcrRegistryAndScanner(gcrId)
    }

    @Unroll
    @Category([BAT])
    def "Verify Admission Controller Config"() {
        when:
        Assume.assumeFalse(Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT)

        AdmissionControllerConfig ac = AdmissionControllerConfig.newBuilder()
                                .setEnabled(true)
                                .setDisableBypass(!bypassable)
                                .setScanInline(scan)
                                .setTimeoutSeconds(timeout)
                            .build()

        assert ClusterService.updateAdmissionController(ac)
        // Maximum time to wait for propagation to sensor
        sleep 5000

        then:
        "Run deployment request"
        def created = orchestrator.createDeploymentNoWait(deployment)
        assert created == launched

        cleanup:
        "Revert Cluster"
        if (created) {
            orchestrator.deleteDeployment(deployment)
        }

        where:
        "Data inputs are: "

        timeout | scan  | bypassable | deployment                   | launched
        3       | false | false      | BUSYBOX_NO_BYPASS_DEPLOYMENT | false
        3       | false | false      | BUSYBOX_BYPASS_DEPLOYMENT    | false
        3       | false | true       | BUSYBOX_BYPASS_DEPLOYMENT    | true
        30      | true  | false      | GCR_NGINX_DEPLOYMENT         | false
    }

}
