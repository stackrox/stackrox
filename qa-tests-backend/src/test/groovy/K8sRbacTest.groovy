import static util.Helpers.withRetry

import com.google.common.base.CaseFormat
import orchestratormanager.OrchestratorTypes

import io.stackrox.proto.api.v1.RbacServiceOuterClass
import io.stackrox.proto.api.v1.ServiceAccountServiceOuterClass
import io.stackrox.proto.storage.Rbac

import common.Constants
import objects.Deployment
import objects.K8sPolicyRule
import objects.K8sRole
import objects.K8sRoleBinding
import objects.K8sServiceAccount
import objects.K8sSubject
import services.RbacService
import services.ServiceAccountService
import util.Env

import spock.lang.IgnoreIf
import spock.lang.Stepwise
import spock.lang.Tag

@Stepwise
@Tag("PZ")
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

    @Tag("BAT")
    @Tag("COMPATIBILITY")
    // TODO(ROX-14666): This test times out under openshift
    @IgnoreIf({ Env.mustGetOrchestratorType() == OrchestratorTypes.OPENSHIFT })
    def "Verify scraped service accounts"() {
        given:
        List<K8sServiceAccount> orchestratorSAs = null
        List<ServiceAccountServiceOuterClass.ServiceAccountAndRoles> stackroxSAs = null

        expect:
        "SR should have the same service accounts"
        // Make sure the qa namespace SA exists before running the test. That SA should be the most recent added.
        // This will ensure scrapping is complete if this test spec is run first
        withRetry(15, 2) {
            stackroxSAs = ServiceAccountService.getServiceAccounts()
            // list of service accounts from the orchestrator
            orchestratorSAs = orchestrator.getServiceAccounts()
            assert stackroxSAs.find { it.serviceAccount.getNamespace() == Constants.ORCHESTRATOR_NAMESPACE }
        }

        stackroxSAs.size() == orchestratorSAs.size()
        for (ServiceAccountServiceOuterClass.ServiceAccountAndRoles s : stackroxSAs) {
            def sa = s.serviceAccount

            K8sServiceAccount k8sMatch = orchestratorSAs.find {
                ServiceAccountService.matchServiceAccounts(it, sa)
            }

            if (!k8sMatch) {
                log.info "SR serviceaccount ${sa.name} has no k8s match"
                log.info "SR serviceaccount: " + sa
                K8sServiceAccount nameOnlyMatch = orchestratorSAs.find {
                    it.name == sa.name &&
                            it.namespace == sa.namespace
                }
                if (nameOnlyMatch) {
                    log.info "K8S serviceaccount: " + nameOnlyMatch.dump()
                }
            }

            assert k8sMatch
            assert ServiceAccountService.getServiceAccountDetails(sa.id).getServiceAccount() == sa
        }
    }

    @Tag("BAT")
    @Tag("COMPATIBILITY")
    def "Add Service Account and verify it gets scraped"() {
        given:
        "create a new service account"
        orchestrator.createServiceAccount(NEW_SA)

        expect:
        "SR should detect the new service account"
        ServiceAccountService.waitForServiceAccount(NEW_SA)
    }

    @Tag("BAT")
    def "Create deployment with service account and verify relationships"() {
        given:
        Deployment deployment = new Deployment()
                .setName(DEPLOYMENT_NAME)
                .setNamespace(Constants.ORCHESTRATOR_NAMESPACE)
                .setServiceAccountName(SERVICE_ACCOUNT_NAME)
                .setImage("quay.io/rhacs-eng/qa-multi-arch:nginx-1-15-4-alpine")
                .setSkipReplicaWait(true)
        orchestrator.createDeployment(deployment)
        assert Services.waitForDeployment(deployment)

        orchestrator.createRole(NEW_ROLE)
        assert RbacService.waitForRole(NEW_ROLE)

        orchestrator.createRoleBinding(NEW_ROLE_BINDING_ROLE_REF)
        assert RbacService.waitForRoleBinding(NEW_ROLE_BINDING_ROLE_REF)

        expect:
        "SR should have the service account and its relationship to the deployment"
        def query = ServiceAccountService.getServiceAccountQuery(NEW_SA)
        withRetry(45, 2) {
            def stackroxSAs = ServiceAccountService.getServiceAccounts(query)
            for (ServiceAccountServiceOuterClass.ServiceAccountAndRoles s : stackroxSAs) {
                def sa = s.serviceAccount
                if (sa.name == NEW_SA.name && sa.namespace == NEW_SA.namespace) {
                    assert s.deploymentRelationshipsCount == 1
                    assert s.deploymentRelationshipsList[0].name == DEPLOYMENT_NAME
                    assert s.clusterRolesCount == 0 && s.scopedRolesCount == 1
                    assert s.scopedRolesList[0].getRoles(0).name == ROLE_NAME
                }
            }
        }

        cleanup:
        orchestrator.deleteAndWaitForDeploymentDeletion(deployment)
    }

    @Tag("BAT")
    def "Remove Service Account and verify it is removed"() {
        given:
        "delete the created service account"
        orchestrator.deleteServiceAccount(NEW_SA)

        expect:
        "SR should not show the service account"
        ServiceAccountService.waitForServiceAccountRemoved(NEW_SA)
    }

    @Tag("BAT")
    def "Verify scraped roles"() {
        expect:
        "SR should have the same roles"

        withRetry(20, 5) {
            def stackroxRoles = RbacService.getRoles()
            def orchestratorRoles = orchestrator.getRoles() + orchestrator.getClusterRoles()

            assert stackroxRoles.size() == orchestratorRoles.size()
            for (Rbac.K8sRole stackroxRole : stackroxRoles) {
                log.info "Looking for SR Role: ${stackroxRole.name} (${stackroxRole.namespace})"
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
                    def found = false
                    K8sPolicyRule oRule = role.rules[i]
                    for (int j = 0; j < stackroxRole.rulesList.size(); j++) {
                        Rbac.PolicyRule sRule = stackroxRole.rulesList[i]
                        if (oRule.verbs != sRule.verbsList) { continue }
                        if (oRule.apiGroups != sRule.apiGroupsList) { continue }
                        if (oRule.resources != sRule.resourcesList) { continue }
                        if (oRule.nonResourceUrls != sRule.nonResourceUrlsList) { continue }
                        if (oRule.resourceNames != sRule.resourceNamesList) { continue }
                        found = true
                    }
                    assert found
                }
                assert RbacService.getRole(stackroxRole.id) == stackroxRole
            }
        }
    }

    @Tag("BAT")
    def "Add Role and verify it gets scraped"() {
        given:
        "create a new role"
        orchestrator.createRole(NEW_ROLE)

        expect:
        "SR should detect the new role"
        RbacService.waitForRole(NEW_ROLE)
    }

    @Tag("BAT")
    def "Remove Role and verify it is removed"() {
        given:
        "delete the created role"
        orchestrator.deleteRole(NEW_ROLE)

        expect:
        "SR should not show the role"
        RbacService.waitForRoleRemoved(NEW_ROLE)
    }

    @Tag("BAT")
    def "Add Cluster Role and verify it gets scraped"() {
        given:
        "create a new cluster role"
        orchestrator.createClusterRole(NEW_CLUSTER_ROLE)

        expect:
        "SR should detect the new cluster role"
        RbacService.waitForRole(NEW_CLUSTER_ROLE)
    }

    @Tag("BAT")
    def "Remove Cluster Role and verify it is removed"() {
        given:
        "delete the created cluster role"
        orchestrator.deleteClusterRole(NEW_CLUSTER_ROLE)

        expect:
        "SR should not show the cluster role"
        RbacService.waitForRoleRemoved(NEW_CLUSTER_ROLE)
    }

    @Tag("BAT")
    def "Verify scraped bindings"() {
        expect:
        "SR should have the same bindings"
        withRetry(45, 2) {
            def stackroxBindings = RbacService.getRoleBindings()
            def orchestratorBindings = orchestrator.getRoleBindings() + orchestrator.getClusterRoleBindings()

            def stackroxBindingsSet = stackroxBindings.collect { "${it.namespace}/${it.name}" }
            def orchestratorBindingsSet = orchestratorBindings.collect { "${it.namespace}/${it.name}" }
            assert stackroxBindingsSet.toSet() == orchestratorBindingsSet.toSet()

            for (Rbac.K8sRoleBinding b : stackroxBindings) {
                K8sRoleBinding binding = orchestratorBindings.find {
                    it.name == b.name && it.namespace == b.namespace
                }
                assert binding != null

                binding.annotations.remove("kubectl.kubernetes.io/last-applied-configuration")
                assert b.labelsMap == binding.labels
                assert b.annotationsMap == binding.annotations
                assert b.roleId == binding.roleRef.uid
                assert b.subjectsCount == binding.subjects.size()

                for (int i = 0; i < binding.subjects.size(); i++) {
                    K8sSubject oSubject = binding.subjects[i]
                    Rbac.Subject sSubject = b.subjectsList[i]
                    assert sSubject.name == oSubject.name
                    assert sSubject.namespace == ( oSubject.namespace ?:"" )
                    assert CaseFormat.UPPER_UNDERSCORE.to(CaseFormat.UPPER_CAMEL, sSubject.kind.toString()) ==
                            oSubject.kind
                }
            }
        }
    }

    @Tag("BAT")
    def "Verify returned subject list is complete"() {
        given:
        List<K8sSubject> orchestratorSubjects = null
        List<RbacServiceOuterClass.SubjectAndRoles> stackroxSubjectsAndRoles = null

        expect:
        "SR should have the same subjects"
        withRetry(15, 2) {
            stackroxSubjectsAndRoles = RbacService.getSubjects()
            // list of bindings from the orchestrator, we will pull unique subjects from this list
            orchestratorSubjects = fetchOrchestratorSubjects()
            assert stackroxSubjectsAndRoles.size() == orchestratorSubjects.size()
        }
        def stackroxSubjects = stackroxSubjectsAndRoles*.subject

        assert stackroxSubjects.size() == orchestratorSubjects.size()
        for (Rbac.Subject sub : stackroxSubjects) {
            K8sSubject subject = orchestratorSubjects.find {
                it.name == sub.name &&
                        it.namespace == sub.namespace &&
                        it.kind.toLowerCase() == sub.kind.toString().toLowerCase()
            }
            assert subject
        }
    }

    @Tag("BAT")
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

    @Tag("BAT")
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

    @Tag("BAT")
    def "Remove Binding with cluster role ref and verify it is removed"() {
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

    @Tag("BAT")
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

    @Tag("BAT")
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
