import org.junit.Assume
import services.ClusterService
import util.Env

class OpenShiftInternalImageScanningTest extends BaseSpecification {

    private static final String OPENSHIFT4_REGISTRY = "image-registry.openshift-image-registry.svc:5000"

    def "Verify image scan finds correct base OS - #imageName"() {
        given:
        Assume.assumeTrue(ClusterService.isOpenShift4())
        Assume.assumeTrue(Env.CI_JOBNAME == "openshift-4-api-e2e-tests")
        when:
        def img = Services.scanImage(OPENSHIFT4_REGISTRY + "/" + project + "/" + imageRemote + ":" + imageTag)
        then:
        assert img.scan.operatingSystem == expected
        where:
        "Data inputs are: "

        imageName | project     | imageRemote | imageTag | expected
        "java:8"  | "openshift" | "java"      | "8"      | "rhel:8"
    }
}
