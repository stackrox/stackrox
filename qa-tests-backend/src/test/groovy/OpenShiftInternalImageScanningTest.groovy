import groups.OpenShift
import org.junit.experimental.categories.Category

@Category(OpenShift)
class OpenShiftInternalImageScanningTest extends BaseSpecification {

    private static String[] projects = ["myproject1", "myproject2"]

    def setupSpec() {
        for (String project : projects) {
            orchestrator.ensureNamespaceExists(project)
        }


    }

    def cleanupSpec() {
        for (String project : projects) {
            orchestrator.deleteNamespace(project)
            orchestrator.waitForNamespaceDeletion(project)
        }
    }

    def "Verify image scan finds correct base OS - #imageName"() {

    }
}
