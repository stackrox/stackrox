import groups.OpenShift
import groups.OpenShift4
import org.junit.experimental.categories.Category

@Category(OpenShift)
class OpenShiftInternalImageScanningTest extends BaseSpecification {

    private static String OPENSHIFT4_REGISTRY = "image-registry.openshift-image-registry.svc:5000"

    @Category(OpenShift4)
    def "Verify image scan finds correct base OS - #imageName"() {
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
