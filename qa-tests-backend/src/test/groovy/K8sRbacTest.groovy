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
            orchestratorSAs = orchestrator.getServiceAccounts()
        }

        println t.IsValid() ? "Found default SA for namespace" : "Never found default SA for namespace"
        assert t.IsValid()

        stackroxSAs.size() == orchestratorSAs.size()
        for (ServiceAccountServiceOuterClass.ServiceAccountAndRoles s : stackroxSAs) {
            def sa = s.serviceAccount

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
                .setSkipReplicaWait(true)
        orchestrator.createDeployment(deployment)
        assert Services.waitForDeployment(deployment)

        orchestrator.createRole(NEW_ROLE)
        assert RbacService.waitForRole(NEW_ROLE)

        orchestrator.createRoleBinding(NEW_ROLE_BINDING_ROLE_REF)
        assert RbacService.waitForRoleBinding(NEW_ROLE_BINDING_ROLE_REF)

        expect:
        "SR should have the service account and its relationship to the deployment"
        Timer t = new Timer(45, 2)
        def passed = false
        while (t.IsValid() && !passed) {
            def stackroxSAs = ServiceAccountService.getServiceAccounts()
            for (ServiceAccountServiceOuterClass.ServiceAccountAndRoles s : stackroxSAs) {
                def sa = s.serviceAccount
                if ( sa.name == NEW_SA.name && sa.namespace == NEW_SA.namespace ) {
                    passed = s.deploymentRelationshipsCount == 1 && \
                        s.deploymentRelationshipsList[0].name == DEPLOYMENT_NAME && \
                        s.clusterRolesCount == 0 && s.scopedRolesCount == 1 && \
                        s.scopedRolesList[0].getRoles(0).name == ROLE_NAME
                    if (passed) {
                        break
                    }
                }
            }
        }
        if (!passed) {
            println "Failed to find the correct service account values"
        }
        assert passed

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
        expect:
        "SR should have the same roles"

        withRetry(20, 5) {
            def stackroxRoles = RbacService.getRoles()
            def orchestratorRoles = orchestrator.getRoles() + orchestrator.getClusterRoles()

            stackroxRoles.size() == orchestratorRoles.size()
            for (Rbac.K8sRole stackroxRole : stackroxRoles) {
                println "Looking for SR Role: ${stackroxRole.name} (${stackroxRole.namespace})"
                K8sRole role = orchestratorRoles.find {
                    it.name == stackroxRole.name &&
                            it.clusterRole == stackroxRole.clusterRole &&
                            it.namespace == stackroxRole.namespace
                }
                assert role
                assert role.labels == stackroxRole.labelsMap
                role.annotations.remove("kubectl.kubernetes.io/last-applied-configuration")
                assert role.annotations == stackroxRole.annotationsMap
                for (int i = 0; i < role.rules.size(); i++) {
                    def oRule = role.rules.get(i) as K8sPolicyRule
                    def sRule = stackroxRole.rulesList.get(i) as Rbac.PolicyRule
                    assert oRule.verbs == sRule.verbsList &&
                            oRule.apiGroups == sRule.apiGroupsList &&
                            oRule.resources == sRule.resourcesList &&
                            oRule.nonResourceUrls == sRule.nonResourceUrlsList &&
                            oRule.resourceNames == sRule.resourceNamesList
                }
                assert RbacService.getRole(stackroxRole.id) == stackroxRole
            }
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
        expect:
        "SR should have the same bindings"
        Timer t = new Timer(45, 2)
        def passed = false
        while (t.IsValid() && !passed) {
            def stackroxBindings = RbacService.getRoleBindings()
            def orchestratorBindings = orchestrator.getRoleBindings() + orchestrator.getClusterRoleBindings()

            if (stackroxBindings.size() != orchestratorBindings.size()) {
                continue
            }
            println "All bindings scraped in ${t.SecondsSince()}s"
            passed = true
            for (Rbac.K8sRoleBinding b : stackroxBindings) {
                K8sRoleBinding binding =  orchestratorBindings.find {
                    it.name == b.name &&
                            it.namespace == b.namespace
                }
                if (binding == null || b.labelsMap != binding.labels) {
                    passed = false
                    break
                }
                binding.annotations.remove("kubectl.kubernetes.io/last-applied-configuration")
                if (b.annotationsMap != binding.annotations ||
                        b.roleId != binding.roleRef.uid || b.subjectsCount != binding.subjects.size()) {
                    passed = false
                    break
                }
                def allMatching = true
                for (int i = 0; i < binding.subjects.size(); i++) {
                    def oSubject = binding.subjects.get(i) as K8sSubject
                    def sSubject = b.subjectsList.get(i) as Rbac.Subject
                    if (!(sSubject.name == oSubject.name &&
                            oSubject.namespace == null ?
                            sSubject.namespace == "" :
                            sSubject.namespace == oSubject.namespace &&
                                    CaseFormat.UPPER_UNDERSCORE.to(CaseFormat.UPPER_CAMEL, sSubject.kind.toString()) ==
                                    oSubject.kind)) {
                        allMatching = false
                    }
                }
                if (!allMatching) {
                    passed = false
                    break
                }

                if (RbacService.getRoleBinding(b.id) != b) {
                    passed = false
                    break
                }
            }
        }
        if (!passed) {
            println "Failed to verify scraped bindings"
        }
        assert passed
    }

    def "Verify returned subject list is complete"() {
        given:
        "list of bindings from the orchestrator, we will pull unique subjects from this list"
        def orchestratorSubjects = fetchOrchestratorSubjects()

        expect:
        "SR should have the same subjects"
        def stackroxSubjectsAndRoles = RbacService.getSubjects()
        Timer t = new Timer(15, 2)
        while (t.IsValid() && stackroxSubjectsAndRoles.size() != orchestratorSubjects.size()) {
            stackroxSubjectsAndRoles = RbacService.getSubjects()
            orchestratorSubjects = fetchOrchestratorSubjects()
        }
        def stackroxSubjects = stackroxSubjectsAndRoles*.subject

        assert stackroxSubjects.size() == orchestratorSubjects.size()
        println "All subjects scraped in ${t.SecondsSince()}s"
        for (Rbac.Subject sub : stackroxSubjects) {
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

    private fetchOrchestratorSubjects() {
        def orchestratorBindings = orchestrator.getRoleBindings() + orchestrator.getClusterRoleBindings()
        def orchestratorSubjects = orchestratorBindings.collectMany {
            it.subjects
        }.findAll {
            it.kind == "User" || it.kind == "Group"
        }
        orchestratorSubjects.unique { a, b -> a.name <=> b.name }
        return orchestratorSubjects
    }
}
