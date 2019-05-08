import common.Constants
import io.stackrox.proto.api.v1.ServiceAccountServiceOuterClass
import io.stackrox.proto.storage.Rbac
import objects.Deployment
import objects.K8sPolicyRule
import objects.K8sRole
import objects.K8sServiceAccount
import services.RbacService
import services.ServiceAccountService
import spock.lang.Stepwise
import util.Timer

@Stepwise
class K8sRbacTest extends BaseSpecification {
    private static final String SERVICE_ACCOUNT_NAME = "test-service-account"
    private static final String ROLE_NAME = "test-role"
    private static final String CLUSTER_ROLE_NAME = "test-cluster-role"
    private static final String DEPLOYMENT_NAME = "test-deployment"

    private static final K8sServiceAccount NEW_SA = new K8sServiceAccount(
            name: SERVICE_ACCOUNT_NAME,
            namespace: Constants.ORCHESTRATOR_NAMESPACE)
    private static final K8sRole NEW_ROLE = new K8sRole(name: ROLE_NAME, namespace: Constants.ORCHESTRATOR_NAMESPACE)
    private static final K8sRole NEW_CLUSTER_ROLE = new K8sRole(name: CLUSTER_ROLE_NAME, clusterRole: true)

    def cleanupSpec() {
        orchestrator.deleteServiceAccount(NEW_SA)
        orchestrator.deleteRole(NEW_ROLE)
        orchestrator.deleteClusterRole(NEW_CLUSTER_ROLE)
    }

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
                it.name == sa.name &&
                    it.namespace == sa.namespace &&
                    it.labels == null ?: it.labels == sa.labelsMap &&
                    it.annotations == null ?: it.annotations == sa.annotationsMap &&
                    it.automountToken == null ? sa.automountToken :
                        it.automountToken == sa.automountToken &&
                    it.secrets == sa.secretsList &&
                    it.imagePullSecrets == sa.imagePullSecretsList
            }
            assert ServiceAccountService.getServiceAccountDetails(sa.id).getServiceAccount() == sa
        }
    }

    def "Add Service Account and verify it gets scraped"() {
        given:
        "create a new service account"
        orchestrator.createServiceAccount(NEW_SA)

        expect:
        "SR should detect the new service account"
        ServiceAccountService.waitForServiceAccount(NEW_SA)
    }

    def "Create deployment with service account and verify relationships"() {
        given:

        Deployment deployment = new Deployment()
                .setName(DEPLOYMENT_NAME)
                .setNamespace(Constants.ORCHESTRATOR_NAMESPACE)
                .setServiceAccountName(SERVICE_ACCOUNT_NAME)
                .setImage("nginx:1.15.4-alpine")
        orchestrator.createDeployment(deployment)
        assert Services.waitForDeployment(deployment)

        expect:
        "SR should have the service account and its relationship to the deployment"
        def stackroxSAs = ServiceAccountService.getServiceAccounts()
        for (ServiceAccountServiceOuterClass.ServiceAccountAndRoles s : stackroxSAs) {
            def sa = s.serviceAccount
            if ( sa.name == NEW_SA.name && sa.namespace == NEW_SA.namespace ) {
                assert(s.deploymentRelationshipsCount == 1)
                assert(s.deploymentRelationshipsList[0].name == DEPLOYMENT_NAME)
            }
        }

        cleanup:
        orchestrator.deleteAndWaitForDeploymentDeletion(deployment)
    }

    def "Remove Service Account and verify it is removed"() {
        given:
        "delete the created service account"
        orchestrator.deleteServiceAccount(NEW_SA)

        expect:
        "SR should not show the service account"
        ServiceAccountService.waitForServiceAccountRemoved(NEW_SA)
    }

    def "Verify scraped roles"() {
        given:
        "list of roles from the orchestrator"
        def orchestratorRoles = orchestrator.getRoles() + orchestrator.getClusterRoles()

        expect:
        "SR should have the same service accounts"
        def stackroxRoles = RbacService.getRoles()

        stackroxRoles.size() == orchestratorRoles.size()
        for (Rbac.K8sRole r : stackroxRoles) {
            println "Looking for SR Role: ${r.name} (${r.namespace})"
            K8sRole role =  orchestratorRoles.find {
                it.name == r.name &&
                        it.clusterRole == r.clusterRole &&
                        it.namespace == r.namespace
            }
            assert role
            assert role.labels == r.labelsMap
            assert role.annotations == r.annotationsMap
            for (int i = 0; i < role.rules.size(); i++) {
                def oRule = role.rules.get(i) as K8sPolicyRule
                def sRule = r.rulesList.get(i) as Rbac.PolicyRule
                assert oRule.verbs == sRule.verbsList &&
                        oRule.apiGroups == sRule.apiGroupsList &&
                        oRule.resources == sRule.resourcesList &&
                        oRule.nonResourceUrls == sRule.nonResourceUrlsList &&
                        oRule.resourceNames == sRule.resourceNamesList
            }
            assert RbacService.getRole(r.id) == r
        }
    }

    def "Add Role and verify it gets scraped"() {
        given:
        "create a new role"
        orchestrator.createRole(NEW_ROLE)

        expect:
        "SR should detect the new service account"
        RbacService.waitForRole(NEW_ROLE)
    }

    def "Remove Role and verify it is removed"() {
        given:
        "delete the created service account"
        orchestrator.deleteRole(NEW_ROLE)

        expect:
        "SR should not show the service account"
        RbacService.waitForRoleRemoved(NEW_ROLE)
    }

    def "Add Cluster Role and verify it gets scraped"() {
        given:
        "create a new role"
        orchestrator.createClusterRole(NEW_CLUSTER_ROLE)

        expect:
        "SR should detect the new service account"
        RbacService.waitForRole(NEW_CLUSTER_ROLE)
    }

    def "Remove Cluster Role and verify it is removed"() {
        given:
        "delete the created service account"
        orchestrator.deleteClusterRole(NEW_CLUSTER_ROLE)

        expect:
        "SR should not show the service account"
        RbacService.waitForRoleRemoved(NEW_CLUSTER_ROLE)
    }
}
