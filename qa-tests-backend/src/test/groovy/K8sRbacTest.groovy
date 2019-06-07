import com.google.common.base.CaseFormat
import common.Constants
import io.stackrox.proto.api.v1.ServiceAccountServiceOuterClass
import io.stackrox.proto.storage.Rbac
import objects.Deployment
import objects.K8sPolicyRule
import objects.K8sRole
import objects.K8sRoleBinding
import objects.K8sServiceAccount
import objects.K8sSubject
import org.junit.Assume
import services.FeatureFlagService
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

    private static final K8sRole NEW_ROLE =
            new K8sRole(name: ROLE_NAME, namespace: Constants.ORCHESTRATOR_NAMESPACE)

    private static final K8sRole NEW_CLUSTER_ROLE =
            new K8sRole(name: CLUSTER_ROLE_NAME, clusterRole: true)

    private static final K8sRoleBinding NEW_ROLE_BINDING_ROLE_REF =
            new K8sRoleBinding(NEW_ROLE, [new K8sSubject(NEW_SA)])

    private static final K8sRoleBinding NEW_ROLE_BINDING_CLUSTER_ROLE_REF =
            new K8sRoleBinding(NEW_CLUSTER_ROLE, [new K8sSubject(NEW_SA)])

    private static final K8sRoleBinding NEW_CLUSTER_ROLE_BINDING =
            new K8sRoleBinding(NEW_CLUSTER_ROLE, [new K8sSubject(NEW_SA)])

    def cleanupSpec() {
        orchestrator.deleteServiceAccount(NEW_SA)
        orchestrator.deleteRole(NEW_ROLE)
        orchestrator.deleteClusterRole(NEW_CLUSTER_ROLE)
    }

    def setup() {
        Assume.assumeTrue(
                FeatureFlagService.isFeatureFlagEnabled(Constants.K8SRBAC_FEATURE_FLAG)
        )
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

        orchestrator.createRole(NEW_ROLE)
        assert RbacService.waitForRole(NEW_ROLE)

        orchestrator.createRoleBinding(NEW_ROLE_BINDING_ROLE_REF)
        assert RbacService.waitForRoleBinding(NEW_ROLE_BINDING_ROLE_REF)

        expect:
        "SR should have the service account and its relationship to the deployment"
        def stackroxSAs = ServiceAccountService.getServiceAccounts()
        for (ServiceAccountServiceOuterClass.ServiceAccountAndRoles s : stackroxSAs) {
            def sa = s.serviceAccount
            if ( sa.name == NEW_SA.name && sa.namespace == NEW_SA.namespace ) {
                assert(s.deploymentRelationshipsCount == 1)
                assert(s.deploymentRelationshipsList[0].name == DEPLOYMENT_NAME)
                assert(s.clusterRolesCount == 0)
                assert(s.scopedRolesCount == 1)
                assert(s.scopedRolesList[0].getRoles(0).name == ROLE_NAME)
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
        "SR should have the same roles"
        def stackroxRoles = RbacService.getRoles()
        Timer t = new Timer(15, 2)
        while (t.IsValid() && stackroxRoles.size() != orchestratorRoles.size()) {
            stackroxRoles = RbacService.getRoles()
        }

        stackroxRoles.size() == orchestratorRoles.size()
        println "All roles scraped in ${t.SecondsSince()}s"
        for (Rbac.K8sRole r : stackroxRoles) {
            println "Looking for SR Role: ${r.name} (${r.namespace})"
            K8sRole role =  orchestratorRoles.find {
                it.name == r.name &&
                        it.clusterRole == r.clusterRole &&
                        it.namespace == r.namespace
            }
            assert role
            assert role.labels == r.labelsMap
            role.annotations.remove("kubectl.kubernetes.io/last-applied-configuration")
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
        "SR should detect the new role"
        RbacService.waitForRole(NEW_ROLE)
    }

    def "Remove Role and verify it is removed"() {
        given:
        "delete the created role"
        orchestrator.deleteRole(NEW_ROLE)

        expect:
        "SR should not show the role"
        RbacService.waitForRoleRemoved(NEW_ROLE)
    }

    def "Add Cluster Role and verify it gets scraped"() {
        given:
        "create a new cluster role"
        orchestrator.createClusterRole(NEW_CLUSTER_ROLE)

        expect:
        "SR should detect the new cluster role"
        RbacService.waitForRole(NEW_CLUSTER_ROLE)
    }

    def "Remove Cluster Role and verify it is removed"() {
        given:
        "delete the created cluster role"
        orchestrator.deleteClusterRole(NEW_CLUSTER_ROLE)

        expect:
        "SR should not show the cluster role"
        RbacService.waitForRoleRemoved(NEW_CLUSTER_ROLE)
    }

    def "Verify scraped bindings"() {
        given:
        "list of bindings from the orchestrator"
        def orchestratorBindings = orchestrator.getRoleBindings() + orchestrator.getClusterRoleBindings()

        expect:
        "SR should have the same bindings"
        def stackroxBindings = RbacService.getRoleBindings()
        Timer t = new Timer(15, 2)
        while (t.IsValid() && stackroxBindings.size() != orchestratorBindings.size()) {
            stackroxBindings = RbacService.getRoleBindings()
        }

        assert stackroxBindings.size() == orchestratorBindings.size()
        println "All bindings scraped in ${t.SecondsSince()}s"
        for (Rbac.K8sRoleBinding b : stackroxBindings) {
            println "Looking for SR Bindings: ${b.name} (${b.namespace})"
            K8sRoleBinding binding =  orchestratorBindings.find {
                it.name == b.name &&
                        it.namespace == b.namespace
            }
            assert binding
            assert b.labelsMap == binding.labels
            binding.annotations.remove("kubectl.kubernetes.io/last-applied-configuration")
            assert b.annotationsMap == binding.annotations
            assert b.roleId == binding.roleRef.uid
            assert b.subjectsCount == binding.subjects.size()
            for (int i = 0; i < binding.subjects.size(); i++) {
                def oSubject = binding.subjects.get(i) as K8sSubject
                def sSubject = b.subjectsList.get(i) as Rbac.Subject
                assert sSubject.name == oSubject.name &&
                        oSubject.namespace == null ?
                                sSubject.namespace == "" :
                                sSubject.namespace == oSubject.namespace &&
                        CaseFormat.UPPER_UNDERSCORE.to(CaseFormat.UPPER_CAMEL, sSubject.kind.toString()) ==
                        oSubject.kind
            }
            assert RbacService.getRoleBinding(b.id) == b
        }
    }

    def "Verify returned subject list is complete"() {
        given:
        "list of bindings from the orchestrator, we will pull unique subjects from this list"
        def orchestratorBindings = orchestrator.getRoleBindings() + orchestrator.getClusterRoleBindings()
        def orchestratorSubjects = orchestratorBindings.collectMany {
            it.subjects
        }.findAll {
            it.kind == "User" || it.kind == "Group"
        }
        orchestratorSubjects.unique { a, b -> a.name <=> b.name }

        expect:
        "SR should have the same subjects"
        def stackroxSubjectsAndRoles = RbacService.getSubjects()
        Timer t = new Timer(15, 2)
        while (t.IsValid() && stackroxSubjectsAndRoles.size() != orchestratorSubjects.size()) {
            stackroxSubjectsAndRoles = RbacService.getSubjects()
        }
        def stackroxSubjects = stackroxSubjectsAndRoles*.subject

        assert stackroxSubjects.size() == orchestratorSubjects.size()
        println "All subjects scraped in ${t.SecondsSince()}s"
        for (Rbac.Subject sub : stackroxSubjects) {
            println "Looking for SR Subject: ${sub.name} (${sub.namespace})"
            K8sSubject subject = orchestratorSubjects.find {
                it.name == sub.name &&
                        it.namespace == sub.namespace &&
                                it.kind.toLowerCase() == sub.kind.toString().toLowerCase()
            }
            assert subject
        }
    }

    def "Add Binding with role ref and verify it gets scraped"() {
        given:
        "create a new role binding"
        orchestrator.createRole(NEW_ROLE)
        orchestrator.createServiceAccount(NEW_SA)
        orchestrator.createRoleBinding(NEW_ROLE_BINDING_ROLE_REF)

        expect:
        "SR should detect the new role binding"
        RbacService.waitForRoleBinding(NEW_ROLE_BINDING_ROLE_REF)
    }

    def "Remove Binding with role ref and verify it is removed"() {
        given:
        "delete the created role binding"
        orchestrator.deleteRoleBinding(NEW_ROLE_BINDING_ROLE_REF)

        expect:
        "SR should not show the role binding"
        RbacService.waitForRoleBindingRemoved(NEW_ROLE_BINDING_ROLE_REF)

        cleanup:
        orchestrator.deleteServiceAccount(NEW_SA)
        orchestrator.deleteRole(NEW_ROLE)
    }

    def "Add Binding with cluster role ref and verify it gets scraped"() {
        given:
        "create a new role binding"
        orchestrator.createClusterRole(NEW_CLUSTER_ROLE)
        orchestrator.createServiceAccount(NEW_SA)
        NEW_ROLE_BINDING_CLUSTER_ROLE_REF.setNamespace(Constants.ORCHESTRATOR_NAMESPACE)
        orchestrator.createRoleBinding(NEW_ROLE_BINDING_CLUSTER_ROLE_REF)

        expect:
        "SR should detect the new role binding"
        RbacService.waitForRoleBinding(NEW_ROLE_BINDING_CLUSTER_ROLE_REF)
    }

    def "Remove Binding with clsuter role ref and verify it is removed"() {
        given:
        "delete the created role binding"
        orchestrator.deleteRoleBinding(NEW_ROLE_BINDING_CLUSTER_ROLE_REF)

        expect:
        "SR should not show the role binding"
        RbacService.waitForRoleBindingRemoved(NEW_ROLE_BINDING_CLUSTER_ROLE_REF)

        cleanup:
        orchestrator.deleteServiceAccount(NEW_SA)
        orchestrator.deleteClusterRole(NEW_CLUSTER_ROLE)
    }

    def "Add cluster Binding and verify it gets scraped"() {
        given:
        "create a new cluster role binding"
        orchestrator.createClusterRole(NEW_CLUSTER_ROLE)
        orchestrator.createServiceAccount(NEW_SA)
        orchestrator.createClusterRoleBinding(NEW_CLUSTER_ROLE_BINDING)

        expect:
        "SR should detect the new cluster role binding"
        RbacService.waitForRoleBinding(NEW_CLUSTER_ROLE_BINDING)
    }

    def "Remove cluster Binding and verify it is removed"() {
        given:
        "delete the created cluster role binding"
        orchestrator.deleteClusterRoleBinding(NEW_CLUSTER_ROLE_BINDING)

        expect:
        "SR should not show the cluster role binding"
        RbacService.waitForRoleBindingRemoved(NEW_CLUSTER_ROLE_BINDING)

        cleanup:
        orchestrator.createServiceAccount(NEW_SA)
        orchestrator.deleteClusterRole(NEW_CLUSTER_ROLE)
    }
}
