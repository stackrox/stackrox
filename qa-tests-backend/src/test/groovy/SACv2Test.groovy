import static io.stackrox.proto.storage.RoleOuterClass.Access.NO_ACCESS
import static io.stackrox.proto.storage.RoleOuterClass.Access.READ_ACCESS
import static io.stackrox.proto.storage.RoleOuterClass.Access.READ_WRITE_ACCESS
import static io.stackrox.proto.storage.RoleOuterClass.SimpleAccessScope.Rules
import static io.stackrox.proto.storage.RoleOuterClass.SimpleAccessScope.newBuilder
import static services.ApiTokenService.generateToken
import static services.ClusterService.DEFAULT_CLUSTER_NAME
import groups.BAT
import io.stackrox.proto.api.v1.ApiTokenService
import io.stackrox.proto.storage.RoleOuterClass
import org.junit.experimental.categories.Category
import services.BaseService
import services.DeploymentService
import services.RoleService
import spock.lang.IgnoreIf
import spock.lang.Shared
import util.Env

@Category(BAT)
@IgnoreIf({ Env.CI_JOBNAME.contains("postgres") })
class SACv2Test extends SACTest {

    @Shared
    private Map<String, RoleOuterClass.Access> allResourcesAccess

    @Shared
    private Map<String, List<String>> tokenToRoles

    def setupSpec() {
        disableAuthzPlugin()

        allResourcesAccess = RoleService.resources.resourcesList.collectEntries { [it, READ_WRITE_ACCESS] }

        // TODO: Replace with the defaultAccessScope id: "denyall"
        def noaccessScope = RoleService.createAccessScope(newBuilder()
                .setName("no-access-scope").build())
        def remoteQaTest1 = createAccessScope(DEFAULT_CLUSTER_NAME, "qa-test1")
        def remoteQaTest2 = createAccessScope(DEFAULT_CLUSTER_NAME, "qa-test2")

        def noaccess = createRole(noaccessScope.id, allResourcesAccess)

        tokenToRoles = [
                (NOACCESSTOKEN)                   : [noaccess],
                (ALLACCESSTOKEN)                  : [createRole(UNRESTRICTED_SCOPE_ID, allResourcesAccess)],
                "deployments-access-token"        : [createRole(remoteQaTest2.id,
                        ["Deployment": READ_ACCESS, "Risk": READ_ACCESS])],
                "getSummaryCountsToken"           : [createRole(remoteQaTest1.id, allResourcesAccess)],
                "listSecretsToken"                : [createRole(UNRESTRICTED_SCOPE_ID, ["Secret": READ_ACCESS])],
                "searchAlertsToken"               : [createRole(remoteQaTest1.id, ["Alert": READ_ACCESS]), noaccess],
                "searchDeploymentsToken"          : [createRole(remoteQaTest1.id,
                        ["Deployment": READ_ACCESS]), noaccess],
                "searchImagesToken"               : [createRole(remoteQaTest1.id, ["Image": READ_ACCESS]), noaccess],
                "searchNamespacesToken"           : [createRole(remoteQaTest1.id,
                        ["Namespace": READ_ACCESS]), noaccess],
                "searchDeploymentsImagesToken"    : [createRole(remoteQaTest1.id,
                        ["Deployment": READ_ACCESS, "Image": READ_ACCESS]), noaccess],
                "stackroxNetFlowsToken"           : [createRole(createAccessScope(DEFAULT_CLUSTER_NAME, "stackrox").id,
                        ["Deployment": READ_ACCESS, "NetworkGraph": READ_ACCESS]),
                                                     createRole(UNRESTRICTED_SCOPE_ID, ["Cluster": READ_ACCESS]),
                                                     noaccess],
                "kubeSystemDeploymentsImagesToken": [createRole(createAccessScope(
                        DEFAULT_CLUSTER_NAME, "kube-system").id, ["Deployment": READ_ACCESS, "Image": READ_ACCESS]),
                                                     noaccess],
                "aggregatedToken"                 : [createRole(remoteQaTest2.id, ["Deployment": READ_ACCESS]),
                                                     createRole(remoteQaTest1.id, ["Deployment": NO_ACCESS]),
                                                     noaccess],
        ]
    }

    def cleanupSpec() {
        cleanupRole(*(tokenToRoles.values().flatten().unique()))
    }

    @Override
    ApiTokenService.GenerateTokenResponse useToken(String tokenName) {
        ApiTokenService.GenerateTokenResponse token = generateToken(tokenName, *(tokenToRoles.get(tokenName)))
        BaseService.useApiToken(token.token)
        token
    }

    @Override
    Boolean summaryTestShouldSeeNoClustersAndNodes() { false }

    def cleanupRole(String... roleName) {
        roleName.each {
            try {
                def role = RoleService.getRole(it)
                RoleService.deleteRole(role.name)
                RoleService.deleteAccessScope(role.accessScopeId)
            } catch (Exception e) {
                log.error("Error deleting role ${name}", e)
            }
        }
    }

    String createRole(String sacId, Map<String, RoleOuterClass.Access> resources) {
        RoleService.createRoleWithScopeAndPermissionSet("SACv2 Test Automation Role " + UUID.randomUUID(),
                sacId, resources
        ).name
    }

    def createAccessScope(String clusterName, String namespaceName) {
        RoleService.createAccessScope(newBuilder()
                .setName(UUID.randomUUID().toString())
                .setRules(Rules.newBuilder()
                        .addIncludedNamespaces(Rules.Namespace.newBuilder()
                                .setClusterName(clusterName)
                                .setNamespaceName(namespaceName)))
                .build())
    }

    @IgnoreIf({ Env.CI_JOBNAME.contains("postgres") })
    def "test role aggregation should not combine permissions sets"() {
        when:
        useToken("aggregatedToken")

        then:
        def result = DeploymentService.listDeployments()
        assert result.find { it.name == DEPLOYMENT_QA2.name }
        assert !result.find { it.name == DEPLOYMENT_QA1.name }
    }

}
