import common.Constants
import io.stackrox.proto.api.v1.ServiceAccountServiceOuterClass
import services.ServiceAccountService
import spock.lang.Stepwise
import util.Timer

@Stepwise
class K8sRbacTest extends BaseSpecification {
    private static final String SERVICE_ACCOUNT_NAME = "test-service-account"

    def "Verify scraped service accounts"() {
        given:
        "list of service accounts from the orchestrator"
        def orchestratorSAs = orchestrator.getServiceAccounts()

        expect:
        "SR should have the same service accounts"
        Timer t = new Timer(15, 2)
        def stackroxSAs = ServiceAccountService.getServiceAccounts()

        // Make sure the qa namespace SA exists before running the test. That SA should be the most recent added.
        // This will ensure scrapping is complete if this test spec is run first
        while (t.IsValid() &&
                !stackroxSAs.find { it.serviceAccount.getNamespace() == Constants.ORCHESTRATOR_NAMESPACE }) {
            stackroxSAs = ServiceAccountService.getServiceAccounts()
        }

        println t.IsValid() ? "Found default SA for namespace" : "Never found default SA for namespace"
        assert t.IsValid()

        stackroxSAs.size() == orchestratorSAs.size()
        for (ServiceAccountServiceOuterClass.ServiceAccountAndRoles s : stackroxSAs) {
            def sa = s.serviceAccount
            println "Looking for SR Service Account: ${sa}"
            assert orchestratorSAs.find {
                it.metadata.name == sa.name &&
                    it.metadata.namespace == sa.namespace &&
                    it.metadata.labels == null ?: it.metadata.labels == sa.labelsMap &&
                    it.metadata.annotations == null ?: it.metadata.annotations == sa.annotationsMap &&
                    it.automountServiceAccountToken == null ? sa.automountToken :
                        it.automountServiceAccountToken == sa.automountToken &&
                    it.secrets*.name == sa.secretsList &&
                    it.imagePullSecrets*.name == sa.imagePullSecretsList
            }
            assert ServiceAccountService.getServiceAccountDetails(sa.id).getServiceAccount() == sa
        }
    }

    def "Add Service Account and verify it gets scraped"() {
        given:
        "create a new service account"
        orchestrator.createServiceAccount(SERVICE_ACCOUNT_NAME)

        expect:
        "SR should detect the new service account"
        ServiceAccountService.waitForServiceAccount(SERVICE_ACCOUNT_NAME)
    }

    def "Remove Service Account and verify it is removed"() {
        given:
        "delete the created service account"
        orchestrator.deleteServiceAccount(SERVICE_ACCOUNT_NAME)

        expect:
        "SR should not show the service account"
        ServiceAccountService.waitForServiceAccountRemoved(SERVICE_ACCOUNT_NAME)
    }
}
